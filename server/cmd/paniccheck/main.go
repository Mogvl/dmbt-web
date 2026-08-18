package main

import (
	"fmt"

	"github.com/Mogvl/dmbt-web/server/internal/anipar"
	"github.com/Mogvl/dmbt-web/server/internal/scraper"
)

func main() {
	page, err := scraper.FetchMoePage(1, 3)
	if err != nil {
		fmt.Println("fetch error:", err)
		return
	}
	fmt.Println("moe page 1 rows:", len(page))
	for i, r := range page {
		func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("PANIC at %d: %v\nTITLE: %s\n", i, p, r.Title)
				}
			}()
			anipar.Parse(r.Title, "")
		}()
	}
	fmt.Println("done")
}
