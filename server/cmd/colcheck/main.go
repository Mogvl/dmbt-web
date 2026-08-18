package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/Mogvl/dmbt-web/server/internal/api"
	"github.com/Mogvl/dmbt-web/server/internal/collections"
	"github.com/Mogvl/dmbt-web/server/internal/query"
	"github.com/Mogvl/dmbt-web/server/internal/resources"
	"github.com/Mogvl/dmbt-web/server/internal/search"
	"github.com/Mogvl/dmbt-web/server/internal/subjects"
)

func main() {
	tok, _ := search.NewTokenizer()
	db, _ := sql.Open("sqlite", "/root/deepseek/dmbt-web/server/data/animegarden.db")
	defer db.Close()
	store := resources.New(db, tok)
	store.RefreshParties()
	users, teams, err := store.LoadUserBriefs()
	fmt.Println("load briefs:", err, len(users), len(teams))
	subs := &subjects.Module{DB: db}
	subs.Load()
	q := query.New(&query.Store{DB: db, NameToUserID: store.NameToUserID, NameToTeamID: store.NameToTeamID, UserIDToName: store.UserIDToName, TeamIDToName: store.TeamIDToName}, tok)

	colStore := &collections.Store{DB: db}
	row, err := colStore.GetRow("3ecb2176c8f77f4c04e57973b2414b1e825822a0")
	fmt.Println("row:", err, row != nil)
	if row == nil {
		return
	}
	for _, f := range row.FiltersRaw {
		options := api.FilterFromStoredMap(f)
		fmt.Printf("options: search=%v types=%v\n", options.Search, options.Types)
		find, err := q.Find(options, 1, 1000)
		fmt.Printf("  find: %d resources, err=%v\n", len(find.Resources), err)
	}
}
