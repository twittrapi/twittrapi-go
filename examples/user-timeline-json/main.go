// Command user-timeline-json saves a user's recent tweets to a JSON file.
//
//	TWITTRAPI_API_KEY=... go run ./examples/user-timeline-json jack 3
//
// Arguments: username, max pages (default 3). Set RAW=1 to save the raw API pages instead.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	twittrapi "github.com/twittrapi/twittrapi-go"
	"github.com/twittrapi/twittrapi-go/examples/internal/extract"
)

func main() {
	username, maxPages := "jack", 3
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
	raw := os.Getenv("RAW") != ""

	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	var pages []json.RawMessage
	err = twittrapi.Paginate(context.Background(), client.Users.Tweets,
		twittrapi.UserTimelineParams{Username: username, Count: 50},
		&twittrapi.PaginateOptions{MaxPages: maxPages},
	).ForEach(func(p twittrapi.Page) error {
		pages = append(pages, p.Data)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	var output any = pages
	summary := fmt.Sprintf("%d raw pages", len(pages))
	if !raw {
		tweets := []extract.Tweet{}
		for _, p := range pages {
			tweets = append(tweets, extract.Tweets(p)...)
		}
		output = tweets
		summary = fmt.Sprintf("%d tweets", len(tweets))
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	file := username + "-tweets.json"
	if err := os.WriteFile(file, data, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote %s to %s\n", summary, file)
}
