// One-off maintenance: patch empty mikan magnets from the upstream mirror.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Mogvl/dmbt-web/server/internal/scraper"
)

func main() {
	db, err := sql.Open("sqlite", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	total := 0
	for page := 1; page <= 200; page++ {
		if page > 1 {
			time.Sleep(600 * time.Millisecond)
		}
		rows, err := scraper.FetchMikanPage(page, 5)
		if err != nil {
			log.Printf("page %d error: %v", page, err)
			continue
		}
		updated := 0
		for _, r := range rows {
			if r.Magnet == "" {
				continue
			}
			res, err := db.Exec(`UPDATE resources SET magnet = ?, tracker = ? WHERE provider_name = 'mikan' AND provider_id = ? AND (magnet IS NULL OR magnet = '')`,
				r.Magnet, r.Tracker, r.ProviderID)
			if err != nil {
				log.Printf("update error: %v", err)
				continue
			}
			if n, _ := res.RowsAffected(); n > 0 {
				updated++
			}
		}
		total += updated
		if updated > 0 || page%20 == 0 {
			log.Printf("page %d: patched %d magnets (total %d)", page, updated, total)
		}
		if updated == 0 && page > 5 {
			// the mirror's newest pages always have magnets; 0 updates
			// across several pages means we're done
			break
		}
	}
	fmt.Printf("done, total patched: %d\n", total)
}
