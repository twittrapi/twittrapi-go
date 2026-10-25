<p align="center">
  <a href="https://twittrapi.com"><img src=".github/banner.png" alt="twittrapi-go: Go SDK for Twitter/X data, standard library only, no X developer account" width="100%"></a>
</p>

# TwittrAPI Go SDK – Twitter/X API client for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/twittrapi/twittrapi-go.svg)](https://pkg.go.dev/github.com/twittrapi/twittrapi-go)
[![CI](https://github.com/twittrapi/twittrapi-go/actions/workflows/ci.yml/badge.svg)](https://github.com/twittrapi/twittrapi-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Official Go SDK for [TwittrAPI](https://twittrapi.com), a REST API for public Twitter/X data:
tweets, profiles, followers, search, lists, communities and Spaces, plus account actions (post, like,
follow, DMs) for your own X account. From $0.065 per 1,000 calls, no X
developer account.

> **Get an API key:** [twittrapi.com/start](https://twittrapi.com/start?utm_source=github&utm_medium=readme&utm_campaign=twittrapi-go) · [Pricing](https://twittrapi.com/pricing?utm_source=github&utm_medium=readme&utm_campaign=twittrapi-go) · [Docs](https://twittrapi.com/docs?utm_source=github&utm_medium=readme&utm_campaign=twittrapi-go) · [Status](https://twittrapi.com/status) · also available on [RapidAPI](https://rapidapi.com/viperscores-viperscores-default/api/twitter-aio)

**SDKs and integrations:** [Python](https://github.com/twittrapi/twittrapi-python) · [Node.js](https://github.com/twittrapi/twittrapi-node) · [MCP server](https://github.com/twittrapi/twittrapi-mcp) · [n8n](https://github.com/twittrapi/n8n-nodes-twittrapi) · [Make](https://www.make.com/en/hq/app-invitation/bfe92727f14ab2c74d3c2e74281d84b1) · [Apify](https://github.com/twittrapi/twittrapi-apify) · [LangChain](https://github.com/twittrapi/langchain-twittrapi) · [LlamaIndex](https://github.com/twittrapi/llama-index-tools-twittrapi) · [Agent skill](https://github.com/twittrapi/twittrapi-skills) · [Examples](https://github.com/twittrapi/twittrapi-examples)

- Standard library only. Go 1.21 or later.
- Every method takes a `context.Context`.
- Retries, timeouts and typed errors that work with `errors.As`.
- A page iterator that follows X's cursors.

Responses are X's raw GraphQL JSON, returned unchanged as `json.RawMessage`. The SDK does not
reshape them yet. Decode them into your own structs or a `map[string]any`. Typed models may be
added in a later version.

Not affiliated with X Corp.

## Install

```sh
go get github.com/twittrapi/twittrapi-go
```

## Quickstart

Get an API key at [twittrapi.com](https://twittrapi.com), or subscribe to TwittrAPI on RapidAPI.

### Direct API key

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	twittrapi "github.com/twittrapi/twittrapi-go"
)

func main() {
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
```

Requests go to `https://api.twittrapi.com` with the `X-API-KEY` header.

### RapidAPI key

```go
client, err := twittrapi.NewClient(twittrapi.WithRapidAPIKey(os.Getenv("RAPIDAPI_KEY")))
```

Requests go to `https://twitter-aio.p.rapidapi.com` with `x-rapidapi-key` and `x-rapidapi-host`.

Subscribed to another TwittrAPI listing on RapidAPI? Set its host with
`twittrapi.WithRapidAPIHost("twttr.p.rapidapi.com")` or the `RAPIDAPI_HOST` environment variable.

### Keys from the environment

With no key option, the client reads `TWITTRAPI_API_KEY`, then `RAPIDAPI_KEY`:

```go
client, err := twittrapi.NewClient() // uses TWITTRAPI_API_KEY or RAPIDAPI_KEY
```

If both environment variables are set, the direct key is used. Passing both `WithAPIKey` and
`WithRapidAPIKey` is an error. When no key is found, `NewClient` returns `twittrapi.ErrNoAPIKey`.

## Options

```go
client, err := twittrapi.NewClient(
	twittrapi.WithAPIKey("..."),                    // direct API key (or TWITTRAPI_API_KEY)
	twittrapi.WithRapidAPIKey("..."),               // RapidAPI key (or RAPIDAPI_KEY)
	twittrapi.WithRapidAPIHost("..."),              // RapidAPI host (or RAPIDAPI_HOST), default twitter-aio.p.rapidapi.com
	twittrapi.WithBaseURL("..."),                   // override the base URL (direct mode also reads TWITTRAPI_BASE_URL)
	twittrapi.WithHTTPClient(&http.Client{}),       // your own *http.Client (proxies, transports)
	twittrapi.WithTimeout(60*time.Second),          // per attempt, default 60s
	twittrapi.WithMaxRetries(2),                    // for 429, 5xx, timeouts and network errors
	twittrapi.WithRetryDelay(500*time.Millisecond), // base backoff delay
	twittrapi.WithFormat(twittrapi.FormatSimple),   // response format, default FormatRaw
)
```

Use the context to cancel a request or set an overall deadline, including retries:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
user, err := client.Users.GetByID(ctx, "12")
```

## Usage

IDs are strings of digits. Methods that take a user accept either `UserID` (faster) or
`Username`. A leading `@` is fine. Paginated methods take `Count` (1 to 100, 0 for the API
default of 20) and `Cursor`.

All examples below assume `ctx := context.Background()` and return `(json.RawMessage, error)`.

### Users

```go
client.Users.GetByUsername(ctx, "jack")
client.Users.GetByID(ctx, "12")
client.Users.UsernameToID(ctx, "jack")
client.Users.GetMany(ctx, []string{"12", "13", "783214"}) // up to 500 IDs

client.Users.Tweets(ctx, twittrapi.UserTimelineParams{Username: "jack", Count: 50})
client.Users.TweetsAndReplies(ctx, twittrapi.UserTimelineParams{UserID: "12"})
client.Users.Media(ctx, twittrapi.UserTimelineParams{UserID: "12"})
client.Users.Followers(ctx, twittrapi.UserTimelineParams{Username: "jack", Count: 100})
client.Users.Following(ctx, twittrapi.UserTimelineParams{UserID: "12"})
client.Users.VerifiedFollowers(ctx, twittrapi.UserTimelineParams{UserID: "12"})
client.Users.Affiliates(ctx, twittrapi.UserTimelineParams{Username: "XDevelopers"}) // accounts affiliated with an organization

client.Users.FriendshipsBatch(ctx, "jack", []string{"elonmusk", "XDevelopers"}) // up to 50 targets, one call each
client.Users.AboutBatch(ctx, []string{"jack", "XDevelopers"})                 // "About this account", up to 20, one call each
```

### Tweets

```go
client.Tweets.Get(ctx, twittrapi.TweetParams{TweetID: "20"}) // tweet plus reply thread
client.Tweets.Get(ctx, twittrapi.TweetParams{TweetID: "20", RankingMode: twittrapi.RankingRecency}) // Relevance | Recency | Likes
client.Tweets.GetMany(ctx, []string{"20", "1585841080431321088"}) // up to 100 IDs, one call
client.Tweets.Retweeters(ctx, twittrapi.TweetPageParams{TweetID: "20"})
client.Tweets.Quotes(ctx, twittrapi.TweetPageParams{TweetID: "20"})
client.Tweets.Replies(ctx, twittrapi.RepliesParams{TweetID: "20", RankingMode: twittrapi.RankingLikes}) // only the replies
client.Tweets.Thread(ctx, twittrapi.ThreadParams{TweetID: "20"})                                       // the author's thread, one page

// Likers works only for your own tweets: X shows who liked a tweet only to its author.
// Pass the Cookie header of the account that posted it (with auth_token and ct0) and a proxy.
client.Tweets.Likers(ctx, twittrapi.LikersParams{
	TweetID: "20",
	Cookie:  "auth_token=...; ct0=...",
	Proxy:   "http://user:pass@host:port",
})
```

### Search

`Query` supports X's search operators, such as `from:`, `to:`, `"exact phrase"`, `lang:`,
`since:`, `until:`, `min_faves:` and `-filter:replies`.

```go
client.Search.Tweets(ctx, twittrapi.SearchParams{
	Query:    "from:jack lang:en -filter:replies",
	Category: twittrapi.CategoryLatest, // Top (default) | Latest | People | Photos | Videos
	Count:    20,
})
```

### Lists

```go
client.Lists.Details(ctx, "1234567890")
client.Lists.Tweets(ctx, twittrapi.ListPageParams{ListID: "1234567890"})
client.Lists.Members(ctx, twittrapi.ListPageParams{ListID: "1234567890"})
```

### Communities

```go
client.Communities.Get(ctx, "1234567890")
client.Communities.About(ctx, "1234567890") // rules, moderators, members preview
client.Communities.Tweets(ctx, twittrapi.CommunityPageParams{CommunityID: "1234567890"})
client.Communities.Search(ctx, twittrapi.CommunitySearchParams{Query: "golang"})
```

### Spaces

```go
client.Spaces.Get(ctx, "1YqKDqWqdPLsV") // ID from x.com/i/spaces/<id>
```

### Trends

```go
client.Trends.Get(ctx, twittrapi.WOEIDWorldwide) // 1
client.Trends.Get(ctx, 23424977)                 // United States
client.Trends.Locations(ctx)                     // all locations with their WOEIDs
```

### Decoding responses

```go
raw, err := client.Users.GetByUsername(ctx, "jack")
if err != nil {
	log.Fatal(err)
}
var resp struct {
	Data struct {
		User struct {
			Result struct {
				RestID string `json:"rest_id"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}
if err := json.Unmarshal(raw, &resp); err != nil {
	log.Fatal(err)
}
fmt.Println(resp.Data.User.Result.RestID)
```

The exact shape is X's and can change. Check a real response before relying on a field.

### Simple format

By default every method returns X's raw JSON. The API also has an opt-in
[simple format](https://twittrapi.com/docs/simple-format): flat camelCase tweets, users, lists and
communities with a fixed schema, in a `{"data", "nextCursor", "previousCursor", "hasNextPage"}`
envelope. It is sent as `format=simple` and costs the same.

Typed helpers request it for one call and decode the result:

```go
page, err := twittrapi.FetchSimplePage[twittrapi.SimpleTweet](ctx, client.Users.Tweets,
	twittrapi.UserTimelineParams{Username: "XDevelopers", Count: 20})
// handle err
for _, t := range page.Data {
	fmt.Println(t.ID, t.Author.Username, t.Counts.Likes, t.Text)
}
fmt.Println(page.HasNextPage, page.Next())

user, err := twittrapi.FetchSimple[twittrapi.SimpleUser](ctx, client.Users.GetByUsername, "jack")
// user is nil when the handle does not exist
```

`twittrapi.PaginateSimple` pages through a method in the simple format, following `nextCursor`
until `hasNextPage` is false:

```go
it := twittrapi.PaginateSimple[twittrapi.SimpleUser](ctx, client.Users.Followers,
	twittrapi.UserTimelineParams{Username: "jack", Count: 100}, nil)
for it.Next() {
	for _, u := range it.Page().Data {
		fmt.Println(u.Username)
	}
}
if err := it.Err(); err != nil {
	log.Fatal(err)
}
```

Types: `SimpleTweet`, `SimpleUser`, `SimpleUserAbout`, `SimpleList`, `SimpleCommunity`,
`SimpleTrend`, `SimpleTrendLocation` and `SimplePage[T]`. A tweet X did not return is decoded
with only `ID` and `Unavailable: true`. `Users.Affiliates` pages are users, `Tweets.Replies` and
`Tweets.Thread` pages are tweets. `Spaces.Get`, `Communities.About`, the batch lookups and the
account actions have no simple format.

To get the simple format as `json.RawMessage` from every method, create the client with
`twittrapi.WithFormat(twittrapi.FormatSimple)`, or set it for one call with
`twittrapi.ContextWithFormat(ctx, twittrapi.FormatSimple)`. Decode such bodies with
`twittrapi.DecodeSimple` / `twittrapi.DecodeSimplePage`. `Paginate` recognizes simple pages and
follows `nextCursor` too.

### Endpoints not wrapped yet

`client.Get` sends an authenticated GET with the same retries and error handling, `client.Post`
a POST with a JSON body:

```go
data, err := client.Get(ctx, "/some/endpoint", url.Values{"foo": {"bar"}})
data, err = client.Post(ctx, "/some/endpoint", nil, map[string]any{"foo": "bar"})
```

## Account actions

Post, like, follow and more as your own X account. Pass the account's session to
`client.Account`; every call is a POST to `/client/...` with the cookie and proxy in the body.

```go
acct := client.Account(twittrapi.XSession{
	Cookie: os.Getenv("X_COOKIE"), // "auth_token=...; ct0=...; twid=u%3D..."
	Proxy:  os.Getenv("X_PROXY"),  // "http://user:pass@host:port"
})

acct.CreateTweet(ctx, twittrapi.CreateTweetParams{Text: "Hello from Go"})
acct.Reply(ctx, twittrapi.ReplyParams{TweetID: "20", Text: "Nice"})
acct.Like(ctx, "20")
acct.Follow(ctx, twittrapi.UserByUsername("jack")) // or twittrapi.UserByID("12")
acct.Bookmarks(ctx, twittrapi.AccountPageParams{Count: 50})
acct.SearchBookmarks(ctx, twittrapi.SearchBookmarksParams{Query: "golang"})

acct.UploadImage(ctx, twittrapi.ImageUpload{Data: png}) // media ID for CreateTweetParams.MediaIDs
acct.UpdateProfile(ctx, twittrapi.ProfileUpdate{Description: twittrapi.Ptr("Gopher")})

dm := acct.WithDMPasscode(os.Getenv("X_CHAT_PASSCODE"))
dm.SendDM(ctx, twittrapi.SendDMParams{UserID: "12", Text: "hi"})
```

Also covered: delete, retweet, pin, mute, block, report, polls, scheduled tweets, home timeline,
likes, blocked and muted accounts, lists, communities and your memberships, drafts, video
upload, profile picture and banner, and direct message settings. See the
[package docs](https://pkg.go.dev/github.com/twittrapi/twittrapi-go#AccountService).

Bookmark folders and articles need X Premium on the account:

```go
acct.BookmarkFolders(ctx, twittrapi.AccountCursorParams{})
acct.BookmarkFolder(ctx, twittrapi.BookmarkFolderParams{FolderID: "1700000000000000000"})

acct.Articles(ctx, twittrapi.ArticlesParams{Lifecycle: twittrapi.ArticlePublished})
acct.Article(ctx, "1800000000000000000")
acct.CreateArticle(ctx, twittrapi.CreateArticleParams{Title: "Hello", Text: "First paragraph\nSecond"})
acct.UpdateArticle(ctx, twittrapi.UpdateArticleParams{ArticleID: "1800000000000000000", Title: twittrapi.Ptr("New title")})
acct.PublishArticle(ctx, twittrapi.PublishArticleParams{ArticleID: "1800000000000000000", TweetText: "New article"})
acct.UnpublishArticle(ctx, "1800000000000000000")
acct.DeleteArticle(ctx, "1800000000000000000")
```

The cookie and proxy are secrets: `XSession` and the service mask them when printed. The cookie
comes from a logged-in x.com browser session; see
[twittrapi.com/docs/x-session](https://twittrapi.com/docs/x-session). Use one proxy per account.

Most answers are `{"status_code": ..., "response": "<X body>"}`; `twittrapi.UnwrapAccountResponse`
returns X's status and body. Actions that create something (tweets, replies, uploads, messages)
and article changes are not retried after a timeout or 5xx, so they never run twice.

## Pagination

`twittrapi.Paginate` calls a paginated method repeatedly and follows the bottom cursor in each
response. Pass the method value and its params:

```go
it := twittrapi.Paginate(ctx, client.Users.Followers,
	twittrapi.UserTimelineParams{Username: "jack", Count: 100},
	&twittrapi.PaginateOptions{MaxPages: 5})
for it.Next() {
	page := it.Page()
	fmt.Println(page.Number, page.NextCursor)
	// page.Data is the raw JSON of this page
}
if err := it.Err(); err != nil {
	log.Fatal(err)
}
```

There is also a callback form:

```go
err := twittrapi.Paginate(ctx, client.Search.Tweets, twittrapi.SearchParams{Query: "golang"}, nil).
	ForEach(func(page twittrapi.Page) error {
		// return an error to stop early
		return nil
	})
```

With Go 1.23 or later you can range over pages:

```go
for page, err := range twittrapi.Paginate(ctx, client.Lists.Tweets, twittrapi.ListPageParams{ListID: "1"}, nil).All() {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(page.Number)
}
```

Every page is a billed request, so `MaxPages` defaults to 10 (`nil` options or `MaxPages: 0`).
Use `twittrapi.NoPageLimit` to fetch until the cursor runs out. Iteration also stops when no
bottom cursor is found, when the cursor repeats, or when a page contains only cursor entries.
Set `Cursor` in the params to start from a saved cursor.

To page manually, pass the cursor yourself:

```go
first, err := client.Search.Tweets(ctx, twittrapi.SearchParams{Query: "golang"})
// handle err
if cursor := twittrapi.ExtractBottomCursor(first); cursor != "" {
	next, err := client.Search.Tweets(ctx, twittrapi.SearchParams{Query: "golang", Cursor: cursor})
	// ...
}
```

Cursor detection is best-effort. It looks for timeline entries with `cursorType: "Bottom"` or an
`entryId` starting with `cursor-bottom-`, which is how X structures its timelines today. If X
changes that format, pagination may stop early until the SDK is updated.

## Errors

HTTP errors are returned as pointer types. Check them with `errors.As`. All of them also match
`*twittrapi.APIError`, which has `StatusCode`, `Message`, `Detail` (the API's error message),
`Body` and `Header`.

| Type                        | When                                          |
| --------------------------- | --------------------------------------------- |
| `*AuthenticationError`      | 401 or 403                                    |
| `*InsufficientBalanceError` | 402: no balance left, top up your account     |
| `*NotFoundError`            | 404: user, tweet, list or community not found |
| `*ValidationError`          | 400 or 422                                    |
| `*RateLimitError`           | 429. `RetryAfter` holds the wait (-1 if none) |
| `*ServerError`              | 5xx                                           |
| `*APIError`                 | Any other error status                        |
| `*ConnectionError`          | Network failure, no response                  |
| `*TimeoutError`             | An attempt took longer than `WithTimeout`     |

`*TimeoutError` also matches `*ConnectionError`. Invalid arguments caught before a request is
sent (a non-numeric ID, more than 500 user IDs, a count over 100) wrap `twittrapi.ErrInvalidParams`.
When your context is canceled or its deadline passes, the context's error is returned.

```go
_, err := client.Users.GetByUsername(ctx, "someone")

var notFound *twittrapi.NotFoundError
var rateLimit *twittrapi.RateLimitError
var noBalance *twittrapi.InsufficientBalanceError
var apiErr *twittrapi.APIError
switch {
case err == nil:
	// ok
case errors.As(err, &notFound):
	fmt.Println("No such user")
case errors.As(err, &rateLimit):
	fmt.Println("Slow down, retry in", rateLimit.RetryAfter)
case errors.As(err, &noBalance):
	fmt.Println("Top up at https://twittrapi.com")
case errors.As(err, &apiErr):
	fmt.Println(apiErr.StatusCode, apiErr.Message)
case errors.Is(err, twittrapi.ErrInvalidParams):
	fmt.Println("Bad argument:", err)
default:
	log.Fatal(err)
}
```

### Retries

429, 5xx, timeouts and network errors are retried up to `WithMaxRetries` times (default 2) with
exponential backoff starting at 500 ms (capped at 8 seconds, with jitter). On 429 the
`Retry-After` header is honoured (capped at 60 seconds). Other 4xx errors are not retried.
Account actions that create something and article changes are retried only after a 429.
Canceling the context stops the request and any pending retry.

## Examples

The [`examples/`](examples) folder has short programs:

- `export-followers-csv`: export a user's followers to CSV
- `keyword-monitor`: poll search and print new tweets
- `user-timeline-json`: save a user's tweets to JSON
- `trends-by-country`: print trends for a country code

```sh
TWITTRAPI_API_KEY=... go run ./examples/export-followers-csv jack 5
TWITTRAPI_API_KEY=... go run ./examples/keyword-monitor "golang lang:en" 60
TWITTRAPI_API_KEY=... go run ./examples/user-timeline-json jack 3
TWITTRAPI_API_KEY=... go run ./examples/trends-by-country DE
```

## Development

```sh
go vet ./...
go test -race ./...
```

## Support

Questions, bugs or billing: email [support@twittrapi.com](mailto:support@twittrapi.com) or open an
issue on GitHub. API docs are at [twittrapi.com/docs](https://twittrapi.com/docs).

## License

MIT. Not affiliated with X Corp. "Twitter" and "X" are trademarks of X Corp.
