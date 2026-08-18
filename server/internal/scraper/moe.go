package scraper

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// moe constants, mirroring packages/scraper/src/moe.
const (
	MoeBase   = "https://bangumi.moe"
	MoeName   = "moe"
	MoeDisplayName = "萌番组"
	MoeUA     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0"
)

// MoeTracker is the hard-coded tracker list appended to moe magnets.
const MoeTracker = "&tr=https%3A%2F%2Ftr.bangumi.moe%3A9696%2Fannounce&tr=http%3A%2F%2Ftr.bangumi.moe%3A6969%2Fannounce&tr=udp%3A%2F%2Ftr.bangumi.moe%3A6969%2Fannounce&tr=http%3A%2F%2Fopen.acgtracker.com%3A1096%2Fannounce&tr=http%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=http%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=udp%3A%2F%2F208.67.16.113%3A8000%2Fannounce&tr=http%3A%2F%2Ftracker.ktxp.com%3A6868%2Fannounce&tr=http%3A%2F%2Ftracker.ktxp.com%3A7070%2Fannounce&tr=http%3A%2F%2Ft2.popgo.org%3A7456%2Fannonce&tr=http%3A%2F%2Fbt.sc-ol.com%3A2710%2Fannounce&tr=http%3A%2F%2Fshare.camoe.cn%3A8080%2Fannounce&tr=http%3A%2F%2F61.154.116.205%3A8000%2Fannounce&tr=http%3A%2F%2Fbt.rghost.net%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.openbittorrent.com%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.publicbt.com%3A80%2Fannounce&tr=http%3A%2F%2Ftracker.prq.to%2Fannounce&tr=http%3A%2F%2Fopen.nyaatorrents.info%3A6544%2Fannounce"

// moe tag id -> type mapping.
var moeTagTypes = map[string]string{
	"549ef207fe682f7549f1ea90": "动画",
	"54967e14ff43b99e284d0bf7": "合集",
	"549eefebfe682f7549f1ea8c": "漫画",
	"549eef6ffe682f7549f1ea8b": "音乐",
	"549ff1db30bcfc225bf9e607": "日剧",
	"549ef015fe682f7549f1ea8d": "游戏",
	"549ef250fe682f7549f1ea91": "其他",
}

type moeTorrent struct {
	ID          string     `json:"_id"`
	Title       string     `json:"title"`
	Magnet      string     `json:"magnet"`
	Size        any        `json:"size"`
	PublishTime string     `json:"publish_time"`
	UploaderID  string     `json:"uploader_id"`
	TeamID      string     `json:"team_id,omitempty"`
	TagIDs      []string   `json:"tag_ids"`
	Introduction string    `json:"introduction"`
	Content     [][]string `json:"content"`
}

type moePageResp struct {
	Torrents []moeTorrent `json:"torrents"`
}

type moeUserResp struct {
	Username  string `json:"username"`
	EmailHash string `json:"emailHash"`
}

type moeTeamResp struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// moeCaches mirror the module-level Map caches in the original scraper.
var (
	moeUserCache sync.Map // id -> moeUserResp
	moeTeamCache sync.Map // id -> moeTeamResp
)

// FetchMoePage fetches one bangumi.moe page (1-based), mirroring
// fetchMoePage from packages/scraper/src/moe/index.ts.
func FetchMoePage(page int, retry int) ([]ScrapedResource, error) {
	return retryFn(retry, func() ([]ScrapedResource, error) {
		url := fmt.Sprintf("%s/api/torrent/page/%d", MoeBase, page)
		resp, err := fetchURL("GET", url, MoeUA, nil, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var payload moePageResp
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, err
		}
		var out []ScrapedResource
		for _, t := range payload.Torrents {
			r, err := parseMoeTorrent(t)
			if err != nil {
				continue
			}
			out = append(out, r)
		}
		return out, nil
	})
}

func parseMoeTorrent(t moeTorrent) (ScrapedResource, error) {
	typ := "其他"
	for _, tagID := range t.TagIDs {
		if v, ok := moeTagTypes[tagID]; ok {
			typ = v
			break
		}
	}

	var publisher *Party
	var fansub *Party

	// resolve uploader
	if t.UploaderID != "" {
		user, err := fetchMoeUser(t.UploaderID)
		if err == nil {
			publisher = &Party{
				ID:   t.UploaderID,
				Name: user.Username,
			}
			if t.TeamID != "" {
				team, err2 := fetchMoeTeam(t.TeamID)
				if err2 == nil {
					if team.Icon != "" {
						publisher.Avatar = team.Icon
					} else if user.EmailHash != "" {
						publisher.Avatar = fmt.Sprintf("https://static.bangumi.moe/avatar/%s", user.EmailHash)
					}
					fansub = &Party{ID: t.TeamID, Name: team.Name, Avatar: team.Icon}
				}
			} else if user.EmailHash != "" {
				publisher.Avatar = fmt.Sprintf("https://static.bangumi.moe/avatar/%s", user.EmailHash)
			}
		}
	}

	size := ""
	switch v := t.Size.(type) {
	case string:
		size = v
	case float64:
		size = fmt.Sprintf("%d", int64(v))
	case nil:
	}

	createdAt := time.Time{}
	if t.PublishTime != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, t.PublishTime); err == nil {
			createdAt = parsed.UTC()
		} else if parsed, err := time.Parse(time.RFC3339, t.PublishTime); err == nil {
			createdAt = parsed.UTC()
		}
	}

	title := titleCleanup(t.Title, fansubName(fansub))

	return ScrapedResource{
		Provider:   MoeName,
		ProviderID: t.ID,
		Title:      title,
		Href:       t.ID,
		Type:       typ,
		Magnet:     t.Magnet,
		Tracker:    MoeTracker,
		Size:       size,
		Publisher:  publisher,
		Fansub:     fansub,
		CreatedAt:  createdAt,
	}, nil
}

func fetchMoeUser(id string) (moeUserResp, error) {
	if v, ok := moeUserCache.Load(id); ok {
		return v.(moeUserResp), nil
	}
	var out []moeUserResp
	err := moePostJSON("/api/user/fetch", map[string]any{"_ids": []string{id}}, &out)
	if err == nil && len(out) > 0 {
		moeUserCache.Store(id, out[0])
		return out[0], nil
	}
	return moeUserResp{}, err
}

func fetchMoeTeam(id string) (moeTeamResp, error) {
	if v, ok := moeTeamCache.Load(id); ok {
		return v.(moeTeamResp), nil
	}
	var out []moeTeamResp
	err := moePostJSON("/api/team/fetch", map[string]any{"_ids": []string{id}}, &out)
	if err == nil && len(out) > 0 {
		moeTeamCache.Store(id, out[0])
		return out[0], nil
	}
	return moeTeamResp{}, err
}

func moePostJSON(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := fetchURL("POST", MoeBase+path, MoeUA, strings.NewReader(string(data)), map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

// FetchMoeDetail fetches a bangumi.moe torrent by id, mirroring
// fetchMoeDetail from packages/scraper/src/moe/index.ts.
func FetchMoeDetail(id string, retry int) (*ScrapedResourceDetail, error) {
	return retryFn(retry, func() (*ScrapedResourceDetail, error) {
		var out []moeTorrent
		err := moePostJSON("/api/torrent/fetch", map[string]any{"_id": id}, &out)
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			return nil, nil
		}
		t := out[0]
		r, err := parseMoeTorrent(t)
		if err != nil {
			return nil, err
		}
		var files []FileEntry
		for _, c := range t.Content {
			if len(c) != 2 {
				continue
			}
			files = append(files, FileEntry{Name: c[0], Size: c[1]})
		}
		return &ScrapedResourceDetail{
			ScrapedResource: r,
			Description:     t.Introduction,
			Files:           files,
			Magnets:         []MagnetEntry{{Name: "磁力链接", URL: t.Magnet}},
			HasMoreFiles:    false,
		}, nil
	})
}
