// Command export-followers-csv exports a user's followers to a CSV file.
//
//	TWITTRAPI_API_KEY=... go run ./examples/export-followers-csv jack 5
//
// Arguments: username, max pages (default 5). Each page is one billed request of up to 100
// users.
package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"

	twittrapi "github.com/twittrapi/twittrapi-go"
	"github.com/twittrapi/twittrapi-go/examples/internal/extract"
)

func main() {
	username, maxPages := "jack", 5
	if len(os.Args) > 1 {
		username = os.Args[1]
	}
	if len(os.Args) > 2 {
		n, err := strconv.Atoi(os.Args[2])
		if err != nil || n < 1 {
			log.Fatalf("max pages must be a positive number, got %q", os.Args[2])
		}
		maxPages = n
	}

	client, err := twittrapi.NewClient() // TWITTRAPI_API_KEY or RAPIDAPI_KEY
	if err != nil {
		log.Fatal(err)
	}

	rows := [][]string{{"id", "username", "name", "followers", "following", "verified", "bio"}}
	seen := map[string]bool{}

	it := twittrapi.Paginate(context.Background(), client.Users.Followers,
		twittrapi.UserTimelineParams{Username: username, Count: 100},
		&twittrapi.PaginateOptions{MaxPages: maxPages})
	for it.Next() {
		page := it.Page()
		for _, u := range extract.Users(page.Data) {
			if seen[u.ID] {
				continue
			}
			seen[u.ID] = true
			rows = append(rows, []string{
				u.ID, u.Username, u.Name,
				strconv.FormatInt(u.Followers, 10), strconv.FormatInt(u.Following, 10),
				strconv.FormatBool(u.Verified), u.Bio,
			})
		}
		fmt.Fprintf(os.Stderr, "page %d: %d followers so far\n", page.Number, len(seen))
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}

	file := username + "-followers.csv"
	f, err := os.Create(file)
	if err != nil {
		log.Fatal(err)
	}
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote %d followers to %s\n", len(rows)-1, file)
}
