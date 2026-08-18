// Package push ports apps/server/src/push: optional Telegram bot pushes for
// new anime resources. All operations no-op when not configured.
package push

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/anipar"
	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/subjects"
)

// Push implements the jobs.Pusher interface with a Telegram bot.
type Push struct {
	DB      *sql.DB
	Store   *resources.Store
	Subjects *subjects.Module
	Token   string
	ChatID  string
	// pendingResourceIds dedupes per resource id in-process.
	pending map[int64]bool
}

// Pusher is the interface consumed by the jobs module.
type Pusher interface {
	EnqueueResourceMessages(ids []int64)
	EnqueueFailedResourceMessages()
	NotifyNewResources(inserted []resources.NotifiedResource)
}

// New builds the push module; returns nil-safe usage via the interface.
func New(db *sql.DB, store *resources.Store, subs *subjects.Module, token, chatID string) *Push {
	return &Push{DB: db, Store: store, Subjects: subs, Token: token, ChatID: chatID, pending: map[int64]bool{}}
}

func (p *Push) configured() bool {
	return p != nil && p.Token != "" && p.ChatID != ""
}

// EnqueueResourceMessages ports the fire-and-forget enqueue.
func (p *Push) EnqueueResourceMessages(ids []int64) {
	if !p.configured() {
		return
	}
	for _, id := range ids {
		if p.pending[id] {
			continue
		}
		p.pending[id] = true
		go p.sendResourceMessage(id)
	}
}

// EnqueueFailedResourceMessages ports the failure compensation.
func (p *Push) EnqueueFailedResourceMessages() {
	if !p.configured() {
		return
	}
	// 1. failed within 7 days
	rows, err := p.DB.Query(`SELECT DISTINCT resource_id FROM telegram_messages
		WHERE status = -1 AND updated_at >= ?`, db.FormatTime(time.Now().UTC().Add(-7*24*time.Hour)))
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	// 2. stale Pending/Sending (>= 6h) -> Failed, returning resource ids
	res, err := p.DB.Exec(`UPDATE telegram_messages SET status = -1
		WHERE status IN (0, 1) AND updated_at <= ?`, db.FormatTime(time.Now().UTC().Add(-6*time.Hour)))
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			rows2, err := p.DB.Query(`SELECT resource_id FROM telegram_messages WHERE status = -1 AND updated_at >= ?`,
				db.FormatTime(time.Now().UTC().Add(-7*24*time.Hour)))
			if err == nil {
				for rows2.Next() {
					var id int64
					if err := rows2.Scan(&id); err == nil {
						ids = append(ids, id)
					}
				}
				rows2.Close()
			}
		}
	}

	dedup := map[int64]bool{}
	for _, id := range ids {
		if !dedup[id] {
			dedup[id] = true
			if !p.pending[id] {
				p.pending[id] = true
				go p.sendResourceMessage(id)
			}
		}
	}
}

// NotifyNewResources is a no-op hook (the original publishes Redis
// notifications; in-process the query caches are per-request so nothing
// needs invalidating).
func (p *Push) NotifyNewResources(inserted []resources.NotifiedResource) {}

// sendResourceMessage ports PushContext.prepare + send logic (simplified to
// the core: guard checks, dedup, telegram sendPhoto with the caption).
func (p *Push) sendResourceMessage(resourceID int64) {
	defer delete(p.pending, resourceID)

	row, err := p.Store.GetByID(resourceID)
	if err != nil || row == nil || row.IsDeleted {
		return
	}

	// shouldSendTypeResource
	if row.Type != "动画" {
		return
	}

	fansubName := ""
	if row.FansubID.Valid {
		fansubName = p.Store.TeamIDToName[row.FansubID.Int64]
	}
	if fansubName == "" {
		fansubName = p.Store.UserIDToName[row.PublisherID]
	}
	if !allowlist(fansubName) {
		return
	}
	if !row.SubjectID.Valid {
		return
	}

	// subject display info from the bgm mirror (alias.zh[0], poster, onair)
	subject := p.fetchSubjectCard(row.SubjectID.Int64)

	// anipar parse for the caption structure (episode/subtitle/format)
	parsed := anipar.Parse(row.Title, fansubName)

	photo, caption := buildResourceCardMessage(
		row.Title,
		row.CreatedAt,
		row.Magnet,
		row.Size,
		row.Provider,
		row.ProviderID,
		fansubName,
		parsed,
		subject,
	)

	// sendPhoto
	body := url.Values{}
	body.Set("chat_id", p.ChatID)
	body.Set("caption", caption)
	body.Set("parse_mode", "HTML")
	body.Set("photo", photo)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", p.Token),
		strings.NewReader(body.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var payload struct {
		OK          bool `json:"ok"`
		Description string `json:"description"`
		Result      struct {
			MessageID int64 `json:"message_id"`
			Chat      struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return
	}

	now := db.Now()
	if payload.OK {
		// upsert telegram_messages (publisher_id, subject_id, episode)
		episode := fmt.Sprintf("resource:%d", row.ID)
		p.DB.Exec(`INSERT INTO telegram_messages
			(resource_id, publisher_id, fansub_id, subject_id, episode, telegram_chat_id, telegram_message_id, status, sent_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 2, ?, ?)
			ON CONFLICT(publisher_id, subject_id, episode) DO UPDATE SET
				resource_id = excluded.resource_id, status = 2, sent_at = excluded.sent_at, updated_at = excluded.updated_at,
				telegram_chat_id = excluded.telegram_chat_id, telegram_message_id = excluded.telegram_message_id`,
			row.ID, row.PublisherID, sql.NullInt64{Int64: row.FansubID.Int64, Valid: row.FansubID.Valid},
			row.SubjectID.Int64, episode, payload.Result.Chat.ID, payload.Result.MessageID, now, now)
	} else {
		log.Printf("telegram push failed: %s", payload.Description)
		p.DB.Exec(`UPDATE telegram_messages SET status = -1, updated_at = ? WHERE resource_id = ?`, now, row.ID)
	}
}

var allowlistSet = map[string]bool{
	"ANi": true, "LoliHouse": true, "绿茶字幕组": true, "桜都字幕组": true,
	"Prejudice-Studio": true, "喵萌奶茶屋": true, "雪飄工作室(FLsnow)": true,
	"三明治摆烂组": true,
}

func allowlist(name string) bool {
	return allowlistSet[name]
}

func htmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	).Replace(s)
}

// fetchSubjectCard loads the bgm subject metadata used by the caption.
func (p *Push) fetchSubjectCard(subjectID int64) *subjectCard {
	card := &subjectCard{}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://bgm.animes.garden/subject/%d", subjectID))
	if err != nil {
		return card
	}
	defer resp.Body.Close()
	var payload struct {
		OK   bool `json:"ok"`
		Data struct {
			Title     string `json:"title"`
			Alias     map[string][]string `json:"alias"`
			OnairDate *string `json:"onair_date"`
			Poster    string `json:"poster"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || !payload.OK {
		return card
	}
	card.Name = payload.Data.Title
	if zhAlias := payload.Data.Alias["zh"]; len(zhAlias) > 0 && zhAlias[0] != "" {
		card.Name = zhAlias[0]
	}
	card.Poster = payload.Data.Poster
	if payload.Data.OnairDate != nil {
		card.OnairDate = *payload.Data.OnairDate
	}
	return card
}

var _ = bytes.NewBuffer
var _ = regexp.MustCompile
