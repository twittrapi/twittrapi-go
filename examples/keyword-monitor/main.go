// Command keyword-monitor polls X search for a keyword and prints new tweets as they appear.
//
//	TWITTRAPI_API_KEY=... go run ./examples/keyword-monitor "golang lang:en -filter:replies" 60
//
// Arguments: query (X search operators work), poll interval in seconds (default 60). Press
// Ctrl+C to stop.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	twittrapi "github.com/twittrapi/twittrapi-go"
	"github.com/twittrapi/twittrapi-go/examples/internal/extract"
)

func main() {
	query, interval := "twittrapi", 60*time.Second
	if len(os.Args) > 1 {
		query = os.Args[1]
	}
	if len(os.Args) > 2 {
		secs, err := strconv.Atoi(os.Args[2])
		if err != nil || secs < 1 {
			log.Fatalf("interval must be a positive number of seconds, got %q", os.Args[2])
		}
		interval = time.Duration(secs) * time.Second
	}

	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	seen := map[string]bool{}
	first := true
	for {
		data, err := client.Search.Tweets(ctx, twittrapi.SearchParams{
			Query:    query,
			Category: twittrapi.CategoryLatest,
			Count:    20,
		})
		var rateLimit *twittrapi.RateLimitError
		switch {
		case ctx.Err() != nil:
			return
		case errors.As(err, &rateLimit):
			log.Printf("Rate limited, waiting. %v", err)
		case err != nil:
			log.Fatal(err)
		default:
			tweets := extract.Tweets(data) // newest first
			fresh := 0
			for i := len(tweets) - 1; i >= 0; i-- {
				t := tweets[i]
				if seen[t.ID] {
					continue
				}
				seen[t.ID] = true
				fresh++
				if !first {
					fmt.Printf("[%s] @%s: %s\n  https://x.com/i/status/%s\n",
						t.CreatedAt, t.Author, strings.Join(strings.Fields(t.Text), " "), t.ID)
				}
			}
			if first {
				fmt.Printf("Watching %q (skipped %d existing tweets)\n", query, fresh)
				first = false
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}
