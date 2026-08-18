// Package jobs ports apps/server/src/resources/jobs.ts: fetch/sync job
// execution, plus the Executor interface used by admin routes.
package jobs

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/providers"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/scraper"
	"github.com/Mogvl/dmbt-web/server/internal/subjects"
)

// Executor runs fetch/sync jobs, single-flight per provider.
type Executor struct {
	Store     *resources.Store
	Providers *providers.Registry
	Subjects  *subjects.Module
	Push      Pusher

	mu      sync.Mutex
	running map[string]string // provider -> 'fetch' | 'sync'
}

// Pusher is the optional telegram push integration.
type Pusher interface {
	// EnqueueResourceMessages queues the given resource ids for pushing.
	EnqueueResourceMessages(ids []int64)
	// EnqueueFailedResourceMessages requeues failed messages.
	EnqueueFailedResourceMessages()
	// NotifyNewResources is called after a fetch job when something changed.
	NotifyNewResources(inserted []resources.NotifiedResource)
}

type noopPusher struct{}

func (noopPusher) EnqueueResourceMessages([]int64)         {}
func (noopPusher) EnqueueFailedResourceMessages()          {}
func (noopPusher) NotifyNewResources([]resources.NotifiedResource) {}

// NewExecutor creates the job executor with an optional pusher.
func NewExecutor(store *resources.Store, reg *providers.Registry, subs *subjects.Module, pusher Pusher) *Executor {
	if pusher == nil {
		pusher = noopPusher{}
	}
	return &Executor{Store: store, Providers: reg, Subjects: subs, Push: pusher, running: map[string]string{}}
}

// matchSubject adapts the subjects module to the upsert callback.
func (e *Executor) matchSubject(titleAlt string) *int64 {
	return e.Subjects.MatchActiveSubject(titleAlt)
}

// FetchProvider ports the resources.fetch RPC handler: ack immediately,
// run fetch job in background.
func (e *Executor) FetchProvider(providerName string) (any, int) {
	e.mu.Lock()
	if mode, ok := e.running[providerName]; ok {
		e.mu.Unlock()
		return map[string]any{
			"status":   "OK",
			"mode":     "already_running",
			"job":      mode,
			"provider": providerName,
		}, 200
	}
	e.running[providerName] = "fetch"
	e.mu.Unlock()

	go func() {
		defer e.clearRunning(providerName)
		if _, err := e.RunFetchJob(providerName); err != nil {
			log.Printf("fetch job %s failed: %v", providerName, err)
		}
	}()

	return map[string]any{
		"status":   "OK",
		"mode":     "queued",
		"job":      "fetch",
		"provider": providerName,
	}, 202
}

// SyncProvider ports the resources.sync RPC handler.
func (e *Executor) SyncProvider(providerName string, start, end int) (any, int) {
	e.mu.Lock()
	if mode, ok := e.running[providerName]; ok {
		e.mu.Unlock()
		return map[string]any{
			"status":   "OK",
			"mode":     "already_running",
			"job":      mode,
			"provider": providerName,
		}, 200
	}
	e.running[providerName] = "sync"
	e.mu.Unlock()

	go func() {
		defer e.clearRunning(providerName)
		if _, err := e.RunSyncJob(providerName, start, end); err != nil {
			log.Printf("sync job %s failed: %v", providerName, err)
		}
	}()

	return map[string]any{
		"status":   "OK",
		"mode":     "queued",
		"job":      "sync",
		"provider": providerName,
	}, 202
}

func (e *Executor) clearRunning(provider string) {
	e.mu.Lock()
	delete(e.running, provider)
	e.mu.Unlock()
}

// RunFetchJob ports runFetchJob.
func (e *Executor) RunFetchJob(providerName string) (*resources.UpsertResult, error) {
	provider, ok := e.Providers.Get(providerName)
	if !ok {
		return nil, nil
	}

	fetchedAt := time.Now().UTC()
	newResources, err := provider.FetchLatest(5)
	if err != nil {
		if scraper.IsNetworkError(err) {
			log.Printf("marking provider %s inactive (network error)", providerName)
		}
		return nil, err
	}
	log.Printf("fetch %s: walked %d new resources, upserting...", providerName, len(newResources))

	converted := make([]resources.NewResource, 0, len(newResources))
	for _, r := range newResources {
		converted = append(converted, resources.ToNewResource(r, &fetchedAt))
	}
	// sort by createdAt ascending
	sort.Slice(converted, func(i, j int) bool {
		return converted[i].CreatedAt.Before(converted[j].CreatedAt)
	})

	upsert, err := e.Store.UpsertResources(converted, true, e.matchSubject)
	if err != nil {
		return nil, err
	}

	attached, detached, err := e.Store.MaintainDuplicatedResources(upsert.Changed)
	if err != nil {
		return nil, err
	}

	changed := len(upsert.Inserted)+len(upsert.Updated) > 0 || len(attached)+len(detached) > 0
	if changed {
		e.updateRefreshTimestamp(providerName, fetchedAt)
		e.Push.NotifyNewResources(upsert.Inserted)
		e.Push.EnqueueResourceMessages(idsOf(upsert.Inserted))
	}
	e.Push.EnqueueFailedResourceMessages()

	return upsert, nil
}

// RunSyncJob ports runSyncJob: no telegram push.
func (e *Executor) RunSyncJob(providerName string, start, end int) (*resources.UpsertResult, error) {
	provider, ok := e.Providers.Get(providerName)
	if !ok {
		return nil, nil
	}

	fetchedAt := time.Now().UTC()
	pageResources, err := provider.FetchPages(start, end)
	if err != nil {
		return nil, err
	}

	converted := make([]resources.NewResource, 0, len(pageResources))
	for _, r := range pageResources {
		converted = append(converted, resources.ToNewResource(r, &fetchedAt))
	}

	upsert, err := e.Store.UpsertResources(converted, true, e.matchSubject)
	if err != nil {
		return nil, err
	}

	deleted, err := e.Store.MarkDeletedResources(providerName, converted)
	if err != nil {
		return nil, err
	}

	changedIDs := append(append([]int64{}, upsert.Changed...), deleted...)
	attached, detached, err := e.Store.MaintainDuplicatedResources(changedIDs)
	if err != nil {
		return nil, err
	}

	changed := len(upsert.Inserted)+len(upsert.Updated)+len(deleted) > 0 || len(attached)+len(detached) > 0
	if changed {
		e.updateRefreshTimestamp(providerName, fetchedAt)
	}
	return upsert, nil
}

// FetchAllProviders ports /admin/providers: refresh all providers.
func (e *Executor) FetchAllProviders() (map[string]*model.Provider, error) {
	out := map[string]*model.Provider{}
	for _, p := range e.Providers.All() {
		if _, err := e.RunFetchJob(p.ID); err != nil {
			log.Printf("admin refresh %s failed: %v", p.ID, err)
		}
	}
	for _, p := range e.Providers.All() {
		out[p.ID] = &model.Provider{
			ID:          p.ID,
			Name:        p.Name,
			RefreshedAt: time.Now().UTC(),
			IsActive:    true,
		}
	}
	return out, nil
}

func (e *Executor) updateRefreshTimestamp(provider string, t time.Time) {
	e.Store.DB.Exec("UPDATE providers SET refreshed_at = ? WHERE id = ?", db.FormatTime(t), provider)
}

func idsOf(notified []resources.NotifiedResource) []int64 {
	out := make([]int64, 0, len(notified))
	for _, n := range notified {
		out = append(out, n.ID)
	}
	return out
}

// UpdateCalendar wraps the subjects calendar sync and enqueues push ids.
func (e *Executor) UpdateCalendar() error {
	indexFn := func(sub subjects.Subject, overwrite bool) ([]int64, error) {
		ids, err := e.Subjects.IndexSubject(sub, overwrite, 30)
		if err == nil && len(ids) > 0 {
			e.Push.EnqueueResourceMessages(ids)
		}
		return ids, err
	}
	return e.Subjects.UpdateCalendar(indexFn)
}
