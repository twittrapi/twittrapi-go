package twittrapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestExtractCursorsBottomAndTop(t *testing.T) {
	data := []byte(`{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[
		{"type":"TimelineAddEntries","entries":[
			{"entryId":"cursor-top-1","content":{"entryType":"TimelineTimelineCursor","cursorType":"Top","value":"TOP"}},
			{"entryId":"tweet-1","content":{"entryType":"TimelineTimelineItem"}},
			{"entryId":"cursor-bottom-1","content":{"entryType":"TimelineTimelineCursor","cursorType":"Bottom","value":"BOTTOM"}}
		]}
	]}}}}}}`)
	got := ExtractCursors(data)
	if got.Next != "BOTTOM" || got.Previous != "TOP" {
		t.Fatalf("got %+v", got)
	}
	if ExtractBottomCursor(data) != "BOTTOM" {
		t.Fatal("ExtractBottomCursor mismatch")
	}
}

func TestExtractCursorsReplaceEntry(t *testing.T) {
	data := []byte(`{"instructions":[
		{"type":"TimelineAddEntries","entries":[{"entryId":"tweet-9","content":{}}]},
		{"type":"TimelineReplaceEntry","entry_id_to_replace":"cursor-bottom-0",
		 "entry":{"entryId":"cursor-bottom-2","content":{"cursorType":"Bottom","value":"NEXT2"}}}
	]}`)
	if got := ExtractBottomCursor(data); got != "NEXT2" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractCursorsItemContent(t *testing.T) {
	// entryId match with the value nested in itemContent and no cursorType next to it.
	data := []byte(`{"entries":[{"entryId":"cursor-bottom-77","content":{"itemContent":{"value":"IC"}}}]}`)
	if got := ExtractBottomCursor(data); got != "IC" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractCursorsSnakeCaseType(t *testing.T) {
	data := []byte(`{"x":{"cursor_type":"Bottom","value":"SNAKE"}}`)
	if got := ExtractBottomCursor(data); got != "SNAKE" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractCursorsFirstInDocumentOrder(t *testing.T) {
	// Keys are deliberately not in alphabetical order; the first in the document wins.
	data := []byte(`{"zeta":{"cursorType":"Bottom","value":"FIRST"},"alpha":{"cursorType":"Bottom","value":"SECOND"}}`)
	for i := 0; i < 20; i++ {
		if got := ExtractBottomCursor(data); got != "FIRST" {
			t.Fatalf("got %q", got)
		}
	}
}

func TestExtractCursorsFallback(t *testing.T) {
	cases := map[string]string{
		`{"next_cursor":"N1"}`:         "N1",
		`{"meta":{"nextCursor":"N2"}}`: "N2",
		`{"next_cursor":"0"}`:          "",
		`{"next_cursor":""}`:           "",
		`{"next_cursor":12345}`:        "",
		`{"next_cursor":"F","a":{"cursorType":"Bottom","value":"B"}}`: "B", // typed cursor beats fallback
		`[]`:                                 "",
		`null`:                               "",
		`not json`:                           "",
		`{"cursorType":"Bottom","value":""}`: "",
		`{"cursorType":"Bottom","value":5}`:  "",
	}
	for in, want := range cases {
		if got := ExtractBottomCursor([]byte(in)); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}

func TestCountTimelineEntries(t *testing.T) {
	data := []byte(`{"a":[{"entryId":"tweet-1"},{"entryId":"user-2","content":{"entryId":"nested-ignored"}},
		{"entryId":"cursor-top-1"},{"entryId":"cursor-bottom-1"}],"b":{"entryId":5}}`)
	items, cursors := CountTimelineEntries(data)
	if items != 2 || cursors != 2 {
		t.Fatalf("items=%d cursors=%d", items, cursors)
	}
	if i, c := CountTimelineEntries([]byte(`{bad`)); i != 0 || c != 0 {
		t.Fatal("invalid JSON should count zero")
	}
}

// timelinePage builds a fake timeline page with n items and an optional bottom cursor.
func timelinePage(n int, next string) string {
	var entries []string
	for i := 0; i < n; i++ {
		entries = append(entries, fmt.Sprintf(`{"entryId":"tweet-%d","content":{}}`, i))
	}
	if next != "" {
		entries = append(entries, fmt.Sprintf(`{"entryId":"cursor-bottom-x","content":{"cursorType":"Bottom","value":%q}}`, next))
	}
	return `{"instructions":[{"entries":[` + strings.Join(entries, ",") + `]}]}`
}

// pagedServer serves pages keyed by the cursor query parameter ("" is the first page).
func pagedServer(t *testing.T, pages map[string]string) (*Client, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var cursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		mu.Lock()
		cursors = append(cursors, cursor)
		mu.Unlock()
		body, ok := pages[cursor]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"unknown cursor"}`))
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, _ := newTestClient(t, srv)
	return c, &cursors
}

func collect(t *testing.T, it *PageIterator) []Page {
	t.Helper()
	var pages []Page
	for it.Next() {
		pages = append(pages, it.Page())
	}
	return pages
}

func TestPaginateFollowsCursorUntilMissing(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"":   timelinePage(2, "c1"),
		"c1": timelinePage(2, "c2"),
		"c2": timelinePage(1, ""),
	})
	it := Paginate(context.Background(), c.Users.Followers, UserTimelineParams{Username: "jack", Count: 100}, nil)
	pages := collect(t, it)
	if err := it.Err(); err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("pages = %d", len(pages))
	}
	if strings.Join(*cursors, "|") != "|c1|c2" {
		t.Fatalf("cursors = %q", *cursors)
	}
	for i, p := range pages {
		if p.Number != i+1 {
			t.Errorf("page %d has Number %d", i, p.Number)
		}
	}
	if pages[0].Cursor != "" || pages[0].NextCursor != "c1" || pages[2].Cursor != "c2" || pages[2].NextCursor != "" {
		t.Fatalf("cursors on pages: %+v", pages)
	}
	if it.Next() {
		t.Fatal("Next after end returned true")
	}
}

func TestPaginateDefaultMaxPages(t *testing.T) {
	pages := map[string]string{}
	for i := 0; i < 20; i++ {
		key := ""
		if i > 0 {
			key = fmt.Sprintf("c%d", i)
		}
		pages[key] = timelinePage(1, fmt.Sprintf("c%d", i+1))
	}
	c, cursors := pagedServer(t, pages)
	got := collect(t, Paginate(context.Background(), c.Search.Tweets, SearchParams{Query: "go"}, nil))
	if len(got) != DefaultMaxPages || len(*cursors) != DefaultMaxPages {
		t.Fatalf("pages=%d requests=%d", len(got), len(*cursors))
	}

	*cursors = nil
	got = collect(t, Paginate(context.Background(), c.Search.Tweets, SearchParams{Query: "go"}, &PaginateOptions{MaxPages: 3}))
	if len(got) != 3 || len(*cursors) != 3 {
		t.Fatalf("MaxPages 3: pages=%d requests=%d", len(got), len(*cursors))
	}

	*cursors = nil
	it := Paginate(context.Background(), c.Search.Tweets, SearchParams{Query: "go"}, &PaginateOptions{MaxPages: NoPageLimit})
	got = collect(t, it)
	// Pages run out at c20, which the server does not know: the 21st request fails with 404.
	var nf *NotFoundError
	if len(got) != 20 || !errors.As(it.Err(), &nf) {
		t.Fatalf("NoPageLimit: pages=%d err=%v", len(got), it.Err())
	}
}

func TestPaginateStopsOnRepeatedCursor(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"":   timelinePage(1, "c1"),
		"c1": timelinePage(1, "c2"),
		"c2": timelinePage(1, "c1"),
	})
	got := collect(t, Paginate(context.Background(), c.Lists.Tweets, ListPageParams{ListID: "1"}, nil))
	if len(got) != 3 || strings.Join(*cursors, "|") != "|c1|c2" {
		t.Fatalf("pages=%d cursors=%q", len(got), *cursors)
	}
}

func TestPaginateStopsWhenCursorEqualsStart(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"start": timelinePage(1, "start"),
	})
	got := collect(t, Paginate(context.Background(), c.Tweets.Quotes, TweetPageParams{TweetID: "20", Cursor: "start"}, nil))
	if len(got) != 1 || got[0].Cursor != "start" || strings.Join(*cursors, "|") != "start" {
		t.Fatalf("pages=%d cursors=%q", len(got), *cursors)
	}
}

func TestPaginateStopsOnCursorOnlyPage(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"":   timelinePage(3, "c1"),
		"c1": timelinePage(0, "c2"), // only cursor entries: no more results
		"c2": timelinePage(3, "c3"),
	})
	got := collect(t, Paginate(context.Background(), c.Communities.Tweets, CommunityPageParams{CommunityID: "5"}, nil))
	if len(got) != 2 || len(*cursors) != 2 {
		t.Fatalf("pages=%d requests=%d", len(got), len(*cursors))
	}
}

func TestPaginateNonTimelinePageWithCursorContinues(t *testing.T) {
	// No entries at all (plain next_cursor API shape): keep going while the cursor changes.
	c, cursors := pagedServer(t, map[string]string{
		"":   `{"users":[1],"next_cursor":"n1"}`,
		"n1": `{"users":[2],"next_cursor":"0"}`,
	})
	got := collect(t, Paginate(context.Background(), c.Communities.Search, CommunitySearchParams{Query: "go"}, nil))
	if len(got) != 2 || len(*cursors) != 2 {
		t.Fatalf("pages=%d requests=%d", len(got), len(*cursors))
	}
}

func TestPaginateError(t *testing.T) {
	c, _ := pagedServer(t, map[string]string{
		"": timelinePage(1, "missing"),
	})
	it := Paginate(context.Background(), c.Users.Tweets, UserTimelineParams{UserID: "12"}, nil)
	got := collect(t, it)
	var nf *NotFoundError
	if len(got) != 1 || !errors.As(it.Err(), &nf) {
		t.Fatalf("pages=%d err=%v", len(got), it.Err())
	}
}

func TestPaginateInvalidParams(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{})
	it := Paginate(context.Background(), c.Users.Tweets, UserTimelineParams{}, nil)
	if it.Next() || !errors.Is(it.Err(), ErrInvalidParams) || len(*cursors) != 0 {
		t.Fatalf("err=%v requests=%d", it.Err(), len(*cursors))
	}
}

func TestPaginatePassesParamsThrough(t *testing.T) {
	var seen []UserTimelineParams
	fn := func(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
		seen = append(seen, p)
		if p.Cursor == "" {
			return json.RawMessage(timelinePage(1, "next")), nil
		}
		return json.RawMessage(timelinePage(1, "")), nil
	}
	it := Paginate(context.Background(), fn, UserTimelineParams{Username: "jack", Count: 40}, nil)
	if n := len(collect(t, it)); n != 2 {
		t.Fatalf("pages = %d", n)
	}
	want := []UserTimelineParams{{Username: "jack", Count: 40}, {Username: "jack", Count: 40, Cursor: "next"}}
	if fmt.Sprint(seen) != fmt.Sprint(want) {
		t.Fatalf("params = %+v", seen)
	}
}

func TestForEach(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"":   timelinePage(1, "c1"),
		"c1": timelinePage(1, "c2"),
		"c2": timelinePage(1, ""),
	})
	n := 0
	err := Paginate(context.Background(), c.Tweets.Get, TweetParams{TweetID: "20"}, nil).ForEach(func(p Page) error {
		n++
		return nil
	})
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}

	*cursors = nil
	stop := errors.New("stop")
	it := Paginate(context.Background(), c.Tweets.Get, TweetParams{TweetID: "20"}, nil)
	err = it.ForEach(func(p Page) error { return stop })
	if !errors.Is(err, stop) || len(*cursors) != 1 || it.Next() {
		t.Fatalf("err=%v requests=%d", err, len(*cursors))
	}
}
