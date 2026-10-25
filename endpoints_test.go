package twittrapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type endpointCase struct {
	name  string // "Service.Method ..." so coverage can be checked
	call  func(c *Client) (json.RawMessage, error)
	path  string // escaped path
	query url.Values
}

func endpointCases() []endpointCase {
	ctx := context.Background()
	return []endpointCase{
		// Users
		{"Users.GetByUsername", func(c *Client) (json.RawMessage, error) { return c.Users.GetByUsername(ctx, " @jack ") },
			"/user/by/username/jack", url.Values{}},
		{"Users.GetByUsername escapes", func(c *Client) (json.RawMessage, error) { return c.Users.GetByUsername(ctx, "a/b c") },
			"/user/by/username/a%2Fb%20c", url.Values{}},
		{"Users.GetByID", func(c *Client) (json.RawMessage, error) { return c.Users.GetByID(ctx, "12") },
			"/user/12", url.Values{}},
		{"Users.UsernameToID", func(c *Client) (json.RawMessage, error) { return c.Users.UsernameToID(ctx, "@jack") },
			"/username/to/id/jack", url.Values{}},
		{"Users.GetMany", func(c *Client) (json.RawMessage, error) {
			return c.Users.GetMany(ctx, []string{"12", " 13 ", "783214"})
		}, "/user/users/by/ids", url.Values{"ids": {"12,13,783214"}}},
		{"Users.Tweets by ID", func(c *Client) (json.RawMessage, error) {
			return c.Users.Tweets(ctx, UserTimelineParams{UserID: "12", Count: 50, Cursor: "abc"})
		}, "/user/12/tweets", url.Values{"count": {"50"}, "cursor": {"abc"}}},
		{"Users.Tweets by username", func(c *Client) (json.RawMessage, error) {
			return c.Users.Tweets(ctx, UserTimelineParams{Username: "@jack"})
		}, "/user/-1/tweets", url.Values{"username": {"jack"}}},
		{"Users.Tweets ID wins", func(c *Client) (json.RawMessage, error) {
			return c.Users.Tweets(ctx, UserTimelineParams{UserID: "12", Username: "jack"})
		}, "/user/12/tweets", url.Values{}},
		{"Users.TweetsAndReplies", func(c *Client) (json.RawMessage, error) {
			return c.Users.TweetsAndReplies(ctx, UserTimelineParams{UserID: "12"})
		}, "/user/12/tweetsAndReplies", url.Values{}},
		{"Users.Media", func(c *Client) (json.RawMessage, error) {
			return c.Users.Media(ctx, UserTimelineParams{UserID: "12", Count: 1})
		}, "/user/12/media", url.Values{"count": {"1"}}},
		{"Users.Followers", func(c *Client) (json.RawMessage, error) {
			return c.Users.Followers(ctx, UserTimelineParams{Username: "jack", Count: 100})
		}, "/user/-1/followers", url.Values{"username": {"jack"}, "count": {"100"}}},
		{"Users.Following", func(c *Client) (json.RawMessage, error) {
			return c.Users.Following(ctx, UserTimelineParams{UserID: "12"})
		}, "/user/12/followings", url.Values{}},
		{"Users.VerifiedFollowers", func(c *Client) (json.RawMessage, error) {
			return c.Users.VerifiedFollowers(ctx, UserTimelineParams{UserID: "12", Cursor: "c1"})
		}, "/user/12/verifiedFollowers", url.Values{"cursor": {"c1"}}},
		{"Users.Affiliates by username", func(c *Client) (json.RawMessage, error) {
			return c.Users.Affiliates(ctx, UserTimelineParams{Username: "@XDevelopers", Count: 100, Cursor: "a1"})
		}, "/user/-1/affiliates", url.Values{"username": {"XDevelopers"}, "count": {"100"}, "cursor": {"a1"}}},
		{"Users.Affiliates by ID", func(c *Client) (json.RawMessage, error) {
			return c.Users.Affiliates(ctx, UserTimelineParams{UserID: "783214"})
		}, "/user/783214/affiliates", url.Values{}},
		{"Users.FriendshipsBatch", func(c *Client) (json.RawMessage, error) {
			return c.Users.FriendshipsBatch(ctx, " @jack ", []string{"@elonmusk", " XDevelopers ", "ELONMUSK", "jack"})
		}, "/user/friendships/batch", url.Values{"source_screen_name": {"jack"}, "target_screen_names": {"elonmusk,XDevelopers,jack"}}},
		{"Users.AboutBatch", func(c *Client) (json.RawMessage, error) {
			return c.Users.AboutBatch(ctx, []string{"@jack", "Jack", "x_dev"})
		}, "/user/about/batch", url.Values{"usernames": {"jack,x_dev"}}},

		// Tweets
		{"Tweets.Get", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Get(ctx, TweetParams{TweetID: "20"})
		}, "/tweet/20", url.Values{}},
		{"Tweets.Get with options", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Get(ctx, TweetParams{TweetID: "20", RankingMode: RankingRecency, Cursor: "c"})
		}, "/tweet/20", url.Values{"rankingMode": {"Recency"}, "cursor": {"c"}}},
		{"Tweets.Retweeters", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Retweeters(ctx, TweetPageParams{TweetID: "20", Count: 10})
		}, "/tweet/20/retweets", url.Values{"count": {"10"}}},
		{"Tweets.Likers", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Likers(ctx, LikersParams{TweetID: "20", Cookie: "auth_token=a; ct0=b", Proxy: "http://u:p@h:1"})
		}, "/tweet/20/favorites", url.Values{}},
		{"Tweets.Quotes", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Quotes(ctx, TweetPageParams{TweetID: "20", Cursor: "q"})
		}, "/tweet/20/quotes", url.Values{"cursor": {"q"}}},
		{"Tweets.Replies", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Replies(ctx, RepliesParams{TweetID: " 20 "})
		}, "/tweet/20/replies", url.Values{}},
		{"Tweets.Replies with options", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Replies(ctx, RepliesParams{TweetID: "20", RankingMode: RankingLikes, IncludePromotedContent: true, Cursor: "r"})
		}, "/tweet/20/replies", url.Values{"rankingMode": {"Likes"}, "includePromotedContent": {"true"}, "cursor": {"r"}}},
		{"Tweets.GetMany", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.GetMany(ctx, []string{"20", " 1585841080431321088 "})
		}, "/tweet/tweets/by/ids", url.Values{"ids": {"20,1585841080431321088"}}},
		{"Tweets.Thread", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Thread(ctx, ThreadParams{TweetID: "20"})
		}, "/tweet/20/thread", url.Values{}},
		{"Tweets.Thread with options", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Thread(ctx, ThreadParams{TweetID: "20", RankingMode: RankingRecency, IncludePromotedContent: true})
		}, "/tweet/20/thread", url.Values{"rankingMode": {"Recency"}, "includePromotedContent": {"true"}}},

		// Search
		{"Search.Tweets", func(c *Client) (json.RawMessage, error) {
			return c.Search.Tweets(ctx, SearchParams{Query: " from:jack lang:en ", Category: CategoryLatest, Count: 20, Cursor: "s"})
		}, "/search", url.Values{"searchItem": {"from:jack lang:en"}, "category": {"Latest"}, "count": {"20"}, "cursor": {"s"}}},
		{"Search.Tweets minimal", func(c *Client) (json.RawMessage, error) {
			return c.Search.Tweets(ctx, SearchParams{Query: "golang"})
		}, "/search", url.Values{"searchItem": {"golang"}}},

		// Lists
		{"Lists.Details", func(c *Client) (json.RawMessage, error) { return c.Lists.Details(ctx, "1234567890") },
			"/lists/details", url.Values{"listId": {"1234567890"}}},
		{"Lists.Tweets", func(c *Client) (json.RawMessage, error) {
			return c.Lists.Tweets(ctx, ListPageParams{ListID: "1234567890", Count: 5})
		}, "/lists/tweets", url.Values{"listId": {"1234567890"}, "count": {"5"}}},
		{"Lists.Members", func(c *Client) (json.RawMessage, error) {
			return c.Lists.Members(ctx, ListPageParams{ListID: "1234567890", Cursor: "m"})
		}, "/lists/members", url.Values{"listId": {"1234567890"}, "cursor": {"m"}}},

		// Communities
		{"Communities.Get", func(c *Client) (json.RawMessage, error) { return c.Communities.Get(ctx, "1500") },
			"/communities/1500", url.Values{}},
		{"Communities.About", func(c *Client) (json.RawMessage, error) { return c.Communities.About(ctx, " 1500 ") },
			"/communities/1500/about", url.Values{}},
		{"Communities.Tweets", func(c *Client) (json.RawMessage, error) {
			return c.Communities.Tweets(ctx, CommunityPageParams{CommunityID: "1500", Count: 7})
		}, "/communities/1500/tweets", url.Values{"count": {"7"}}},
		{"Communities.Search", func(c *Client) (json.RawMessage, error) {
			return c.Communities.Search(ctx, CommunitySearchParams{Query: "go lang/dev", Cursor: "x"})
		}, "/communities/search/go%20lang%2Fdev", url.Values{"cursor": {"x"}}},

		// Trends
		{"Trends.Get", func(c *Client) (json.RawMessage, error) { return c.Trends.Get(ctx, WOEIDUnitedStates) },
			"/trends/23424977", url.Values{}},
		{"Trends.Get worldwide", func(c *Client) (json.RawMessage, error) { return c.Trends.Get(ctx, WOEIDWorldwide) },
			"/trends/1", url.Values{}},
		{"Trends.Locations", func(c *Client) (json.RawMessage, error) { return c.Trends.Locations(ctx) },
			"/misc/locations", url.Values{}},

		// Spaces
		{"Spaces.Get", func(c *Client) (json.RawMessage, error) { return c.Spaces.Get(ctx, " 1YqKDqWqdPLsV ") },
			"/spaces/1YqKDqWqdPLsV", url.Values{}},
	}
}

func TestEndpoints(t *testing.T) {
	cases := endpointCases()
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	for _, tc := range cases {
		before := len(rec.all())
		if _, err := tc.call(c); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if len(rec.all()) != before+1 {
			t.Errorf("%s: expected one request", tc.name)
			continue
		}
		r := rec.last(t)
		if got := r.URL.EscapedPath(); got != tc.path {
			t.Errorf("%s: path = %s, want %s", tc.name, got, tc.path)
		}
		if got := r.URL.Query(); fmt.Sprint(got) != fmt.Sprint(tc.query) {
			t.Errorf("%s: query = %v, want %v", tc.name, got, tc.query)
		}
	}
}

// TestEndpointCoverage makes sure there are 31 service methods and that each one has a row
// in endpointCases.
func TestEndpointCoverage(t *testing.T) {
	services := map[string]any{
		"Users": &UsersService{}, "Tweets": &TweetsService{}, "Search": &SearchService{},
		"Lists": &ListsService{}, "Communities": &CommunitiesService{}, "Trends": &TrendsService{},
		"Spaces": &SpacesService{},
	}
	covered := map[string]bool{}
	for _, tc := range endpointCases() {
		covered[strings.Fields(tc.name)[0]] = true
	}
	total := 0
	for name, svc := range services {
		typ := reflect.TypeOf(svc)
		for i := 0; i < typ.NumMethod(); i++ {
			total++
			key := name + "." + typ.Method(i).Name
			if !covered[key] {
				t.Errorf("no endpoint test for %s", key)
			}
		}
	}
	if total != 31 {
		t.Fatalf("expected 31 endpoint methods, found %d", total)
	}
}

func TestValidation(t *testing.T) {
	ctx := context.Background()
	many := make([]string, MaxUsersPerLookup+1)
	for i := range many {
		many[i] = fmt.Sprint(i + 1)
	}
	manyTweets := make([]string, MaxTweetsPerLookup+1)
	for i := range manyTweets {
		manyTweets[i] = fmt.Sprint(i + 1)
	}
	cases := []struct {
		name string
		call func(c *Client) (json.RawMessage, error)
		want string
	}{
		{"empty username", func(c *Client) (json.RawMessage, error) { return c.Users.GetByUsername(ctx, " @ ") }, "username must not be empty"},
		{"non-numeric user ID", func(c *Client) (json.RawMessage, error) { return c.Users.GetByID(ctx, "jack") }, "numeric ID"},
		{"empty username to ID", func(c *Client) (json.RawMessage, error) { return c.Users.UsernameToID(ctx, "") }, "username must not be empty"},
		{"no IDs", func(c *Client) (json.RawMessage, error) { return c.Users.GetMany(ctx, nil) }, "at least one"},
		{"501 IDs", func(c *Client) (json.RawMessage, error) { return c.Users.GetMany(ctx, many) }, "at most 500"},
		{"bad ID in list", func(c *Client) (json.RawMessage, error) { return c.Users.GetMany(ctx, []string{"1", "x"}) }, "numeric ID"},
		{"no tweet IDs", func(c *Client) (json.RawMessage, error) { return c.Tweets.GetMany(ctx, nil) }, "at least one"},
		{"101 tweet IDs", func(c *Client) (json.RawMessage, error) { return c.Tweets.GetMany(ctx, manyTweets) }, "at most 100"},
		{"bad tweet ID in list", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.GetMany(ctx, []string{"20", "12a"})
		}, "numeric ID"},
		{"timeline without user", func(c *Client) (json.RawMessage, error) { return c.Users.Tweets(ctx, UserTimelineParams{}) }, "UserID or Username"},
		{"timeline bad ID", func(c *Client) (json.RawMessage, error) {
			return c.Users.Followers(ctx, UserTimelineParams{UserID: "12a"})
		}, "numeric ID"},
		{"count too high", func(c *Client) (json.RawMessage, error) {
			return c.Users.Tweets(ctx, UserTimelineParams{UserID: "12", Count: 101})
		}, "count must be"},
		{"count negative", func(c *Client) (json.RawMessage, error) {
			return c.Lists.Tweets(ctx, ListPageParams{ListID: "1", Count: -1})
		}, "count must be"},
		{"bad tweet ID", func(c *Client) (json.RawMessage, error) { return c.Tweets.Get(ctx, TweetParams{TweetID: ""}) }, "numeric ID"},
		{"bad ranking mode", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Get(ctx, TweetParams{TweetID: "20", RankingMode: "Newest"})
		}, "RankingMode must be one of"},
		{"bad tweet ID for likers", func(c *Client) (json.RawMessage, error) { return c.Tweets.Likers(ctx, LikersParams{}) }, "numeric ID"},
		{"likers without cookie", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Likers(ctx, LikersParams{TweetID: "20", Proxy: "http://u:p@h:1"})
		}, "Cookie must be"},
		{"likers without proxy", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Likers(ctx, LikersParams{TweetID: "20", Cookie: "auth_token=a; ct0=b", Proxy: " "})
		}, "Proxy must be"},
		{"likers count too high", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Likers(ctx, LikersParams{TweetID: "20", Cookie: "auth_token=a; ct0=b", Proxy: "http://u:p@h:1", Count: 101})
		}, "count must be"},
		{"empty search", func(c *Client) (json.RawMessage, error) { return c.Search.Tweets(ctx, SearchParams{Query: "  "}) }, "Query must be"},
		{"bad category", func(c *Client) (json.RawMessage, error) {
			return c.Search.Tweets(ctx, SearchParams{Query: "x", Category: "latest"})
		}, "Category must be one of"},
		{"bad list ID", func(c *Client) (json.RawMessage, error) { return c.Lists.Details(ctx, "abc") }, "numeric ID"},
		{"bad list ID members", func(c *Client) (json.RawMessage, error) { return c.Lists.Members(ctx, ListPageParams{}) }, "numeric ID"},
		{"bad community ID", func(c *Client) (json.RawMessage, error) { return c.Communities.Get(ctx, "-1") }, "numeric ID"},
		{"bad community ID tweets", func(c *Client) (json.RawMessage, error) {
			return c.Communities.Tweets(ctx, CommunityPageParams{})
		}, "numeric ID"},
		{"empty community search", func(c *Client) (json.RawMessage, error) {
			return c.Communities.Search(ctx, CommunitySearchParams{})
		}, "Query must be"},
		{"bad space ID", func(c *Client) (json.RawMessage, error) { return c.Spaces.Get(ctx, "1Yq-KD") }, "spaceID must be"},
		{"empty space ID", func(c *Client) (json.RawMessage, error) { return c.Spaces.Get(ctx, " ") }, "spaceID must be"},
		{"long space ID", func(c *Client) (json.RawMessage, error) { return c.Spaces.Get(ctx, strings.Repeat("a", 33)) }, "spaceID must be"},
		{"bad community ID about", func(c *Client) (json.RawMessage, error) { return c.Communities.About(ctx, "abc") }, "numeric ID"},
		{"affiliates without user", func(c *Client) (json.RawMessage, error) {
			return c.Users.Affiliates(ctx, UserTimelineParams{})
		}, "UserID or Username"},
		{"affiliates count too high", func(c *Client) (json.RawMessage, error) {
			return c.Users.Affiliates(ctx, UserTimelineParams{UserID: "12", Count: 101})
		}, "count must be"},
		{"bad tweet ID replies", func(c *Client) (json.RawMessage, error) { return c.Tweets.Replies(ctx, RepliesParams{TweetID: "x"}) }, "numeric ID"},
		{"bad ranking mode replies", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Replies(ctx, RepliesParams{TweetID: "20", RankingMode: "recency"})
		}, "RankingMode must be one of"},
		{"bad tweet ID thread", func(c *Client) (json.RawMessage, error) { return c.Tweets.Thread(ctx, ThreadParams{}) }, "numeric ID"},
		{"bad ranking mode thread", func(c *Client) (json.RawMessage, error) {
			return c.Tweets.Thread(ctx, ThreadParams{TweetID: "20", RankingMode: "Top"})
		}, "RankingMode must be one of"},
		{"friendships empty source", func(c *Client) (json.RawMessage, error) {
			return c.Users.FriendshipsBatch(ctx, "@", []string{"jack"})
		}, "source must be a username"},
		{"friendships bad source", func(c *Client) (json.RawMessage, error) {
			return c.Users.FriendshipsBatch(ctx, "jack dorsey", []string{"jack"})
		}, "source must be a username"},
		{"friendships no targets", func(c *Client) (json.RawMessage, error) { return c.Users.FriendshipsBatch(ctx, "jack", nil) }, "targets must contain at least one"},
		{"friendships bad target", func(c *Client) (json.RawMessage, error) {
			return c.Users.FriendshipsBatch(ctx, "jack", []string{"ok", "a,b"})
		}, "each username in targets"},
		{"friendships too many", func(c *Client) (json.RawMessage, error) {
			return c.Users.FriendshipsBatch(ctx, "jack", handles(MaxFriendshipTargets+1))
		}, "at most 50"},
		{"about batch empty", func(c *Client) (json.RawMessage, error) { return c.Users.AboutBatch(ctx, []string{}) }, "usernames must contain at least one"},
		{"about batch long handle", func(c *Client) (json.RawMessage, error) {
			return c.Users.AboutBatch(ctx, []string{strings.Repeat("a", 16)})
		}, "each username in usernames"},
		{"about batch too many", func(c *Client) (json.RawMessage, error) {
			return c.Users.AboutBatch(ctx, handles(MaxAboutUsernames+1))
		}, "at most 20"},
		{"negative WOEID", func(c *Client) (json.RawMessage, error) { return c.Trends.Get(ctx, -1) }, "WOEID"},
		{"huge WOEID", func(c *Client) (json.RawMessage, error) { return c.Trends.Get(ctx, 1_000_000_000_000) }, "WOEID"},
	}

	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	for _, tc := range cases {
		_, err := tc.call(c)
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("%s: want ErrInvalidParams, got %v", tc.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
	if n := len(rec.all()); n != 0 {
		t.Fatalf("invalid params sent %d requests", n)
	}
}

// handles returns n distinct valid handles.
func handles(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("user_%d", i)
	}
	return out
}

func TestBatchLookupsAcceptExactlyMax(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	ctx := context.Background()
	// Duplicates do not count towards the limit.
	targets := append(handles(MaxFriendshipTargets), "USER_0", "@user_1")
	if _, err := c.Users.FriendshipsBatch(ctx, "jack", targets); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(rec.last(t).URL.Query().Get("target_screen_names"), ",") + 1; got != MaxFriendshipTargets {
		t.Fatalf("sent %d targets", got)
	}
	if _, err := c.Users.AboutBatch(ctx, append(handles(MaxAboutUsernames), "User_19")); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(rec.last(t).URL.Query().Get("usernames"), ",") + 1; got != MaxAboutUsernames {
		t.Fatalf("sent %d usernames", got)
	}
}

func TestGetManyAcceptsExactlyMax(t *testing.T) {
	ids := make([]string, MaxUsersPerLookup)
	for i := range ids {
		ids[i] = fmt.Sprint(i + 1)
	}
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	if _, err := c.Users.GetMany(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(rec.last(t).URL.Query().Get("ids"), ",") + 1; got != MaxUsersPerLookup {
		t.Fatalf("sent %d IDs", got)
	}
}

func TestTweetsGetManyAcceptsExactlyMax(t *testing.T) {
	ids := make([]string, MaxTweetsPerLookup)
	for i := range ids {
		ids[i] = fmt.Sprint(i + 1)
	}
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	if _, err := c.Tweets.GetMany(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(rec.last(t).URL.Query().Get("ids"), ",") + 1; got != MaxTweetsPerLookup {
		t.Fatalf("sent %d IDs", got)
	}
}

func TestLikersPost(t *testing.T) {
	var method, contentType, format string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, contentType, format = r.Method, r.Header.Get("Content-Type"), r.URL.Query().Get("format")
		body = nil
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := newTestClient(t, srv)
	ctx := context.Background()

	_, err := c.Tweets.Likers(ctx, LikersParams{TweetID: " 20 ", Cookie: " auth_token=a; ct0=b ", Proxy: "http://u:p@h:1", Count: 50, Cursor: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPost || contentType != "application/json" || format != "" {
		t.Fatalf("method=%s content-type=%q format=%q", method, contentType, format)
	}
	want := map[string]any{"cookie": "auth_token=a; ct0=b", "proxy": "http://u:p@h:1", "count": float64(50), "cursor": "c1"}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}

	_, err = c.Tweets.Likers(ContextWithFormat(ctx, FormatSimple), LikersParams{TweetID: "20", Cookie: "auth_token=a; ct0=b", Proxy: "http://u:p@h:1"})
	if err != nil {
		t.Fatal(err)
	}
	if format != "simple" {
		t.Fatalf("format = %q, want simple", format)
	}
	want = map[string]any{"cookie": "auth_token=a; ct0=b", "proxy": "http://u:p@h:1"}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}
}

func TestPaginateLikersSendsCursorInBody(t *testing.T) {
	pages := [][]byte{simpleFixture(t, "user-userid-followers"), simpleFixture(t, "tweet-tweetid-favorites")}
	var cursors []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		cursors = append(cursors, body["cursor"])
		_, _ = w.Write(pages[min(len(cursors), len(pages))-1])
	}))
	t.Cleanup(srv.Close)
	c, _ := newTestClient(t, srv, WithFormat(FormatSimple))
	it := Paginate(context.Background(), c.Tweets.Likers, LikersParams{TweetID: "20", Cookie: "auth_token=a; ct0=b", Proxy: "http://u:p@h:1"}, nil)
	var nexts []string
	for it.Next() {
		nexts = append(nexts, it.Page().NextCursor)
	}
	if it.Err() != nil || len(cursors) != 2 || cursors[0] != nil || cursors[1] != nexts[0] {
		t.Fatalf("cursors=%v nexts=%v err=%v", cursors, nexts, it.Err())
	}
}
