package twittrapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	twittrapi "github.com/twittrapi/twittrapi-go"
)

func ExampleNewClient() {
	// Direct API key from https://twittrapi.com.
	client, err := twittrapi.NewClient(twittrapi.WithAPIKey(os.Getenv("TWITTRAPI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}

	user, err := client.Users.GetByUsername(context.Background(), "jack")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(user))
}

func ExampleNewClient_rapidAPI() {
	client, err := twittrapi.NewClient(
		twittrapi.WithRapidAPIKey(os.Getenv("RAPIDAPI_KEY")),
		twittrapi.WithMaxRetries(3),
		twittrapi.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(client.AuthMode(), client.BaseURL())
}

func ExampleWithRapidAPIHost() {
	// A key from another TwittrAPI listing on RapidAPI. RAPIDAPI_HOST works too.
	client, err := twittrapi.NewClient(
		twittrapi.WithRapidAPIKey(os.Getenv("RAPIDAPI_KEY")),
		twittrapi.WithRapidAPIHost("twttr.p.rapidapi.com"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(client.RapidAPIHost(), client.BaseURL())
}

func ExamplePaginate() {
	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	it := twittrapi.Paginate(context.Background(), client.Users.Followers,
		twittrapi.UserTimelineParams{Username: "jack", Count: 100},
		&twittrapi.PaginateOptions{MaxPages: 5})
	for it.Next() {
		page := it.Page()
		fmt.Println(page.Number, page.NextCursor, len(page.Data))
	}
	if err := it.Err(); err != nil {
		log.Fatal(err)
	}
}

func ExampleExtractBottomCursor() {
	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	first, err := client.Search.Tweets(ctx, twittrapi.SearchParams{Query: "golang", Category: twittrapi.CategoryLatest})
	if err != nil {
		log.Fatal(err)
	}
	if cursor := twittrapi.ExtractBottomCursor(first); cursor != "" {
		next, err := client.Search.Tweets(ctx, twittrapi.SearchParams{Query: "golang", Category: twittrapi.CategoryLatest, Cursor: cursor})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(len(next))
	}
}

func ExampleRateLimitError() {
	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.Users.GetByUsername(context.Background(), "someone")
	var notFound *twittrapi.NotFoundError
	var rateLimit *twittrapi.RateLimitError
	var noBalance *twittrapi.InsufficientBalanceError
	var apiErr *twittrapi.APIError
	switch {
	case err == nil:
		fmt.Println("found")
	case errors.As(err, &notFound):
		fmt.Println("no such user")
	case errors.As(err, &rateLimit):
		fmt.Println("slow down, retry after", rateLimit.RetryAfter)
	case errors.As(err, &noBalance):
		fmt.Println("top up at https://twittrapi.com")
	case errors.As(err, &apiErr):
		fmt.Println(apiErr.StatusCode, apiErr.Detail)
	default:
		log.Fatal(err)
	}
}

func ExampleUsersService_UsernameToID() {
	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	// Decode the raw JSON into whatever shape you need.
	raw, err := client.Users.UsernameToID(context.Background(), "jack")
	if err != nil {
		log.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
}
