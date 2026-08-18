// Package providers: registry + provider state (mirrors
// apps/server/src/providers/module.ts isActive/refreshedAt tracking).
package providers

import (
	"database/sql"
	"sync"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/model"
)

// State tracks the providers table in memory and persists changes,
// mirroring ProvidersModule.updateRefreshTimestamp / updateActiveStatus.
type State struct {
	mu        sync.RWMutex
	db        *sql.DB
	providers map[string]*model.Provider
}

// LoadState reads the providers table.
func LoadState(db *sql.DB) (*State, error) {
	s := &State{db: db, providers: map[string]*model.Provider{}}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload re-reads the providers table.
func (s *State) Reload() error {
	rows, err := s.db.Query("SELECT id, name, refreshed_at, is_active FROM providers")
	if err != nil {
		return err
	}
	defer rows.Close()
	next := map[string]*model.Provider{}
	for rows.Next() {
		var id, name, refreshedAt string
		var isActive int
		if err := rows.Scan(&id, &name, &refreshedAt, &isActive); err != nil {
			return err
		}
		t, _ := db.ParseTime(refreshedAt)
		p := &model.Provider{
			ID:          id,
			Name:        name,
			RefreshedAt: t,
			IsActive:    isActive != 0,
		}
		next[id] = p
	}
	if err := rows.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.providers = next
	s.mu.Unlock()
	return nil
}

// Timestamp mirrors ProvidersModule.timestamp: max refreshedAt.
func (s *State) Timestamp() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var max time.Time
	for _, p := range s.providers {
		if p.RefreshedAt.After(max) {
			max = p.RefreshedAt
		}
	}
	return max
}

// Get returns a provider state snapshot (or nil).
func (s *State) Get(id string) *model.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.providers[id]
	if !ok {
		return nil
	}
	copy := *p
	return &copy
}

// All returns all provider states in SupportProviders order.
func (s *State) All() []*model.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Provider, 0, len(SupportProviders))
	for _, id := range SupportProviders {
		if p, ok := s.providers[id]; ok {
			copy := *p
			out = append(out, &copy)
		}
	}
	return out
}

// UpdateRefreshTimestamp mirrors updateRefreshTimestamp: reactivates and
// stamps the refresh time.
func (s *State) UpdateRefreshTimestamp(provider string, t time.Time) {
	s.db.Exec("UPDATE providers SET is_active = 1, refreshed_at = ? WHERE id = ?", db.FormatTime(t), provider)
	s.Reload()
}

// UpdateActiveStatus mirrors updateActiveStatus.
func (s *State) UpdateActiveStatus(provider string, isActive bool) {
	v := 0
	if isActive {
		v = 1
	}
	s.db.Exec("UPDATE providers SET is_active = ? WHERE id = ?", v, provider)
	s.Reload()
}
