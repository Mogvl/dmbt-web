// Package resources ports apps/server/src/resources: upsert, duplicate
// maintenance, details management, import pipeline.
package resources

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/anipar"
	"github.com/Mogvl/dmbt-web/server/internal/db"
	"github.com/Mogvl/dmbt-web/server/internal/model"
	"github.com/Mogvl/dmbt-web/server/internal/scraper"
	"github.com/Mogvl/dmbt-web/server/internal/search"
)

// Store wraps the DB and the name<->id maps.
type Store struct {
	DB *sql.DB

	NameToUserID map[string]int64
	NameToTeamID map[string]int64
	UserIDToName map[int64]string
	TeamIDToName map[int64]string

	Tok *search.Tokenizer
}

// New creates a Store with empty party maps.
func New(db *sql.DB, tok *search.Tokenizer) *Store {
	return &Store{DB: db, NameToUserID: map[string]int64{}, NameToTeamID: map[string]int64{}, UserIDToName: map[int64]string{}, TeamIDToName: map[int64]string{}, Tok: tok}
}

// RefreshParties reloads the name<->id maps from the DB.
func (s *Store) RefreshParties() error {
	s.NameToUserID = map[string]int64{}
	s.UserIDToName = map[int64]string{}
	rows, err := s.DB.Query("SELECT id, name FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		s.NameToUserID[name] = id
		s.UserIDToName[id] = name
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.NameToTeamID = map[string]int64{}
	s.TeamIDToName = map[int64]string{}
	rows, err = s.DB.Query("SELECT id, name FROM teams")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		s.NameToTeamID[name] = id
		s.TeamIDToName[id] = name
	}
	return rows.Err()
}

// LoadUserBriefs loads the /users and /teams API lists.
func (s *Store) LoadUserBriefs() (users, teams []model.UserBrief, err error) {
	rows, err := s.DB.Query("SELECT id, name, avatar FROM users ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u model.UserBrief
		var avatar sql.NullString
		if err := rows.Scan(&u.ID, &u.Name, &avatar); err != nil {
			return nil, nil, err
		}
		if avatar.Valid {
			u.Avatar = avatar.String
		}
		users = append(users, u)
	}
	rows2, err := s.DB.Query("SELECT id, name, avatar FROM teams ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var u model.UserBrief
		var avatar sql.NullString
		if err := rows2.Scan(&u.ID, &u.Name, &avatar); err != nil {
			return nil, nil, err
		}
		if avatar.Valid {
			u.Avatar = avatar.String
		}
		teams = append(teams, u)
	}
	return users, teams, nil
}

// NewResource mirrors NewResource in the original.
type NewResource struct {
	Provider   string
	ProviderID string
	Title      string
	Href       string
	Type       string
	Magnet     string
	Tracker    string
	Size       string
	CreatedAt  time.Time
	FetchedAt  *time.Time
	Publisher  *scraper.Party
	Fansub     *scraper.Party
	IsDeleted  bool
}

// ToNewResource ports toNewResource from resources/import.ts.
func ToNewResource(r scraper.ScrapedResource, fetchedAt *time.Time) NewResource {
	return NewResource{
		Provider:   r.Provider,
		ProviderID: r.ProviderID,
		Title:      r.Title,
		Href:       r.Href,
		Type:       r.Type,
		Magnet:     r.Magnet,
		Tracker:    r.Tracker,
		Size:       r.Size,
		CreatedAt:  r.CreatedAt,
		FetchedAt:  fetchedAt,
		Publisher:  r.Publisher,
		Fansub:     r.Fansub,
	}
}

// UpsertResult mirrors upsertResources return.
type UpsertResult struct {
	Inserted []NotifiedResource
	Updated  []NotifiedResource
	Changed  []int64 // ids
	Errors   []string
}

// NotifiedResource mirrors {id, provider, providerId, title}.
type NotifiedResource struct {
	ID         int64  `json:"id"`
	Provider   string `json:"provider"`
	ProviderID string `json:"providerId"`
	Title      string `json:"title"`
}

// DBResourceRow is a parsed resources row.
type DBResourceRow struct {
	ID           int64
	Provider     string
	ProviderID   string
	Title        string
	TitleAlt     string
	TitleSearch  string
	Href         string
	Type         string
	Magnet       string
	Tracker      string
	Size         int64
	CreatedAt    time.Time
	FetchedAt    time.Time
	IndexedAt    time.Time
	PublisherID  int64
	FansubID     sql.NullInt64
	DuplicatedID sql.NullInt64
	SubjectID    sql.NullInt64
	Metadata     string
	IsDeleted    bool
}

const resourceColumns = `id, provider_name, provider_id, title, title_alt, title_search, href, type, magnet, tracker, size, created_at, fetched_at, indexed_at, publisher_id, fansub_id, duplicated_id, subject_id, metadata, is_deleted`

func scanResourceRow(row *sql.Row) (*DBResourceRow, error) {
	var r DBResourceRow
	var createdAt, fetchedAt, indexedAt string
	var isDeleted int
	var metadata sql.NullString
	err := row.Scan(&r.ID, &r.Provider, &r.ProviderID, &r.Title, &r.TitleAlt, &r.TitleSearch,
		&r.Href, &r.Type, &r.Magnet, &r.Tracker, &r.Size, &createdAt, &fetchedAt, &indexedAt,
		&r.PublisherID, &r.FansubID, &r.DuplicatedID, &r.SubjectID, &metadata, &isDeleted)
	if err != nil {
		return nil, err
	}
	r.CreatedAt, _ = db.ParseTime(createdAt)
	r.FetchedAt, _ = db.ParseTime(fetchedAt)
	r.IndexedAt, _ = db.ParseTime(indexedAt)
	r.Metadata = metadata.String
	r.IsDeleted = isDeleted != 0
	return &r, nil
}

// GetByProviderID fetches a resource row by provider + providerId.
func (s *Store) GetByProviderID(provider, providerID string) (*DBResourceRow, error) {
	row := s.DB.QueryRow(
		"SELECT "+resourceColumns+" FROM resources WHERE provider_name = ? AND provider_id = ?",
		provider, providerID)
	r, err := scanResourceRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetResourceHref returns the stored href of a resource, if any (part of the
// providers.System interface).
func (s *Store) GetResourceHref(provider, providerID string) (string, error) {
	var href string
	err := s.DB.QueryRow("SELECT href FROM resources WHERE provider_name = ? AND provider_id = ?", provider, providerID).Scan(&href)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return href, err
}

// ResourceIDsExist returns a set of providerIds already present in the DB
// (part of the providers.System interface).
func (s *Store) ResourceIDsExist(provider string, ids []string) (map[string]bool, error) {
	set := map[string]bool{}
	if len(ids) == 0 {
		return set, nil
	}
	query := "SELECT provider_id FROM resources WHERE provider_name = ? AND provider_id IN (" + placeholders(len(ids)) + ")"
	args := make([]any, 0, len(ids)+1)
	args = append(args, provider)
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, err
		}
		set[pid] = true
	}
	return set, rows.Err()
}

// GetByID fetches a resource row by id.
func (s *Store) GetByID(id int64) (*DBResourceRow, error) {
	row := s.DB.QueryRow("SELECT "+resourceColumns+" FROM resources WHERE id = ?", id)
	r, err := scanResourceRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetByInfoHash finds the latest non-deleted resource with the given btih
// (mirroring the magnet ILIKE query, uppercase-insensitive).
func (s *Store) GetByInfoHash(infoHash string) (*DBResourceRow, error) {
	upper := strings.ToUpper(infoHash)
	row := s.DB.QueryRow(
		"SELECT "+resourceColumns+" FROM resources WHERE UPPER(magnet) LIKE ? AND is_deleted = 0 ORDER BY created_at DESC, id DESC LIMIT 1",
		"magnet:?xt=urn:btih:"+upper+"%")
	r, err := scanResourceRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// ensureParties upserts publishers -> users and fansubs -> teams, mirroring
// ensureParties in resources/index.ts. Returns the resolved ids.
func (s *Store) ensureParties(res NewResource) (publisherID int64, fansubID sql.NullInt64, err error) {
	publisherName := "anonymous"
	avatar := ""
	if res.Publisher != nil {
		publisherName = res.Publisher.Name
		avatar = res.Publisher.Avatar
	}
	if publisherName == "" {
		publisherName = "anonymous"
	}

	if id, ok := s.NameToUserID[publisherName]; ok {
		publisherID = id
		if avatar != "" {
			var existing string
			if err := s.DB.QueryRow("SELECT avatar FROM users WHERE id = ?", id).Scan(&existing); err == nil && existing == "" {
				s.DB.Exec("UPDATE users SET avatar = ? WHERE id = ?", avatar, id)
			}
		}
	} else {
		resp, err := s.DB.Exec("INSERT INTO users (name, avatar, providers) VALUES (?, ?, ?)",
			publisherName, avatar, "{}")
		if err != nil {
			var id2 int64
			if err2 := s.DB.QueryRow("SELECT id FROM users WHERE name = ?", publisherName).Scan(&id2); err2 == nil {
				publisherID = id2
			} else {
				return 0, sql.NullInt64{}, err
			}
		} else {
			publisherID, _ = resp.LastInsertId()
		}
		s.NameToUserID[publisherName] = publisherID
		s.UserIDToName[publisherID] = publisherName
	}

	if res.Fansub != nil && res.Fansub.Name != "" {
		fansubName := res.Fansub.Name
		if id, ok := s.NameToTeamID[fansubName]; ok {
			fansubID = sql.NullInt64{Int64: id, Valid: true}
			if avatar != "" {
				var existing string
				if err := s.DB.QueryRow("SELECT avatar FROM teams WHERE id = ?", id).Scan(&existing); err == nil && existing == "" {
					s.DB.Exec("UPDATE teams SET avatar = ? WHERE id = ?", avatar, id)
				}
			}
		} else {
			resp, err := s.DB.Exec("INSERT INTO teams (name, avatar, providers) VALUES (?, ?, ?)",
				fansubName, avatar, "{}")
			if err != nil {
				var id2 int64
				if err2 := s.DB.QueryRow("SELECT id FROM teams WHERE name = ?", fansubName).Scan(&id2); err2 == nil {
					fansubID = sql.NullInt64{Int64: id2, Valid: true}
				} else {
					return publisherID, sql.NullInt64{}, err
				}
			} else {
				id2, _ := resp.LastInsertId()
				fansubID = sql.NullInt64{Int64: id2, Valid: true}
			}
			s.NameToTeamID[fansubName] = fansubID.Int64
			s.TeamIDToName[fansubID.Int64] = fansubName
		}
	}
	return publisherID, fansubID, nil
}

// transformedResource is the normalized row payload.
type transformedResource struct {
	Provider    string
	ProviderID  string
	Title       string
	TitleAlt    string
	TitleSearch string
	Href        string
	Type        string
	Magnet      string
	Tracker     string
	Size        int64
	CreatedAt   time.Time
	FetchedAt   time.Time
	PublisherID int64
	FansubID    sql.NullInt64
	SubjectID   *int64
	IsDeleted   bool
}

// TransformNewResources ports transformNewResources: validation, titleAlt,
// size, titleSearch, subjectId.
func (s *Store) TransformNewResources(res NewResource, indexSubject bool, matchSubject func(titleAlt string) *int64) (*transformedResource, []string) {
	var errors []string

	supported := false
	for _, p := range []string{"dmhy", "moe", "mikan", "ani"} {
		if res.Provider == p {
			supported = true
			break
		}
	}
	if !supported {
		errors = append(errors, fmt.Sprintf("Unknown provider: %s", res.Provider))
	}

	publisherName := "anonymous"
	if res.Publisher != nil && res.Publisher.Name != "" {
		publisherName = res.Publisher.Name
	}
	fansubName := ""
	if res.Fansub != nil {
		fansubName = res.Fansub.Name
	}

	titleAlt := search.NormalizeTitle(res.Title)
	size := ParseSize(res.Size)
	if _, okU := s.NameToUserID[publisherName]; !okU {
		errors = append(errors, fmt.Sprintf("Unknown publisher: %s", publisherName))
	}
	var fansubID sql.NullInt64
	if fansubName != "" {
		if id, ok := s.NameToTeamID[fansubName]; ok {
			fansubID = sql.NullInt64{Int64: id, Valid: true}
		} else {
			errors = append(errors, fmt.Sprintf("Unknown fansub: %s", fansubName))
		}
	}

	titleSearch := BuildTitleSearch(s.Tok, titleAlt)

	var subjectID *int64
	if indexSubject && matchSubject != nil {
		subjectID = matchSubject(titleAlt)
	}

	fetchedAt := time.Now().UTC()
	if res.FetchedAt != nil {
		fetchedAt = res.FetchedAt.UTC()
	}

	if len(errors) > 0 {
		return nil, errors
	}

	return &transformedResource{
		Provider:    res.Provider,
		ProviderID:  res.ProviderID,
		Title:       res.Title,
		TitleAlt:    titleAlt,
		TitleSearch: titleSearch,
		Href:        res.Href,
		Type:        res.Type,
		Magnet:      res.Magnet,
		Tracker:     res.Tracker,
		Size:        size,
		CreatedAt:   res.CreatedAt,
		FetchedAt:   fetchedAt,
		PublisherID: -1, // replaced by ensureParties result
		FansubID:    fansubID,
		SubjectID:   subjectID,
		IsDeleted:   res.IsDeleted,
	}, nil
}

// BuildTitleSearch ports transform.ts: weight-A tokens from anipar.title (if
// parsed), weight-D tokens from titleAlt; tokens joined with single spaces.
func BuildTitleSearch(tok *search.Tokenizer, titleAlt string) string {
	var tokens []string
	if parsed := anipar.Parse(titleAlt, ""); parsed != nil && parsed.Title != "" {
		for _, t := range tok.Cut(parsed.Title) {
			tokens = append(tokens, strings.ToLower(t))
		}
	}
	for _, t := range tok.Cut(titleAlt) {
		tokens = append(tokens, strings.ToLower(t))
	}
	return strings.Join(tokens, " ")
}

// ParseSize ports parseSize in transform.ts (bytes).
func ParseSize(size string) int64 {
	if size == "" {
		return 0
	}
	entries := []struct {
		re   *regexp.Regexp
		mult float64
	}{
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Kk]i?[Bb]$`), 1},
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Mm]i?[Bb]$`), 1024},
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Gg]i?[Bb]$`), 1024 * 1024},
		{regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*[Tt]i?[Bb]$`), 1024 * 1024 * 1024},
	}
	for _, entry := range entries {
		if m := entry.re.FindStringSubmatch(size); m != nil {
			var f float64
			fmt.Sscanf(m[1], "%f", &f)
			return int64(f * entry.mult)
		}
	}
	var n int64
	if _, err := fmt.Sscanf(size, "%d", &n); err == nil {
		return n
	}
	return 0
}

// UpsertResources ports upsertResources.
func (s *Store) UpsertResources(resources []NewResource, indexSubject bool, matchSubject func(string) *int64) (*UpsertResult, error) {
	result := &UpsertResult{}

	// dedupe by provider:providerId, first wins
	seen := map[string]bool{}
	var unique []NewResource
	for _, r := range resources {
		key := r.Provider + ":" + r.ProviderID
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, r)
	}

	type pendingTransform struct {
		tf *transformedResource
	}
	var pending []pendingTransform

	for _, r := range unique {
		publisherID, fansubID, err := s.ensureParties(r)
		if err != nil {
			return nil, err
		}
		tf, errors := s.TransformNewResources(r, indexSubject, matchSubject)
		if errors != nil {
			result.Errors = append(result.Errors, errors...)
			continue
		}
		tf.PublisherID = publisherID
		tf.FansubID = fansubID
		pending = append(pending, pendingTransform{tf: tf})
	}

	for _, p := range pending {
		tf := p.tf
		existing, err := s.GetByProviderID(tf.Provider, tf.ProviderID)
		if err != nil {
			return nil, err
		}

		if existing == nil {
			resp, err := s.DB.Exec(`INSERT INTO resources
				(provider_name, provider_id, title, title_alt, title_search, href, type, magnet, tracker,
				 size, created_at, fetched_at, indexed_at, publisher_id, fansub_id, duplicated_id, subject_id, metadata, is_deleted)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, '{}', 0)
				ON CONFLICT(provider_name, provider_id) DO NOTHING`,
				tf.Provider, tf.ProviderID, tf.Title, tf.TitleAlt, tf.TitleSearch, tf.Href, tf.Type,
				tf.Magnet, tf.Tracker, tf.Size, db.FormatTime(tf.CreatedAt), db.FormatTime(tf.FetchedAt),
				db.FormatTime(tf.FetchedAt), tf.PublisherID, nullableInt64SQL(tf.FansubID), nullableInt64Ptr(tf.SubjectID))
			if err != nil {
				return nil, err
			}
			if n, _ := resp.RowsAffected(); n > 0 {
				id, _ := resp.LastInsertId()
				result.Inserted = append(result.Inserted, NotifiedResource{ID: id, Provider: tf.Provider, ProviderID: tf.ProviderID, Title: tf.Title})
				result.Changed = append(result.Changed, id)
			}
			continue
		}

		// update when a compared field differs
		updated := existing.IsDeleted != tf.IsDeleted ||
			existing.Href != tf.Href ||
			existing.Magnet != tf.Magnet ||
			existing.Tracker != tf.Tracker ||
			existing.Title != tf.Title ||
			existing.TitleAlt != tf.TitleAlt ||
			existing.TitleSearch != tf.TitleSearch ||
			existing.PublisherID != tf.PublisherID ||
			!equalNullInt64(existing.FansubID, tf.FansubID) ||
			!equalSubjectID(existing.SubjectID, tf.SubjectID)

		if updated {
			subj := existing.SubjectID
			if nv := tf.SubjectID; nv != nil {
				subj = sql.NullInt64{Int64: *nv, Valid: true}
			}
			_, err := s.DB.Exec(`UPDATE resources SET
				is_deleted = ?, href = ?, magnet = ?, tracker = ?, subject_id = ?,
				publisher_id = ?, fansub_id = ?, title = ?, title_alt = ?, title_search = ?,
				fetched_at = ?
				WHERE id = ?`,
				boolInt(tf.IsDeleted), tf.Href, tf.Magnet, tf.Tracker, nullableInt64SQL(subj),
				tf.PublisherID, tf.FansubID, tf.Title, tf.TitleAlt, tf.TitleSearch,
				db.FormatTime(tf.FetchedAt), existing.ID)
			if err != nil {
				return nil, err
			}
			result.Updated = append(result.Updated, NotifiedResource{ID: existing.ID, Provider: tf.Provider, ProviderID: tf.ProviderID, Title: tf.Title})
			result.Changed = append(result.Changed, existing.ID)
		}
	}

	return result, nil
}

// MarkDeletedResources ports markDeletedResources: rows of this provider with
// created_at strictly inside (min, max) of the batch that are not in the
// batch get is_deleted = true.
func (s *Store) MarkDeletedResources(provider string, resources []NewResource) ([]int64, error) {
	if len(resources) == 0 {
		return nil, nil
	}
	minT, maxT := resources[0].CreatedAt, resources[0].CreatedAt
	for _, r := range resources {
		if r.CreatedAt.Before(minT) {
			minT = r.CreatedAt
		}
		if r.CreatedAt.After(maxT) {
			maxT = r.CreatedAt
		}
	}
	inSet := map[string]bool{}
	for _, r := range resources {
		inSet[r.ProviderID] = true
	}

	rows, err := s.DB.Query(
		`SELECT id, provider_id FROM resources
		 WHERE provider_name = ? AND is_deleted = 0 AND created_at > ? AND created_at < ?`,
		provider, db.FormatTime(minT), db.FormatTime(maxT))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var pid string
		if err := rows.Scan(&id, &pid); err != nil {
			return nil, err
		}
		if inSet[pid] {
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, id := range ids {
		if _, err := s.DB.Exec("UPDATE resources SET is_deleted = 1 WHERE id = ?", id); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// MaintainDuplicatedResources ports maintainDuplicatedResources: for the
// changed ids, compute magnet variants, pick the winner by provider priority /
// createdAt / id, attach others as duplicates.
func (s *Store) MaintainDuplicatedResources(ids []int64) (attached, detached []int64, err error) {
	for _, id := range ids {
		r, err := s.GetByID(id)
		if err != nil {
			return attached, detached, err
		}
		if r == nil || r.Magnet == "" || r.IsDeleted {
			continue
		}
		hexVar := scraper.NormalizeBtihToHex(r.Magnet)
		b32Var := scraper.NormalizeBtihToBase32(r.Magnet)
		if hexVar == b32Var {
			continue // invalid magnet: both variants unchanged
		}

		rows, err := s.DB.Query(
			`SELECT id, provider_name, created_at, duplicated_id FROM resources
			 WHERE is_deleted = 0 AND (magnet = ? OR magnet = ?)`,
			hexVar, b32Var)
		if err != nil {
			return attached, detached, err
		}
		type cand struct {
			id           int64
			provider     string
			createdAt    time.Time
			duplicatedID sql.NullInt64
		}
		var cands []cand
		for rows.Next() {
			var c cand
			var createdAt string
			if err := rows.Scan(&c.id, &c.provider, &createdAt, &c.duplicatedID); err != nil {
				rows.Close()
				return attached, detached, err
			}
			c.createdAt, _ = db.ParseTime(createdAt)
			cands = append(cands, c)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return attached, detached, err
		}

		if len(cands) < 2 {
			continue
		}

		providerPriority := map[string]int{}
		for i, p := range []string{"dmhy", "moe", "mikan", "ani"} {
			providerPriority[p] = i
		}
		sort.Slice(cands, func(i, j int) bool {
			pi := providerPriority[cands[i].provider]
			pj := providerPriority[cands[j].provider]
			if pi != pj {
				return pi < pj
			}
			if !cands[i].createdAt.Equal(cands[j].createdAt) {
				return cands[i].createdAt.Before(cands[j].createdAt)
			}
			return cands[i].id < cands[j].id
		})

		winner := cands[0]
		for _, c := range cands[1:] {
			if !c.duplicatedID.Valid || c.duplicatedID.Int64 != winner.id {
				if _, err := s.DB.Exec("UPDATE resources SET duplicated_id = ? WHERE id = ?", winner.id, c.id); err != nil {
					return attached, detached, err
				}
				attached = append(attached, c.id)
			}
		}
		if winner.duplicatedID.Valid {
			if _, err := s.DB.Exec("UPDATE resources SET duplicated_id = NULL WHERE id = ?", winner.id); err != nil {
				return attached, detached, err
			}
			detached = append(detached, winner.id)
		}
	}
	return attached, detached, nil
}

// --- Details ---

// DetailRow mirrors the details table row.
type DetailRow struct {
	ID           int64
	Description  string
	Magnets      string
	Files        string
	HasMoreFiles bool
	FetchedAt    time.Time
}

const detailColumns = `id, description, magnets, files, has_more_files, fetched_at`

// GetDetail fetches the detail row for a resource id.
func (s *Store) GetDetail(id int64) (*DetailRow, error) {
	row := s.DB.QueryRow("SELECT "+detailColumns+" FROM details WHERE id = ?", id)
	var d DetailRow
	var hasMore int
	var fetchedAt string
	err := row.Scan(&d.ID, &d.Description, &d.Magnets, &d.Files, &hasMore, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.HasMoreFiles = hasMore != 0
	d.FetchedAt, _ = db.ParseTime(fetchedAt)
	return &d, nil
}

// InsertDetail ports insertDetail (upsert).
func (s *Store) InsertDetail(id int64, detail model.ResourceDetail) error {
	magnets, _ := json.Marshal(detail.Magnets)
	files, _ := json.Marshal(detail.Files)
	now := db.Now()
	resp, err := s.DB.Exec(`INSERT INTO details (id, description, magnets, files, has_more_files, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		id, detail.Description, string(magnets), string(files), boolInt(detail.HasMoreFiles), now)
	if err != nil {
		return err
	}
	if n, _ := resp.RowsAffected(); n == 0 {
		_, err = s.DB.Exec(`UPDATE details SET description = ?, magnets = ?, files = ?, has_more_files = ?, fetched_at = ? WHERE id = ?`,
			detail.Description, string(magnets), string(files), boolInt(detail.HasMoreFiles), now, id)
	}
	return err
}

// --- helpers ---

func nullableInt64SQL(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func nullableInt64Ptr(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func equalNullInt64(a, b sql.NullInt64) bool {
	if a.Valid != b.Valid {
		return false
	}
	return !a.Valid || a.Int64 == b.Int64
}

// equalSubjectID ports shouldOverwriteSubjectId: old != next; a null next
// never differs (we never reset subject ids).
func equalSubjectID(a sql.NullInt64, b *int64) bool {
	if b == nil {
		return false
	}
	return a.Valid && a.Int64 == *b
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
