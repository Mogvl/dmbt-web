// Package providers ports apps/server/src/providers: per-provider wiring of
// the scrapers into the system (latest fetch, page sync, detail resolution).
package providers

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/Mogvl/dmbt-web/server/internal/scraper"
)

// Provider is the runtime abstraction over one scraper.
type Provider struct {
	ID           string
	Name         string
	FetchLatest  func(retry int) ([]scraper.ScrapedResource, error)
	FetchPages   func(start, end int) ([]scraper.ScrapedResource, error)
	FetchDetail  func(id string, retry int) (*scraper.ScrapedResourceDetail, error)
	GetDetailURL func(sys System, path string) (*DetailURL, error)
}

// DetailURL mirrors the resolved detail URL.
type DetailURL struct {
	ProviderID string
	Href       string
}

// System is the minimal interface providers need from the app.
type System interface {
	GetResourceHref(provider, providerID string) (string, error)
	ResourceIDsExist(provider string, ids []string) (map[string]bool, error)
}

// Registry holds all providers keyed by id.
type Registry struct {
	providers map[string]*Provider
}

// NewRegistry builds the provider registry in SupportProviders order.
func NewRegistry(sys System) *Registry {
	reg := &Registry{providers: map[string]*Provider{}}
	reg.providers["dmhy"] = newDmhyProvider(sys)
	reg.providers["moe"] = newMoeProvider(sys)
	reg.providers["mikan"] = newMikanProvider(sys)
	reg.providers["ani"] = newAniProvider(sys)
	return reg
}

// SupportProviders mirrors the original constant order (business priority).
var SupportProviders = []string{"dmhy", "moe", "mikan", "ani"}

// Get returns a provider by id.
func (r *Registry) Get(id string) (*Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// All returns providers in SupportProviders order.
func (r *Registry) All() []*Provider {
	out := make([]*Provider, 0, len(SupportProviders))
	for _, id := range SupportProviders {
		if p, ok := r.providers[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

// IDs returns the provider ids in SupportProviders order.
func (r *Registry) IDs() []string {
	return append([]string{}, SupportProviders...)
}

func newDmhyProvider(sys System) *Provider {
	return &Provider{
		ID:   "dmhy",
		Name: "动漫花园",
		FetchLatest: func(retry int) ([]scraper.ScrapedResource, error) {
			return FetchLatestPages(sys, "dmhy", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchDmhyPage(page, retry)
			})
		},
		FetchPages: func(start, end int) ([]scraper.ScrapedResource, error) {
			return FetchResourcePages(sys, "dmhy", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchDmhyPage(page, 5)
			}, start, end)
		},
		FetchDetail: func(id string, retry int) (*scraper.ScrapedResourceDetail, error) {
			return scraper.FetchDmhyDetail(id, retry)
		},
		GetDetailURL: func(sys System, path string) (*DetailURL, error) {
			// ports DmhyProvider.getDetailURL
			m := regexp.MustCompile(`^(\d+)`).FindStringSubmatch(path)
			if m == nil {
				return nil, nil
			}
			providerID := m[1]
			if path == providerID {
				href, err := sys.GetResourceHref("dmhy", path)
				if err != nil || href == "" {
					return nil, nil
				}
				return &DetailURL{ProviderID: providerID, Href: href}, nil
			}
			return &DetailURL{ProviderID: providerID, Href: path}, nil
		},
	}
}

func newMoeProvider(sys System) *Provider {
	return &Provider{
		ID:   "moe",
		Name: "萌番组",
		FetchLatest: func(retry int) ([]scraper.ScrapedResource, error) {
			return FetchLatestPages(sys, "moe", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchMoePage(page, retry)
			})
		},
		FetchPages: func(start, end int) ([]scraper.ScrapedResource, error) {
			return FetchResourcePages(sys, "moe", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchMoePage(page, 5)
			}, start, end)
		},
		FetchDetail: func(id string, retry int) (*scraper.ScrapedResourceDetail, error) {
			return scraper.FetchMoeDetail(id, retry)
		},
		GetDetailURL: func(_ System, path string) (*DetailURL, error) {
			return &DetailURL{ProviderID: path, Href: path}, nil
		},
	}
}

func newMikanProvider(sys System) *Provider {
	return &Provider{
		ID:   "mikan",
		Name: "蜜柑计划",
		FetchLatest: func(retry int) ([]scraper.ScrapedResource, error) {
			return FetchLatestPages(sys, "mikan", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchMikanPage(page, retry)
			})
		},
		FetchPages: func(start, end int) ([]scraper.ScrapedResource, error) {
			return FetchResourcePages(sys, "mikan", func(page int) ([]scraper.ScrapedResource, error) {
				return scraper.FetchMikanPage(page, 5)
			}, start, end)
		},
		FetchDetail: func(id string, retry int) (*scraper.ScrapedResourceDetail, error) {
			return scraper.FetchMikanDetail(id, retry)
		},
		GetDetailURL: func(_ System, path string) (*DetailURL, error) {
			return &DetailURL{ProviderID: path, Href: path}, nil
		},
	}
}

func newAniProvider(sys System) *Provider {
	return &Provider{
		ID:   "ani",
		Name: "ANi",
		FetchLatest: func(retry int) ([]scraper.ScrapedResource, error) {
			// ANi override: single RSS fetch, then filter against the DB.
			resources, err := scraper.FetchAniLatest(retry)
			if err != nil || sys == nil {
				return resources, err
			}
			ids := idsOf(resources)
			existing, err := sys.ResourceIDsExist("ani", ids)
			if err != nil {
				return resources, err
			}
			var out []scraper.ScrapedResource
			for _, r := range resources {
				if !existing[r.ProviderID] {
					out = append(out, r)
				}
			}
			return out, nil
		},
		FetchPages: func(_, _ int) ([]scraper.ScrapedResource, error) {
			return scraper.FetchAniLatest(5)
		},
		FetchDetail: func(id string, retry int) (*scraper.ScrapedResourceDetail, error) {
			return scraper.FetchAniDetail(id, retry)
		},
		GetDetailURL: func(_ System, path string) (*DetailURL, error) {
			return &DetailURL{ProviderID: path, Href: path}, nil
		},
	}
}

// pageDelayMs is a politeness delay between upstream page requests. The
// original fires pages back-to-back; upstream rate limits require a small
// delay (SCRAPE_PAGE_DELAY_MS, default 800).
var pageDelayMs = func() int {
	if v := os.Getenv("SCRAPE_PAGE_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 800
}()

// maxFetchPages caps a single fetch walk so the job commits progress on
// first-run history imports (MAX_FETCH_PAGES, default 200; 0 = unlimited).
var maxFetchPages = func() int {
	if v := os.Getenv("MAX_FETCH_PAGES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 200
}()

func politeDelay() {
	if pageDelayMs > 0 {
		time.Sleep(time.Duration(pageDelayMs) * time.Millisecond)
	}
}

// FetchLatestPages ports fetchLatestPages from providers/scraper/base.ts:
// loop pages until a page yields zero real-new resources.
func FetchLatestPages(sys System, provider string, fetch func(page int) ([]scraper.ScrapedResource, error)) ([]scraper.ScrapedResource, error) {
	visited := map[string]scraper.ScrapedResource{}

	for page := 1; ; page++ {
		if maxFetchPages > 0 && page > maxFetchPages {
			break
		}
		if page > 1 {
			politeDelay()
		}
		resp, err := fetch(page)
		if err != nil {
			// Rate-limited or upstream error mid-walk: keep what we have
			// (the alternative — aborting the whole job — yields nothing).
			if scraper.IsNetworkError(err) && len(visited) > 0 {
				break
			}
			return nil, err
		}
		var newRes []scraper.ScrapedResource
		for _, r := range resp {
			if _, ok := visited[r.ProviderID]; !ok {
				newRes = append(newRes, r)
			}
		}
		var set map[string]bool
		if sys != nil {
			set, err = sys.ResourceIDsExist(provider, idsOf(newRes))
			if err != nil {
				return nil, err
			}
		} else {
			set = map[string]bool{}
		}
		var realNew []scraper.ScrapedResource
		for _, r := range newRes {
			if !set[r.ProviderID] {
				visited[r.ProviderID] = r
				realNew = append(realNew, r)
			}
		}
		if len(realNew) == 0 {
			break
		}
	}

	out := make([]scraper.ScrapedResource, 0, len(visited))
	for _, r := range visited {
		out = append(out, r)
	}
	return out, nil
}

// FetchResourcePages ports fetchResourcePages: page range with run-local dedup.
func FetchResourcePages(sys System, provider string, fetch func(page int) ([]scraper.ScrapedResource, error), start, end int) ([]scraper.ScrapedResource, error) {
	visited := map[string]scraper.ScrapedResource{}
	for page := start; page <= end; page++ {
		if page > start {
			politeDelay()
		}
		resp, err := fetch(page)
		if err != nil {
			if scraper.IsNetworkError(err) && len(visited) > 0 {
				break
			}
			return nil, err
		}
		for _, r := range resp {
			if _, ok := visited[r.ProviderID]; !ok {
				visited[r.ProviderID] = r
			}
		}
	}
	out := make([]scraper.ScrapedResource, 0, len(visited))
	for _, r := range visited {
		out = append(out, r)
	}
	_ = sys
	return out, nil
}

func idsOf(resources []scraper.ScrapedResource) []string {
	out := make([]string, 0, len(resources))
	for _, r := range resources {
		out = append(out, r.ProviderID)
	}
	return out
}

// SortByCreatedAt sorts resources by createdAt ascending (used by fetch jobs).
func SortByCreatedAt(resources []scraper.ScrapedResource) {
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].CreatedAt.Before(resources[j].CreatedAt)
	})
}

var _ = fmt.Sprintf
