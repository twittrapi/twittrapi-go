package twittrapi

// Tests for the opt-in simple format. Fixtures in testdata/simple are the backend's
// expected simple-format outputs (copied from the API's normalize testdata).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func simpleFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "simple", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestFormatDefaultRawSendsNoParam(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	if c.Format() != FormatRaw {
		t.Fatalf("default Format() = %q", c.Format())
	}
	if _, err := c.Users.Tweets(context.Background(), UserTimelineParams{UserID: "12", Count: 5}); err != nil {
		t.Fatal(err)
	}
	q := rec.last(t).URL.Query()
	if _, ok := q["format"]; ok {
		t.Fatalf("raw request sent format: %v", q)
	}
	if q.Get("count") != "5" {
		t.Fatalf("count = %q", q.Get("count"))
	}
}

func TestWithFormatSimpleSendsParam(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv, WithFormat(FormatSimple))
	if c.Format() != FormatSimple {
		t.Fatalf("Format() = %q", c.Format())
	}
	ctx := context.Background()
	if _, err := c.Users.GetByUsername(ctx, "jack"); err != nil {
		t.Fatal(err)
	}
	if got := rec.last(t).URL.Query().Get("format"); got != "simple" {
		t.Fatalf("format = %q", got)
	}
	if _, err := c.Search.Tweets(ctx, SearchParams{Query: "golang", Cursor: "C1"}); err != nil {
		t.Fatal(err)
	}
	q := rec.last(t).URL.Query()
	if q.Get("format") != "simple" || q.Get("searchItem") != "golang" || q.Get("cursor") != "C1" {
		t.Fatalf("query = %v", q)
	}
}

func TestWithFormatUnknownIsRaw(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv, WithFormat("xml"))
	if c.Format() != FormatRaw {
		t.Fatalf("Format() = %q", c.Format())
	}
	if _, err := c.Users.GetByID(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if rec.last(t).URL.RawQuery != "" {
		t.Fatalf("query = %q", rec.last(t).URL.RawQuery)
	}
}

func TestContextWithFormatOverridesClient(t *testing.T) {
	srv, rec := newServer(t)
	ctx := context.Background()

	raw, _ := newTestClient(t, srv)
	if _, err := raw.Users.GetByID(ContextWithFormat(ctx, FormatSimple), "12"); err != nil {
		t.Fatal(err)
	}
	if got := rec.last(t).URL.Query().Get("format"); got != "simple" {
		t.Fatalf("per-call simple: format = %q", got)
	}
	if _, err := raw.Users.GetByID(ctx, "12"); err != nil {
		t.Fatal(err)
	}
	if rec.last(t).URL.RawQuery != "" {
		t.Fatalf("next call should be raw again, query = %q", rec.last(t).URL.RawQuery)
	}

	simple, _ := newTestClient(t, srv, WithFormat(FormatSimple))
	if _, err := simple.Users.GetByID(ContextWithFormat(ctx, FormatRaw), "12"); err != nil {
		t.Fatal(err)
	}
	if rec.last(t).URL.RawQuery != "" {
		t.Fatalf("per-call raw: query = %q", rec.last(t).URL.RawQuery)
	}
}

func TestGetKeepsExplicitFormatParam(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv, WithFormat(FormatSimple))
	q := url.Values{"format": {"raw"}}
	if _, err := c.Get(context.Background(), "/misc/locations", q); err != nil {
		t.Fatal(err)
	}
	if got := rec.last(t).URL.Query()["format"]; len(got) != 1 || got[0] != "raw" {
		t.Fatalf("format = %v", got)
	}
	if q.Get("format") != "raw" || len(q) != 1 {
		t.Fatalf("caller's query was modified: %v", q)
	}
}

// strictDecode decodes v with unknown fields rejected, so the SDK structs must cover every
// field the backend emits.
func strictDecode(t *testing.T, data []byte, v any) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("strict decode into %T: %v", v, err)
	}
}

func TestSimpleFixturesMatchStructs(t *testing.T) {
	pages := map[string]func(t *testing.T, data []byte){
		"tweets": func(t *testing.T, d []byte) {
			var v struct {
				SimplePage[SimpleTweet]
			}
			strictDecode(t, d, &v)
		},
		"users": func(t *testing.T, d []byte) {
			var v struct {
				SimplePage[SimpleUser]
			}
			strictDecode(t, d, &v)
		},
		"communities": func(t *testing.T, d []byte) {
			var v struct {
				SimplePage[SimpleCommunity]
			}
			strictDecode(t, d, &v)
		},
		"trends": func(t *testing.T, d []byte) {
			var v struct {
				SimplePage[SimpleTrend]
			}
			strictDecode(t, d, &v)
		},
		"locations": func(t *testing.T, d []byte) {
			var v struct {
				SimplePage[SimpleTrendLocation]
			}
			strictDecode(t, d, &v)
		},
	}
	single := func(v any) func(t *testing.T, d []byte) {
		return func(t *testing.T, d []byte) {
			strictDecode(t, d, &struct {
				Data any `json:"data"`
			}{Data: v})
		}
	}
	cases := map[string]func(t *testing.T, data []byte){
		"search":                                 pages["tweets"],
		"tweet-tweetid":                          pages["tweets"],
		"tweet-tweetid-article":                  pages["tweets"],
		"tweet-tweetid-quotes":                   pages["tweets"],
		"user-userid-tweets":                     pages["tweets"],
		"user-userid-tweetsandreplies":           pages["tweets"],
		"user-userid-media":                      pages["tweets"],
		"user-userid-likes":                      pages["tweets"],
		"user-userid-articles":                   pages["tweets"],
		"communities-communityid-tweets":         pages["tweets"],
		"lists-tweets":                           pages["tweets"],
		"tweet-tweets-by-ids":                    pages["tweets"],
		"tweet-tweetid-favorites":                pages["users"],
		"tweet-tweetid-retweets":                 pages["users"],
		"user-userid-followers":                  pages["users"],
		"user-userid-followings":                 pages["users"],
		"user-userid-verifiedfollowers":          pages["users"],
		"user-users-by-ids":                      pages["users"],
		"communities-communityid-members":        pages["users"],
		"communities-communityid-moderators":     pages["users"],
		"communities-communityid-members-search": pages["users"],
		"lists-members":                          pages["users"],
		"lists-followers":                        pages["users"],
		"communities-search-searchterm":          pages["communities"],
		"trends-locationid":                      pages["trends"],
		"misc-locations":                         pages["locations"],
		"simpletweet-tweetid":                    single(&SimpleTweet{}),
		"user-by-username-username":              single(&SimpleUser{}),
		"user-userid":                            single(&SimpleUser{}),
		"user-about-username":                    single(&SimpleUserAbout{}),
		"lists-details":                          single(&SimpleList{}),
		"communities-communityid":                single(&SimpleCommunity{}),
	}
	for name, check := range cases {
		t.Run(name, func(t *testing.T) { check(t, simpleFixture(t, name)) })
	}
}

func TestDecodeSimplePageTweets(t *testing.T) {
	page, err := DecodeSimplePage[SimpleTweet](simpleFixture(t, "user-userid-tweets"))
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasNextPage || page.NextCursor == nil || page.Next() != "DAAHCgABHUS5pXT__-wLAAIAAAATMjEwMjQ0NzU2NzA0MDgyMzc2MAgAAwAAAAIAAA" {
		t.Fatalf("cursor: has=%v next=%v", page.HasNextPage, page.NextCursor)
	}
	if page.PreviousCursor == nil || *page.PreviousCursor == "" {
		t.Fatal("previousCursor missing")
	}
	if len(page.Data) != 1 {
		t.Fatalf("len = %d", len(page.Data))
	}
	tw := page.Data[0]
	if tw.ID != "2019881223666233717" || tw.Author == nil || tw.Author.Username != "XDevelopers" || !tw.IsPinned {
		t.Fatalf("tweet = %+v", tw)
	}
	if tw.Counts.Likes != 5599 || tw.Counts.Views == nil || *tw.Counts.Views != 2552260 {
		t.Fatalf("counts = %+v", tw.Counts)
	}
	if tw.Media == nil || tw.Hashtags == nil || tw.Mentions == nil {
		t.Fatal("lists should decode as empty, not nil")
	}
	if tw.Quoted != nil || tw.RetweetOf != nil || tw.ReplyTo != nil {
		t.Fatal("quoted/retweetOf/replyTo should be nil")
	}
}

func TestDecodeSimplePageReplyAndMedia(t *testing.T) {
	page, err := DecodeSimplePage[SimpleTweet](simpleFixture(t, "user-userid-tweetsandreplies"))
	if err != nil {
		t.Fatal(err)
	}
	r := page.Data[1].ReplyTo
	if r == nil || r.TweetID == nil || *r.TweetID != "2000000000000000001" || r.Username == nil || *r.Username != "example_user_1" {
		t.Fatalf("replyTo = %+v", r)
	}
	media, err := DecodeSimplePage[SimpleTweet](simpleFixture(t, "user-userid-media"))
	if err != nil {
		t.Fatal(err)
	}
	if m := media.Data[0].Media; len(m) != 1 || m[0].Type == "" || m[0].URL == "" {
		t.Fatalf("media = %+v", m)
	}
	quotes, err := DecodeSimplePage[SimpleTweet](simpleFixture(t, "tweet-tweetid-quotes"))
	if err != nil {
		t.Fatal(err)
	}
	if q := quotes.Data[0].Quoted; q == nil || q.ID == "" {
		t.Fatalf("quoted = %+v", q)
	}
}

func TestDecodeSimplePageLastPage(t *testing.T) {
	page, err := DecodeSimplePage[SimpleUser](simpleFixture(t, "tweet-tweetid-favorites"))
	if err != nil {
		t.Fatal(err)
	}
	if page.HasNextPage || page.NextCursor != nil || page.Next() != "" || page.Data == nil || len(page.Data) != 0 {
		t.Fatalf("page = %+v", page)
	}
}

func TestDecodeSimplePageUsersAndTrends(t *testing.T) {
	users, err := DecodeSimplePage[SimpleUser](simpleFixture(t, "user-userid-followers"))
	if err != nil {
		t.Fatal(err)
	}
	if len(users.Data) != 2 || users.Data[0].Username != "example_user_1" || users.Data[0].Counts.Followers != 129088 || !users.Data[0].Verified {
		t.Fatalf("users = %+v", users.Data)
	}
	trends, err := DecodeSimplePage[SimpleTrend](simpleFixture(t, "trends-locationid"))
	if err != nil {
		t.Fatal(err)
	}
	if trends.Data[0].Query != "#GoLang" || trends.Data[0].TweetVolume != nil || *trends.Data[1].TweetVolume != 15342 {
		t.Fatalf("trends = %+v", trends.Data)
	}
}

func TestDecodeSimpleSingle(t *testing.T) {
	user, err := DecodeSimple[SimpleUser](simpleFixture(t, "user-by-username-username"))
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "2244994945" || user.Username != "XDevelopers" || user.URL == nil || *user.URL != "https://docs.x.com/" {
		t.Fatalf("user = %+v", user)
	}
	about, err := DecodeSimple[SimpleUserAbout](simpleFixture(t, "user-about-username"))
	if err != nil {
		t.Fatal(err)
	}
	if about.Username != "XDevelopers" || about.About.AccountBasedIn == nil || *about.About.AccountBasedIn != "United States" || about.About.UsernameChanges != 1 {
		t.Fatalf("about = %+v", about)
	}
	list, err := DecodeSimple[SimpleList](simpleFixture(t, "lists-details"))
	if err != nil {
		t.Fatal(err)
	}
	if list.ID != "84839422" || list.MemberCount != 81 || list.Owner == nil || list.Owner.Username != "X" {
		t.Fatalf("list = %+v", list)
	}
	missing, err := DecodeSimple[SimpleUser]([]byte(`{"data":null}`))
	if err != nil || missing != nil {
		t.Fatalf("null data: %v %v", missing, err)
	}
}

func TestDecodeSimpleRejectsOtherShapes(t *testing.T) {
	if _, err := DecodeSimplePage[SimpleTweet]([]byte(liveFixture(t, "user-userid-tweets"))); !errors.Is(err, ErrNotSimpleFormat) {
		t.Fatalf("raw body: want ErrNotSimpleFormat, got %v", err)
	}
	if _, err := DecodeSimple[SimpleUser](simpleFixture(t, "user-userid-followers")); !errors.Is(err, ErrNotSimpleFormat) {
		t.Fatalf("page as single: want ErrNotSimpleFormat, got %v", err)
	}
	if _, err := DecodeSimplePage[SimpleTweet]([]byte(`not json`)); err == nil {
		t.Fatal("want error for invalid JSON")
	}
}

func TestUnavailablePlaceholder(t *testing.T) {
	body := `{"data":[{"id":"1","unavailable":true},{"id":"2","url":"u","text":"t","quoted":{"id":"3","unavailable":true},"counts":{"likes":1}}],"nextCursor":null,"previousCursor":null,"hasNextPage":false}`
	page, err := DecodeSimplePage[SimpleTweet]([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if !page.Data[0].Unavailable || page.Data[0].ID != "1" {
		t.Fatalf("placeholder = %+v", page.Data[0])
	}
	if page.Data[1].Unavailable || page.Data[1].Quoted == nil || !page.Data[1].Quoted.Unavailable || page.Data[1].Quoted.ID != "3" {
		t.Fatalf("quoted placeholder = %+v", page.Data[1])
	}
}

func TestFetchSimple(t *testing.T) {
	srv, rec := newServer(t, response{status: 200, body: string(simpleFixture(t, "user-by-username-username"))})
	c, _ := newTestClient(t, srv) // raw default: FetchSimple forces the simple format
	user, err := FetchSimple[SimpleUser](context.Background(), c.Users.GetByUsername, "@XDevelopers")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "XDevelopers" {
		t.Fatalf("user = %+v", user)
	}
	r := rec.last(t)
	if r.URL.Path != "/user/by/username/XDevelopers" || r.URL.Query().Get("format") != "simple" {
		t.Fatalf("request = %s", r.URL)
	}
}

func TestFetchSimplePage(t *testing.T) {
	srv, rec := newServer(t, response{status: 200, body: string(simpleFixture(t, "user-userid-tweets"))})
	c, _ := newTestClient(t, srv)
	page, err := FetchSimplePage[SimpleTweet](context.Background(), c.Users.Tweets, UserTimelineParams{UserID: "2244994945", Count: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || !page.HasNextPage {
		t.Fatalf("page = %+v", page)
	}
	q := rec.last(t).URL.Query()
	if q.Get("format") != "simple" || q.Get("count") != "20" {
		t.Fatalf("query = %v", q)
	}
}

func TestFetchSimplePropagatesErrors(t *testing.T) {
	srv, _ := newServer(t, response{status: 404, body: `{"error":"not found"}`})
	c, _ := newTestClient(t, srv)
	_, err := FetchSimple[SimpleUser](context.Background(), c.Users.GetByID, "12")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want NotFoundError, got %v", err)
	}
	if _, err := FetchSimple[SimpleUser](context.Background(), c.Users.GetByID, "abc"); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("want ErrInvalidParams, got %v", err)
	}
}

func simpleTweetPage(t *testing.T, ids []string, next string) string {
	t.Helper()
	page := SimplePage[SimpleTweet]{Data: []SimpleTweet{}}
	for _, id := range ids {
		page.Data = append(page.Data, SimpleTweet{ID: id})
	}
	if next != "" {
		page.NextCursor = &next
		page.HasNextPage = true
	}
	b, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPaginateSimpleFollowsNextCursor(t *testing.T) {
	srv, rec := newServer(t,
		response{status: 200, body: simpleTweetPage(t, []string{"1", "2"}, "C2")},
		response{status: 200, body: simpleTweetPage(t, []string{"3"}, "C3")},
		response{status: 200, body: simpleTweetPage(t, []string{"4"}, "")},
		response{status: 500, body: `{}`},
	)
	c, _ := newTestClient(t, srv)
	it := PaginateSimple[SimpleTweet](context.Background(), c.Users.Tweets, UserTimelineParams{UserID: "12"}, nil)
	var ids []string
	for it.Next() {
		for _, tw := range it.Page().Data {
			ids = append(ids, tw.ID)
		}
	}
	if err := it.Err(); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 4 || ids[3] != "4" {
		t.Fatalf("ids = %v", ids)
	}
	reqs := rec.all()
	if len(reqs) != 3 {
		t.Fatalf("requests = %d", len(reqs))
	}
	wantCursors := []string{"", "C2", "C3"}
	for i, r := range reqs {
		q := r.URL.Query()
		if q.Get("format") != "simple" || q.Get("cursor") != wantCursors[i] {
			t.Fatalf("request %d query = %v", i, q)
		}
	}
	if it.Raw().Number != 3 || it.Raw().NextCursor != "" {
		t.Fatalf("raw page = %+v", it.Raw())
	}
}

func TestPaginateSimpleStopsWhenHasNextPageFalse(t *testing.T) {
	// hasNextPage false wins even if a cursor is present.
	body := `{"data":[{"id":"1"}],"nextCursor":"C2","previousCursor":null,"hasNextPage":false}`
	srv, rec := newServer(t, response{status: 200, body: body})
	c, _ := newTestClient(t, srv)
	pages := 0
	err := PaginateSimple[SimpleTweet](context.Background(), c.Search.Tweets, SearchParams{Query: "go"}, nil).
		ForEach(func(p *SimplePage[SimpleTweet]) error { pages++; return nil })
	if err != nil || pages != 1 || len(rec.all()) != 1 {
		t.Fatalf("pages=%d requests=%d err=%v", pages, len(rec.all()), err)
	}
}

func TestPaginateSimpleMaxPagesAndRepeatedCursor(t *testing.T) {
	srv, rec := newServer(t, response{status: 200, body: simpleTweetPage(t, []string{"1"}, "SAME")})
	c, _ := newTestClient(t, srv)
	it := PaginateSimple[SimpleTweet](context.Background(), c.Users.Tweets, UserTimelineParams{UserID: "12"}, &PaginateOptions{MaxPages: NoPageLimit})
	n := 0
	for it.Next() {
		n++
	}
	// Page 1 (no cursor), page 2 (SAME), then SAME repeats.
	if n != 2 || len(rec.all()) != 2 || it.Err() != nil {
		t.Fatalf("pages=%d requests=%d err=%v", n, len(rec.all()), it.Err())
	}
}

func TestPaginateSimpleDecodeError(t *testing.T) {
	srv, _ := newServer(t, response{status: 200, body: liveFixture(t, "user-userid-tweets")})
	c, _ := newTestClient(t, srv)
	it := PaginateSimple[SimpleTweet](context.Background(), c.Users.Tweets, UserTimelineParams{UserID: "12"}, nil)
	if it.Next() {
		t.Fatal("Next should fail on a raw body")
	}
	if !errors.Is(it.Err(), ErrNotSimpleFormat) {
		t.Fatalf("err = %v", it.Err())
	}
}

func TestPaginateRawIteratorUnderstandsSimplePages(t *testing.T) {
	srv, rec := newServer(t,
		response{status: 200, body: string(simpleFixture(t, "user-userid-followers"))},
		response{status: 200, body: string(simpleFixture(t, "tweet-tweetid-favorites"))},
	)
	c, _ := newTestClient(t, srv, WithFormat(FormatSimple))
	it := Paginate(context.Background(), c.Users.Followers, UserTimelineParams{UserID: "12"}, nil)
	var nexts []string
	for it.Next() {
		nexts = append(nexts, it.Page().NextCursor)
	}
	if it.Err() != nil || len(nexts) != 2 || nexts[1] != "" {
		t.Fatalf("nexts=%v err=%v", nexts, it.Err())
	}
	reqs := rec.all()
	if reqs[1].URL.Query().Get("cursor") != nexts[0] || reqs[1].URL.Query().Get("format") != "simple" {
		t.Fatalf("second request = %s", reqs[1].URL)
	}
}

func TestSimpleFormatNewCollections(t *testing.T) {
	ctx := context.Background()

	srv, rec := newServer(t, response{status: 200, body: string(simpleFixture(t, "user-userid-followers"))})
	c, _ := newTestClient(t, srv)
	users, err := FetchSimplePage[SimpleUser](ctx, c.Users.Affiliates, UserTimelineParams{Username: "XDevelopers"})
	if err != nil {
		t.Fatal(err)
	}
	if len(users.Data) == 0 || users.Data[0].Username == "" {
		t.Fatalf("affiliates = %+v", users)
	}
	if r := rec.last(t); r.URL.Path != "/user/-1/affiliates" || r.URL.Query().Get("format") != "simple" {
		t.Fatalf("request = %s", r.URL)
	}

	srv, rec = newServer(t,
		response{status: 200, body: simpleTweetPage(t, []string{"21", "22"}, "R2")},
		response{status: 200, body: simpleTweetPage(t, []string{"23"}, "")},
	)
	c, _ = newTestClient(t, srv)
	it := PaginateSimple[SimpleTweet](ctx, c.Tweets.Replies, RepliesParams{TweetID: "20", RankingMode: RankingRecency}, nil)
	var ids []string
	for it.Next() {
		for _, tw := range it.Page().Data {
			ids = append(ids, tw.ID)
		}
	}
	if it.Err() != nil || strings.Join(ids, ",") != "21,22,23" {
		t.Fatalf("replies ids=%v err=%v", ids, it.Err())
	}
	if q := rec.last(t).URL.Query(); q.Get("cursor") != "R2" || q.Get("rankingMode") != "Recency" || q.Get("format") != "simple" {
		t.Fatalf("replies query = %v", q)
	}

	srv, rec = newServer(t, response{status: 200, body: simpleTweetPage(t, []string{"19", "20", "24"}, "")})
	c, _ = newTestClient(t, srv)
	thread, err := FetchSimplePage[SimpleTweet](ctx, c.Tweets.Thread, ThreadParams{TweetID: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if len(thread.Data) != 3 || thread.HasNextPage || thread.Next() != "" {
		t.Fatalf("thread = %+v", thread)
	}
	if r := rec.last(t); r.URL.Path != "/tweet/20/thread" || r.URL.Query().Get("format") != "simple" {
		t.Fatalf("request = %s", r.URL)
	}
}

func TestNewPublicReadsAreRetried(t *testing.T) {
	ctx := context.Background()
	for name, call := range map[string]func(c *Client) (json.RawMessage, error){
		"Spaces.Get":             func(c *Client) (json.RawMessage, error) { return c.Spaces.Get(ctx, "1YqKDqWqdPLsV") },
		"Communities.About":      func(c *Client) (json.RawMessage, error) { return c.Communities.About(ctx, "1500") },
		"Users.FriendshipsBatch": func(c *Client) (json.RawMessage, error) { return c.Users.FriendshipsBatch(ctx, "a", []string{"b"}) },
		"Users.AboutBatch":       func(c *Client) (json.RawMessage, error) { return c.Users.AboutBatch(ctx, []string{"a"}) },
	} {
		srv, rec := newServer(t, response{status: 503, body: `{"error":"busy"}`}, response{status: 200, body: `{}`})
		c, _ := newTestClient(t, srv)
		if _, err := call(c); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if n := len(rec.all()); n != 2 {
			t.Errorf("%s: %d requests, want 2", name, n)
		}
	}
}

// Tweets.GetMany: the raw answer has one entry per requested ID ({} for an ID without a
// tweet); the simple answer leaves those IDs out and keeps request order.
func TestTweetsByIDsFixtures(t *testing.T) {
	var raw struct {
		Data struct {
			TweetResult []map[string]json.RawMessage `json:"tweetResult"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(liveFixture(t, "tweet-tweets-by-ids")), &raw); err != nil {
		t.Fatal(err)
	}
	entries := raw.Data.TweetResult
	if len(entries) != 3 || len(entries[2]) != 0 || entries[0]["result"] == nil {
		t.Fatalf("raw entries = %d, last = %v", len(entries), entries[len(entries)-1])
	}

	page, err := DecodeSimplePage[SimpleTweet](simpleFixture(t, "tweet-tweets-by-ids"))
	if err != nil {
		t.Fatal(err)
	}
	if page.HasNextPage || page.Next() != "" {
		t.Fatalf("cursor: has=%v next=%q", page.HasNextPage, page.Next())
	}
	var ids []string
	for _, tw := range page.Data {
		ids = append(ids, tw.ID)
	}
	if strings.Join(ids, ",") != "2019881223666233717,20" {
		t.Fatalf("ids = %v", ids)
	}
}
