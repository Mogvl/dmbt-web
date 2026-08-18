// dmbt-web server — AnimeGarden-compatible API server (port 9701).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/api"
	"github.com/Mogvl/dmbt-web/server/internal/collections"
	"github.com/Mogvl/dmbt-web/server/internal/config"
	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/jobs"
	"github.com/Mogvl/dmbt-web/server/internal/providers"
	"github.com/Mogvl/dmbt-web/server/internal/push"
	"github.com/Mogvl/dmbt-web/server/internal/query"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/scraper"
	"github.com/Mogvl/dmbt-web/server/internal/search"
	"github.com/Mogvl/dmbt-web/server/internal/subjects"
)

func main() {
	cfg := config.Load()

	tok, err := search.NewTokenizer()
	if err != nil {
		log.Fatalf("failed to init tokenizer: %v", err)
	}

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlDB.Close()

	store := resources.New(sqlDB, tok)
	if err := store.RefreshParties(); err != nil {
		log.Fatalf("failed to load parties: %v", err)
	}
	users, teams, err := store.LoadUserBriefs()
	if err != nil {
		log.Fatalf("failed to load user briefs: %v", err)
	}

	subs := &subjects.Module{DB: sqlDB}
	if err := subs.Load(); err != nil {
		log.Fatalf("failed to load subjects: %v", err)
	}

	// wire the anipar alias rewriter for ANi titles
	scraper.SetAliasRewriter(func(title string) string {
		return title
	})

	q := query.New(&query.Store{
		DB:          sqlDB,
		NameToUserID: store.NameToUserID,
		NameToTeamID: store.NameToTeamID,
		UserIDToName: store.UserIDToName,
		TeamIDToName: store.TeamIDToName,
	}, tok)

	sys := &api.System{
		Cfg:         cfg,
		DB:          sqlDB,
		Store:       store,
		Query:       q,
		Subjects:    subs,
		Collections: &collections.Store{DB: sqlDB},
		Users:       users,
		Teams:       teams,
		Timestamp:   time.Now().UTC(),
		AdminSecret: cfg.AdminToken,
		Site:        cfg.PublicURLHost(),
	}
	if sys.AdminSecret == "" {
		sys.AdminSecret = "dmbt-web-admin"
	}

	reg := providers.NewRegistry(sys)
	sys.Providers = reg

	// telegram push (optional)
	var pusher push.Pusher = nil
	if cfg.TelegramBotToken != "" {
		pusher = push.New(sqlDB, store, subs, cfg.TelegramBotToken, cfg.TelegramChannelID)
	}

	executor := jobs.NewExecutor(store, reg, subs, pusher)
	sys.Executor = executor

	// schedule cron jobs (every 5 min fetch, hourly sync, hourly calendar)
	if cfg.Cron {
		go scheduler(executor, reg)
	}

	handler := api.NewServer(sys)

	addr := ":" + itoa(cfg.Port)
	log.Printf("dmbt-web server listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: handler}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func itoa(n int) string {
	if n == 0 {
		return "9701"
	}
	return fmtInt(n)
}

func fmtInt(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// scheduler runs the cron jobs in the Asia/Shanghai timezone:
// fetch every 5 minutes, sync at the top of each hour, calendar at :17.
func scheduler(executor *jobs.Executor, reg *providers.Registry) {
	runFetch := func() {
		for _, id := range reg.IDs() {
			executor.FetchProvider(id)
		}
	}
	runSync := func() {
		for _, id := range reg.IDs() {
			executor.SyncProvider(id, 1, 10)
		}
	}
	runCalendar := func() {
		if err := executor.UpdateCalendar(); err != nil {
			log.Printf("calendar sync failed: %v", err)
		}
	}

	shanghai := time.FixedZone("Asia/Shanghai", 8*60*60)

	// delayUntil fires f when the Shanghai clock reaches the next
	// occurrence of minute (0..59) strictly after now.
	delayUntil := func(minute int, f func()) {
		now := time.Now().In(shanghai)
		candidate := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), minute, 0, 0, shanghai)
		if !candidate.After(now) {
			candidate = candidate.Add(time.Hour)
		}
		time.AfterFunc(time.Until(candidate), f)
	}

	// initial runs shortly after boot
	time.AfterFunc(10*time.Second, runFetch)
	time.AfterFunc(30*time.Second, runCalendar)

	// fetch: every 5 minutes (minute % 5 == 0)
	{
		minute := (time.Now().In(shanghai).Minute()/5 + 1) * 5
		if minute >= 60 {
			minute = 0
		}
		delayUntil(minute, func() {
			runFetch()
			ticker := time.NewTicker(5 * time.Minute)
			for range ticker.C {
				runFetch()
			}
		})
	}

	// sync: top of every hour
	delayUntil(0, func() {
		runSync()
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			runSync()
		}
	})

	// calendar: at :17 every hour
	delayUntil(17, func() {
		runCalendar()
		ticker := time.NewTicker(time.Hour)
		for range ticker.C {
			runCalendar()
		}
	})

	select {}
}
