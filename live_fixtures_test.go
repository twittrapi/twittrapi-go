package twittrapi

// Tests against trimmed real TwittrAPI responses in testdata/live (captured 2026-10-10;
// first content entries plus all cursor entries kept, structure unchanged).

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func liveFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "live", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// items counts top-level timeline entries (module items inside an entry are not counted).
func TestLiveFixturesCursors(t *testing.T) {
	cases := []struct {
		file           string
		next, previous string
		items, cursors int
	}{
		{"communities-communityid-members", "CycRAAAAAAwABAwKmwzBVQo+vRYlx8zZ15ACCm07AAABjezzdesADJorCj69f/////////8KbTt//////////wAIP8AAAAACAAAA", "", 0, 0},
		{"communities-communityid-moderators", "", "", 0, 0},
		{"communities-communityid-tweets", "DAAFCgABHUS5xm5__-oLAAIAAADwRW1QQzZ3QUFBZlEvZ0dKTjB2R3AvQUFBQUJRZFExYXloNWJ3Y3gxQ0Y2MGtHbEJ5SFVNNVVEbFdvTmNkUWFwdWpWdkJVeDFFVHd1N0d0QUhIVUg1blJFWGtIc2RRZm1RZXBxQmh4MUVxcm9QbTlBNUhVU1VNMUVhNEdvZFFON1dsRnNoTkIxQ0o1cjJGckVySFVOMVlwSGJzUVlkUTN2Ym14WXhzUjFCRUlBZzJrR2VIVU1ueVZOV0VYc2RRTkxYaGRvUUJSMUF5a2s4MjFBS0hVQ2puekhXb1dnZFFNNGxwTnRBNmgxRUl3U0IyeUhmCAADAAAAAgAA", "DAAJAAA", 1, 2},
		{"communities-search-searchterm", "WzExMy41ODc5MSwxODk3OTI3NDQxNDQxNTEzNzE4XQ==", "", 0, 0},
		{"jobs-search-searchterm", "WzMuNDcwMjg4OCwiMCIsIjE4MTMwMzEwNDExNTE3Mzc4NTgiXQ==", "", 0, 0},
		{"lists-followers", "1000000000000000000|2109016363606474730", "-1|2109016363606474753", 2, 2},
		{"lists-members", "1000000000000000000|2109016359324090346", "-1|2109016359324090369", 2, 2},
		{"lists-tweets", "DAABCgABHUS7Mid__58KAAIdQPHo7lqQcQgAAwAAAAIAAA", "DAABCgABHUS7MieAJxEKAAIdRI-M3ZvQJQgAAwAAAAEAAA", 1, 2},
		{"search", "DAADDAABCgABHUS5rRGX0YcKAAIdRLkhaRohfwAIAAIAAAACCAADAAAAAAgABAAAAAAKAAUdRLmzyoAnEAoABh1EubPKf9jwCwAHAAAA8EVtUEM2d0FBQWZRL2dHSk4wdkdwL0FBQUFCUWRSTGszZmRidzZCMUV1U2M0VnJGWkhVUzVJV2thSVg4ZFJMbHdOTmR3ZngxRXVTaEMydkVmSFVTNWRTeVhJWU1kUkxsSjVSZUFGUjFFdVc1eFc5R1hIVVM1UjQ4YXdLMGRSTGxoV05kZzJSMUV1YTBSbDlHSEhVUzVkVTFia1MwZFJMazJvSmRSb1IxRXVVRURXc0FPSFVTNWZoQWFvU3NkUkxsbVJwYlF0aDFFdVpRdVdrR2xIVVM1cWI1YkVHTWRSTG1GTmxjUXFCMUV1VlhtMXZHcgAA", "DAADDAABCgABHUS5rRGX0YcKAAIdRLkhaRohfwAIAAIAAAABCAADAAAAAAgABAAAAAAKAAUdRLmzyoAnEAoABh1EubPKf9jwCwAHAAAA8EVtUEM2d0FBQWZRL2dHSk4wdkdwL0FBQUFCUWRSTGszZmRidzZCMUV1U2M0VnJGWkhVUzVJV2thSVg4ZFJMbHdOTmR3ZngxRXVTaEMydkVmSFVTNWRTeVhJWU1kUkxsSjVSZUFGUjFFdVc1eFc5R1hIVVM1UjQ4YXdLMGRSTGxoV05kZzJSMUV1YTBSbDlHSEhVUzVkVTFia1MwZFJMazJvSmRSb1IxRXVVRURXc0FPSFVTNWZoQWFvU3NkUkxsbVJwYlF0aDFFdVpRdVdrR2xIVVM1cWI1YkVHTWRSTG1GTmxjUXFCMUV1VlhtMXZHcgAA", 1, 2},
		{"tweet-tweetid-article", "DAAKCgABHUS5vqW__qkLAAIAAAGoRW1QQzZ3QUFBZlEvZ0dKTjB2R3AvQUFBQUNVY0NSME1jcHB3Ymh3SVJYV2dsc0JPSEFpQW5USWFNTFVjQ0E5R2NWcHgxeHdJRDBZb1ZvRDBIQWdlT0QrYWNQSWNDQ3VkYk5weFBCd0lMZDFXMXBDSUhBaUdnTTRhY0dRY0NHYVhWaGZSTEJ3STBDV1FHekZHSEFrQ0lDNlh3YzhjQ0E5RjI1cVFyaHdJV3F4Sm0wRWFIQWh4bHBkYWNYa2NDQlFCNU5hd3FSd0lmbkdrRnlHU0hBaHZBem1iTU8wY0NEUVBYZGR4b1J3SUhKZ3hWb0hCSEFnUFJXa2JjWFVjQ0hUZ2pOcHd1aHdKTXlsNlZpQzVIQWl0bHo0YndlUWNDSjY2ZFJweGFod0lobGlCbTJFMEhBaWFobkZYa0JNY0NqdWpqNW9ndEJ3SW1mUnhHaUVSSEFpbzEwWWJnSlFjQ0NaR0hWcHdtQndJa0tXQmxwQWFIQWgyVHYyYUFSMGNDS21wRDVleHF4d0lIZTd3bDZGd0hBaGZDMXNia1dVY0NJeWFQNVpoaFE9PQgAAwAAAAILAAQAAAAGQm90dG9tAAA", "", 2, 1},
		{"tweet-tweetid-favorites", "", "", 0, 0},
		{"tweet-tweetid-quotes", "DAACCgACHUS5wrfAJxAKAAMdRLnCt7_Y8AgABAAAAAILAAUAAADwRW1QQzZ3QUFBZlEvZ0dKTjB2R3AvQUFBQUJRY0NVbExSaHVCWXh3SUpyT3JsMUJqSEFndytqSFhrZFljQ0JGSzBkcHhGQndJRnNQbTF4RWNIQWhKRDNKYWNjRWNDR3ZaaXB2QkNSd0lnTVMzR3VIMkhBaSt2UEJiY0lzZFJEVVh2RnV3a3h3STU5UlpWb0NxSEF1QWZ5dlhVVHNjQ0x6eFBwcHhJQndKT01Ncm04RW5IQWlFcHRLV3NBWWNDTGNJZlphQm9Sd0laYU1nR25CakhBaTAyOC9XTVZrY0NJeUlBdGVoTUJ3SXUwMmJtbkRlCAAGAAAAAAgABwAAAAAMAAgKAAEcCBFK0dpxFAAAAA", "DAACCgACHUS5wrfAJxAKAAMdRLnCt7_Y8AgABAAAAAELAAUAAADwRW1QQzZ3QUFBZlEvZ0dKTjB2R3AvQUFBQUJRY0NVbExSaHVCWXh3SUpyT3JsMUJqSEFndytqSFhrZFljQ0JGSzBkcHhGQndJRnNQbTF4RWNIQWhKRDNKYWNjRWNDR3ZaaXB2QkNSd0lnTVMzR3VIMkhBaSt2UEJiY0lzZFJEVVh2RnV3a3h3STU5UlpWb0NxSEF1QWZ5dlhVVHNjQ0x6eFBwcHhJQndKT01Ncm04RW5IQWlFcHRLV3NBWWNDTGNJZlphQm9Sd0laYU1nR25CakhBaTAyOC9XTVZrY0NJeUlBdGVoTUJ3SXUwMmJtbkRlCAAGAAAAAAgABwAAAAAMAAgKAAEcCBFK0dpxFAAAAA", 1, 2},
		{"tweet-tweetid-retweets", "HBaAgIC+kLf1iDQAAA==", "HCaAgIDOzuXjkTQAAA==", 2, 2},
		{"tweet-tweetid", "DAAKCgABHUS5u-s__qkLAAIAAAGoRW1QQzZ3QUFBZlEvZ0dKTjB2R3AvQUFBQUNVY0NSME1jcHB3Ymh3SVJYV2dsc0JPSEFpQW5USWFNTFVjQ0E5R2NWcHgxeHdJRDBZb1ZvRDBIQWdlT0QrYWNQSWNDQ3VkYk5weFBCd0lMZDFXMXBDSUhBaUdnTTRhY0dRY0NHYVhWaGZSTEJ3STBDV1FHekZHSEFrQ0lDNlh3YzhjQ0E5RjI1cVFyaHdJV3F4Sm0wRWFIQWh4bHBkYWNYa2NDQlFCNU5hd3FSd0lmbkdrRnlHU0hBaHZBem1iTU8wY0NEUVBYZGR4b1J3SUhKZ3hWb0hCSEFnUFJXa2JjWFVjQ0hUZ2pOcHd1aHdKTXlsNlZpQzVIQWl0bHo0YndlUWNDSjY2ZFJweGFod0lobGlCbTJFMEhBaWFobkZYa0JNY0NqdWpqNW9ndEJ3SW1mUnhHaUVSSEFpbzEwWWJnSlFjQ0NaR0hWcHdtQndJa0tXQmxwQWFIQWgyVHYyYUFSMGNDS21wRDVleHF4d0lIZTd3bDZGd0hBaGZDMXNia1dVY0NJeWFQNVpoaFE9PQgAAwAAAAILAAQAAAAGQm90dG9tAAA", "", 2, 1},
		{"user-userid-articles", "DAABCgABHUS5rv1___sKAAIAAAAAAAAAAAgAAwAAAAIAAA", "DAAJAAA", 1, 2},
		{"user-userid-followers", "1000000000000000000|2109014666083565496", "-1|2109014666083565569", 2, 2},
		{"user-userid-followings", "1000000000000000000|2109014676376387522", "-1|2109014676376387585", 2, 2},
		{"user-userid-likes", "", "", 0, 0},
		{"user-userid-media", "DAABCgABHUS5qIC___0KAAIbOL6IfJeQgwgAAwAAAAIAAA", "DAABCgABHUS5qIDAJxEKAAIbOL6IfJeQgwgAAwAAAAEAAA", 1, 2},
		{"user-userid-tweets", "DAAHCgABHUS5pXT__-wLAAIAAAATMjEwMjQ0NzU2NzA0MDgyMzc2MAgAAwAAAAIAAA", "DAAHCgABHUS5pXUAJxELAAIAAAATMjEwODk2ODM2MDY5MjQwNDI2MQgAAwAAAAEAAA", 2, 2},
		{"user-userid-tweetsandreplies", "DAAHCgABHUS5pvk__-sLAAIAAAATMjEwMjU0ODUxMzE4NzgyNzc3MwgAAwAAAAIAAA", "DAAHCgABHUS5pvlAJxELAAIAAAATMjEwODk2ODM2MDY5MjQwNDI2MQgAAwAAAAEAAA", 2, 2},
		{"user-userid-verifiedfollowers", "1000000000000000000|2109014682063863786", "-1|2109014682063863809", 2, 2},
		{"user-usernames-status", "", "", 0, 0},
		{"user-usernames-to-ids", "", "", 0, 0},
		{"user-users-by-ids", "", "", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			data := []byte(liveFixture(t, tc.file))
			got := ExtractCursors(data)
			if got.Next != tc.next {
				t.Errorf("Next = %q, want %q", got.Next, tc.next)
			}
			if got.Previous != tc.previous {
				t.Errorf("Previous = %q, want %q", got.Previous, tc.previous)
			}
			if strings.HasPrefix(got.Next, "-1|") {
				t.Errorf("Next is a Top cursor: %q", got.Next)
			}
			items, cursors := CountTimelineEntries(data)
			if items != tc.items || cursors != tc.cursors {
				t.Errorf("CountTimelineEntries = %d, %d, want %d, %d", items, cursors, tc.items, tc.cursors)
			}
		})
	}
}

// emptyTimeline is a cursor-only page: X keeps returning a cursor after the last result.
const emptyTimeline = `{"data":{"user":{"result":{"timeline":{"timeline":{"instructions":[{"type":"TimelineAddEntries","entries":[
	{"entryId":"cursor-bottom-1","content":{"entryType":"TimelineTimelineCursor","cursorType":"Bottom","value":"1000000000000000000|1"}},
	{"entryId":"cursor-top-1","content":{"entryType":"TimelineTimelineCursor","cursorType":"Top","value":"-1|2"}}]}]}}}}}}`

func TestLivePaginateShapes(t *testing.T) {
	ctx := context.Background()
	followersNext := "1000000000000000000|2109014666083565496"
	tweetsNext := ExtractBottomCursor([]byte(liveFixture(t, "user-userid-tweets")))
	detailNext := ExtractBottomCursor([]byte(liveFixture(t, "tweet-tweetid")))
	searchNext := ExtractBottomCursor([]byte(liveFixture(t, "search")))
	commNext := "WzExMy41ODc5MSwxODk3OTI3NDQxNDQxNTEzNzE4XQ=="
	emptySlice := `{"data":{"communities_search_slice":{"items_results":[],"slice_info":{"next_cursor":"AGAIN"}}}}`

	cases := []struct {
		name         string
		pages        map[string]string
		run          func(*Client) *PageIterator
		wantRequests []string
	}{
		{"followers Bottom before Top, then cursor-only page", map[string]string{
			"": liveFixture(t, "user-userid-followers"), followersNext: emptyTimeline,
		}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Users.Followers, UserTimelineParams{UserID: "2244994945"}, nil)
		}, []string{"", followersNext}},
		{"user tweets Top before Bottom", map[string]string{
			"": liveFixture(t, "user-userid-tweets"), tweetsNext: liveFixture(t, "user-userid-likes"),
		}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Users.Tweets, UserTimelineParams{UserID: "2244994945"}, nil)
		}, []string{"", tweetsNext}},
		{"tweet detail", map[string]string{
			"": liveFixture(t, "tweet-tweetid"), detailNext: liveFixture(t, "tweet-tweetid-favorites"),
		}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Tweets.Get, TweetParams{TweetID: "2019881223666233717"}, nil)
		}, []string{"", detailNext}},
		{"search wrapper", map[string]string{
			"": liveFixture(t, "search"), searchNext: liveFixture(t, "search"),
		}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Search.Tweets, SearchParams{Query: "openai", Category: CategoryLatest}, nil)
		}, []string{"", searchNext}}, // second page repeats the cursor: stop
		{"community search slice, then empty slice", map[string]string{
			"": liveFixture(t, "communities-search-searchterm"), commNext: emptySlice,
		}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Communities.Search, CommunitySearchParams{Query: "openai"}, nil)
		}, []string{"", commNext}},
		{"empty likes timeline", map[string]string{"": liveFixture(t, "user-userid-likes")}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Users.Tweets, UserTimelineParams{UserID: "2244994945"}, &PaginateOptions{MaxPages: NoPageLimit})
		}, []string{""}},
		{"empty favorites timeline", map[string]string{"": liveFixture(t, "tweet-tweetid-favorites")}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Tweets.Likers, TweetPageParams{TweetID: "2019881223666233717"}, &PaginateOptions{MaxPages: NoPageLimit})
		}, []string{""}},
		{"articles", map[string]string{"": liveFixture(t, "user-userid-articles")}, func(c *Client) *PageIterator {
			return Paginate(ctx, c.Users.Tweets, UserTimelineParams{UserID: "2244994945"}, &PaginateOptions{MaxPages: 1})
		}, []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, requests := pagedServer(t, tc.pages)
			it := tc.run(c)
			pages := collect(t, it)
			if err := it.Err(); err != nil {
				t.Fatal(err)
			}
			if strings.Join(*requests, "\n") != strings.Join(tc.wantRequests, "\n") {
				t.Fatalf("requests = %q, want %q", *requests, tc.wantRequests)
			}
			if len(pages) != len(tc.wantRequests) {
				t.Fatalf("pages = %d, want %d", len(pages), len(tc.wantRequests))
			}
		})
	}
}

func TestCountTimelineEntriesModuleItems(t *testing.T) {
	// Later media pages add grid items with TimelineAddToModule; the entry list then only
	// holds cursors. Those module items are results, so pagination must continue.
	data := []byte(`{"instructions":[
		{"type":"TimelineAddToModule","moduleEntryId":"profile-grid-0","moduleItems":[{"entryId":"profile-grid-0-tweet-1","item":{}}]},
		{"type":"TimelineAddEntries","entries":[
			{"entryId":"cursor-top-1","content":{"cursorType":"Top","value":"T"}},
			{"entryId":"cursor-bottom-1","content":{"cursorType":"Bottom","value":"B"}}]}]}`)
	if items, cursors := CountTimelineEntries(data); items != 1 || cursors != 2 {
		t.Fatalf("got %d, %d", items, cursors)
	}
}
