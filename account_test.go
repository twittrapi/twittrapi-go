package twittrapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testCookie = "auth_token=secret-token; ct0=secret-csrf; twid=u%3D42"
	testProxy  = "http://puser:ppass@proxy.example:8080"
)

var testSession = XSession{Cookie: testCookie, Proxy: testProxy}

type accountRequest struct {
	method string
	path   string
	query  string
	body   map[string]any
}

// accountServer records POST bodies and answers with the given responses in turn (the last
// one repeats).
func accountServer(t *testing.T, responses ...response) (*httptest.Server, func() []accountRequest) {
	t.Helper()
	if len(responses) == 0 {
		responses = []response{{status: 200, body: `{"status_code":200,"response":"{}"}`}}
	}
	var mu sync.Mutex
	var reqs []accountRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		mu.Lock()
		reqs = append(reqs, accountRequest{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, body})
		resp := responses[min(len(reqs), len(responses))-1]
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []accountRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]accountRequest(nil), reqs...)
	}
}

type accountCase struct {
	name string // "Method ..." so coverage can be checked
	call func(a *AccountService) (json.RawMessage, error)
	path string
	body map[string]any // without cookie and proxy
}

func accountCases() []accountCase {
	ctx := context.Background()
	at := time.Unix(1893456000, 0)
	img := []byte("png-bytes")
	b64 := base64.StdEncoding.EncodeToString(img)
	return []accountCase{
		// Tweets
		{"CreateTweet", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "Hello"})
		}, "/client/create", map[string]any{"text": "Hello"}},
		{"CreateTweet all options", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{
				Text: "Hi", MediaIDs: []string{"1", " 2 "}, PossiblySensitive: true, ReplyControl: ReplyVerified,
				QuoteURL: "https://x.com/jack/status/20", CommunityID: "99", NoteTweet: true,
			})
		}, "/client/create", map[string]any{
			"text": "Hi", "media_ids": []any{"1", "2"}, "possibly_sensitive": true, "conversation_control_mode": "only_verified",
			"quote_url": "https://x.com/jack/status/20", "community_id": "99", "is_note_tweet": true,
		}},
		{"CreateTweet media only", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{MediaIDs: []string{"7"}})
		}, "/client/create", map[string]any{"text": "", "media_ids": []any{"7"}}},
		{"CreateTweet poll", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "Tabs or spaces?", Poll: &Poll{Choices: []string{"Tabs", "Spaces"}, DurationMinutes: 60}})
		}, "/client/create", map[string]any{"text": "Tabs or spaces?", "poll": map[string]any{"choices": []any{"Tabs", "Spaces"}, "duration_minutes": float64(60)}}},
		{"CreateTweet poll default duration", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "?", Poll: &Poll{Choices: []string{"a", "b", "c"}}})
		}, "/client/create", map[string]any{"text": "?", "poll": map[string]any{"choices": []any{"a", "b", "c"}}}},
		{"ScheduleTweet", func(a *AccountService) (json.RawMessage, error) {
			return a.ScheduleTweet(ctx, ScheduleTweetParams{Text: "Later", MediaIDs: []string{"5"}, At: at})
		}, "/client/create", map[string]any{"text": "Later", "media_ids": []any{"5"}, "schedule_time": float64(1893456000)}},
		{"ScheduledTweets", func(a *AccountService) (json.RawMessage, error) {
			return a.ScheduledTweets(ctx, ScheduledTweetsParams{})
		}, "/client/schedule-overview", map[string]any{}},
		{"ScheduledTweets descending", func(a *AccountService) (json.RawMessage, error) {
			return a.ScheduledTweets(ctx, ScheduledTweetsParams{Descending: true})
		}, "/client/schedule-overview", map[string]any{"ascending": false}},
		{"DeleteScheduledTweet", func(a *AccountService) (json.RawMessage, error) {
			return a.DeleteScheduledTweet(ctx, "333")
		}, "/client/schedule-delete", map[string]any{"scheduled_tweet_id": "333"}},
		{"Reply", func(a *AccountService) (json.RawMessage, error) {
			return a.Reply(ctx, ReplyParams{TweetID: " 20 ", Text: "Nice", MediaIDs: []string{"3"}, QuoteURL: "https://x.com/a/status/1", PossiblySensitive: true})
		}, "/client/reply", map[string]any{"id": "20", "text": "Nice", "media_ids": []any{"3"}, "attachment_url": "https://x.com/a/status/1", "possibly_sensitive": true}},
		{"DeleteTweet", func(a *AccountService) (json.RawMessage, error) { return a.DeleteTweet(ctx, "20") },
			"/client/deleteTweet", map[string]any{"tweet_id": "20"}},
		{"Like", func(a *AccountService) (json.RawMessage, error) { return a.Like(ctx, " 20 ") },
			"/client/like", map[string]any{"tweet_id": "20"}},
		{"Unlike", func(a *AccountService) (json.RawMessage, error) { return a.Unlike(ctx, "20") },
			"/client/unlike", map[string]any{"tweet_id": "20"}},
		{"Retweet", func(a *AccountService) (json.RawMessage, error) { return a.Retweet(ctx, "20") },
			"/client/retweet", map[string]any{"tweet_id": "20"}},
		{"Unretweet", func(a *AccountService) (json.RawMessage, error) { return a.Unretweet(ctx, "20") },
			"/client/unretweet", map[string]any{"tweet_id": "20"}},
		{"Pin", func(a *AccountService) (json.RawMessage, error) { return a.Pin(ctx, "20") },
			"/client/pin", map[string]any{"tweet_id": "20"}},
		{"Unpin", func(a *AccountService) (json.RawMessage, error) { return a.Unpin(ctx, "20") },
			"/client/unpin", map[string]any{"tweet_id": "20"}},
		{"Bookmark", func(a *AccountService) (json.RawMessage, error) { return a.Bookmark(ctx, "20") },
			"/client/bookmark", map[string]any{"tweet_id": "20"}},
		{"Unbookmark", func(a *AccountService) (json.RawMessage, error) { return a.Unbookmark(ctx, "20") },
			"/client/unbookmark", map[string]any{"tweet_id": "20"}},
		{"Report", func(a *AccountService) (json.RawMessage, error) {
			return a.Report(ctx, ReportParams{TweetID: "20", UserID: "12", Reason: ReportSpam})
		}, "/client/report", map[string]any{"tweet_id": "20", "user_id": "12", "reason": "SpamSimpleOption"}},

		// Users
		{"Follow by ID", func(a *AccountService) (json.RawMessage, error) { return a.Follow(ctx, UserByID("12")) },
			"/client/follow", map[string]any{"user_id": "12"}},
		{"Follow by username", func(a *AccountService) (json.RawMessage, error) { return a.Follow(ctx, UserByUsername(" @jack ")) },
			"/client/follow", map[string]any{"username": "jack"}},
		{"Follow ID wins", func(a *AccountService) (json.RawMessage, error) {
			return a.Follow(ctx, UserRef{ID: "12", Username: "jack"})
		}, "/client/follow", map[string]any{"user_id": "12"}},
		{"Unfollow", func(a *AccountService) (json.RawMessage, error) { return a.Unfollow(ctx, UserByUsername("jack")) },
			"/client/unfollow", map[string]any{"username": "jack"}},
		{"Mute", func(a *AccountService) (json.RawMessage, error) { return a.Mute(ctx, UserByID("12")) },
			"/client/mute", map[string]any{"user_id": "12"}},
		{"Unmute", func(a *AccountService) (json.RawMessage, error) { return a.Unmute(ctx, UserByUsername("jack")) },
			"/client/unmute", map[string]any{"username": "jack"}},
		{"Block", func(a *AccountService) (json.RawMessage, error) { return a.Block(ctx, UserByID("12")) },
			"/client/block", map[string]any{"user_id": "12"}},
		{"Unblock", func(a *AccountService) (json.RawMessage, error) { return a.Unblock(ctx, UserByUsername("jack")) },
			"/client/unblock", map[string]any{"username": "jack"}},

		// Reads
		{"HomeTimeline", func(a *AccountService) (json.RawMessage, error) { return a.HomeTimeline(ctx, HomeTimelineParams{}) },
			"/client/home-timeline", map[string]any{}},
		{"HomeTimeline options", func(a *AccountService) (json.RawMessage, error) {
			return a.HomeTimeline(ctx, HomeTimelineParams{Count: 40, ExcludePromoted: true})
		}, "/client/home-timeline", map[string]any{"count": float64(40), "include_promoted_content": false}},
		{"Bookmarks", func(a *AccountService) (json.RawMessage, error) {
			return a.Bookmarks(ctx, AccountPageParams{Count: 50, Cursor: "c1"})
		}, "/client/bookmarks", map[string]any{"count": float64(50), "cursor": "c1"}},
		{"BookmarkFolders", func(a *AccountService) (json.RawMessage, error) { return a.BookmarkFolders(ctx, AccountCursorParams{}) },
			"/client/bookmarks/folders", map[string]any{}},
		{"BookmarkFolders cursor", func(a *AccountService) (json.RawMessage, error) {
			return a.BookmarkFolders(ctx, AccountCursorParams{Cursor: "f1"})
		}, "/client/bookmarks/folders", map[string]any{"cursor": "f1"}},
		{"BookmarkFolder", func(a *AccountService) (json.RawMessage, error) {
			return a.BookmarkFolder(ctx, BookmarkFolderParams{FolderID: " 1700 ", Cursor: "b"})
		}, "/client/bookmarks/folder", map[string]any{"folder_id": "1700", "cursor": "b"}},
		{"SearchBookmarks", func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: " golang ", Count: 100, Cursor: "s"})
		}, "/client/bookmarks/search", map[string]any{"query": "golang", "count": float64(100), "cursor": "s"}},
		{"SearchBookmarks minimal", func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: "go"})
		}, "/client/bookmarks/search", map[string]any{"query": "go"}},
		{"Blocked", func(a *AccountService) (json.RawMessage, error) { return a.Blocked(ctx, AccountPageParams{}) },
			"/client/blocked", map[string]any{}},
		{"Muted", func(a *AccountService) (json.RawMessage, error) { return a.Muted(ctx, AccountPageParams{Cursor: "m"}) },
			"/client/muted", map[string]any{"cursor": "m"}},
		{"Likes", func(a *AccountService) (json.RawMessage, error) { return a.Likes(ctx, LikesParams{Count: 10}) },
			"/client/likes", map[string]any{"count": float64(10)}},
		{"Likes with user ID", func(a *AccountService) (json.RawMessage, error) {
			return a.Likes(ctx, LikesParams{UserID: "42", Cursor: "l"})
		}, "/client/likes", map[string]any{"user_id": "42", "cursor": "l"}},
		{"FollowersYouKnow", func(a *AccountService) (json.RawMessage, error) {
			return a.FollowersYouKnow(ctx, FollowersYouKnowParams{User: UserByUsername("jack"), Count: 5})
		}, "/client/followers-you-know", map[string]any{"username": "jack", "count": float64(5)}},

		// Lists
		{"CreateList", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateList(ctx, CreateListParams{Name: "Go", Description: "Gophers", Private: true})
		}, "/client/lists/create", map[string]any{"name": "Go", "description": "Gophers", "private": true}},
		{"UpdateList", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateList(ctx, UpdateListParams{ListID: "8", Description: Ptr(""), Private: Ptr(false)})
		}, "/client/lists/update", map[string]any{"list_id": "8", "description": "", "private": false}},
		{"DeleteList", func(a *AccountService) (json.RawMessage, error) { return a.DeleteList(ctx, "8") },
			"/client/lists/delete", map[string]any{"list_id": "8"}},
		{"AddListMember", func(a *AccountService) (json.RawMessage, error) {
			return a.AddListMember(ctx, "8", UserByUsername("jack"))
		},
			"/client/lists/members/add", map[string]any{"list_id": "8", "username": "jack"}},
		{"RemoveListMember", func(a *AccountService) (json.RawMessage, error) { return a.RemoveListMember(ctx, "8", UserByID("12")) },
			"/client/lists/members/remove", map[string]any{"list_id": "8", "user_id": "12"}},

		// Communities
		{"CreateCommunity", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateCommunity(ctx, CreateCommunityParams{Name: "API testers", Description: "d", JoinPolicy: "Closed", InvitesPolicy: "AdminInvitesOnly", Question: "Why?"})
		}, "/client/communities/create", map[string]any{"name": "API testers", "description": "d", "joinPolicy": "Closed", "invitesPolicy": "AdminInvitesOnly", "question": "Why?"}},
		{"DeleteCommunity", func(a *AccountService) (json.RawMessage, error) { return a.DeleteCommunity(ctx, "77", "API testers") },
			"/client/communities/remove", map[string]any{"community_id": "77", "name": "API testers"}},
		{"JoinCommunity", func(a *AccountService) (json.RawMessage, error) { return a.JoinCommunity(ctx, "77") },
			"/client/communities/join", map[string]any{"community_id": "77"}},
		{"LeaveCommunity", func(a *AccountService) (json.RawMessage, error) { return a.LeaveCommunity(ctx, "77") },
			"/client/communities/leave", map[string]any{"community_id": "77"}},
		{"CommunityMemberships", func(a *AccountService) (json.RawMessage, error) {
			return a.CommunityMemberships(ctx, AccountCursorParams{Cursor: "cm"})
		}, "/client/communities/memberships", map[string]any{"cursor": "cm"}},

		// Drafts
		{"Drafts", func(a *AccountService) (json.RawMessage, error) { return a.Drafts(ctx, DraftsParams{Ascending: true}) },
			"/client/drafts", map[string]any{"ascending": true}},
		{"CreateDraft", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateDraft(ctx, DraftParams{Text: "Draft", MediaIDs: []string{"4"}})
		}, "/client/drafts/create", map[string]any{"text": "Draft", "media_ids": []any{"4"}}},
		{"EditDraft", func(a *AccountService) (json.RawMessage, error) {
			return a.EditDraft(ctx, "55", DraftParams{Text: "New"})
		},
			"/client/drafts/edit", map[string]any{"draft_tweet_id": "55", "text": "New"}},
		{"DeleteDraft", func(a *AccountService) (json.RawMessage, error) { return a.DeleteDraft(ctx, "55") },
			"/client/drafts/delete", map[string]any{"draft_tweet_id": "55"}},

		// Articles
		{"Articles", func(a *AccountService) (json.RawMessage, error) { return a.Articles(ctx, ArticlesParams{}) },
			"/client/articles", map[string]any{}},
		{"Articles all options", func(a *AccountService) (json.RawMessage, error) {
			return a.Articles(ctx, ArticlesParams{Lifecycle: "Published", UserID: "42", Count: 5, Cursor: "a"})
		}, "/client/articles", map[string]any{"lifecycle": "published", "user_id": "42", "count": float64(5), "cursor": "a"}},
		{"Article", func(a *AccountService) (json.RawMessage, error) { return a.Article(ctx, " 1800 ") },
			"/client/articles/get", map[string]any{"article_id": "1800"}},
		{"CreateArticle empty", func(a *AccountService) (json.RawMessage, error) { return a.CreateArticle(ctx, CreateArticleParams{}) },
			"/client/articles/create", map[string]any{}},
		{"CreateArticle text", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{Title: "Hello", Text: "One\nTwo"})
		}, "/client/articles/create", map[string]any{"title": "Hello", "text": "One\nTwo"}},
		{"CreateArticle content state", func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{ContentState: json.RawMessage(` {"blocks":[{"text":"Hi"}],"entity_map":[]} `)})
		}, "/client/articles/create", map[string]any{"content_state": map[string]any{"blocks": []any{map[string]any{"text": "Hi"}}, "entity_map": []any{}}}},
		{"UpdateArticle title and text", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1800", Title: Ptr(""), Text: Ptr("Body")})
		}, "/client/articles/update", map[string]any{"article_id": "1800", "title": "", "text": "Body"}},
		{"UpdateArticle cover", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1800", CoverMediaID: " 66 ", CoverMediaCategory: "amplifyvideo"})
		}, "/client/articles/update", map[string]any{"article_id": "1800", "cover_media_id": "66", "cover_media_category": "AmplifyVideo"}},
		{"UpdateArticle remove cover", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1800", RemoveCover: true})
		}, "/client/articles/update", map[string]any{"article_id": "1800", "remove_cover": true}},
		{"PublishArticle", func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1800"})
		},
			"/client/articles/publish", map[string]any{"article_id": "1800"}},
		{"PublishArticle all options", func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1800", Visibility: ArticlePublic, ReplyControl: ArticleRepliesByInvitation, TweetText: " Read this "})
		}, "/client/articles/publish", map[string]any{"article_id": "1800", "visibility": "public", "reply_control": "byinvitation", "tweet_text": "Read this"}},
		{"UnpublishArticle", func(a *AccountService) (json.RawMessage, error) { return a.UnpublishArticle(ctx, "1800") },
			"/client/articles/unpublish", map[string]any{"article_id": "1800"}},
		{"DeleteArticle", func(a *AccountService) (json.RawMessage, error) { return a.DeleteArticle(ctx, "1800") },
			"/client/articles/delete", map[string]any{"article_id": "1800"}},

		// Media
		{"UploadImage", func(a *AccountService) (json.RawMessage, error) {
			return a.UploadImage(ctx, ImageUpload{Data: img, MediaType: "image/png"})
		}, "/client/media", map[string]any{"media": b64, "media_type": "image/png"}},
		{"UploadVideo", func(a *AccountService) (json.RawMessage, error) {
			return a.UploadVideo(ctx, VideoUpload{Data: img, Duration: 12 * time.Second})
		}, "/client/media/video", map[string]any{"media": b64, "video_duration_ms": float64(12000)}},
		{"MediaStatus", func(a *AccountService) (json.RawMessage, error) { return a.MediaStatus(ctx, "66") },
			"/client/media/status", map[string]any{"media_id": "66"}},

		// Profile
		{"UpdateProfile", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateProfile(ctx, ProfileUpdate{Name: Ptr("Jack"), Description: Ptr(""), Location: Ptr("Berlin"), URL: Ptr("https://example.com"),
				Birthdate: &Birthdate{Day: 2, Month: 3, Year: 1990, Visibility: VisibilitySelf}})
		}, "/client/profile", map[string]any{"name": "Jack", "description": "", "location": "Berlin", "url": "https://example.com",
			"birthdate": map[string]any{"day": float64(2), "month": float64(3), "year": float64(1990), "visibility": "self"}}},
		{"UpdateProfile remove birthdate", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateProfile(ctx, ProfileUpdate{RemoveBirthdate: true})
		}, "/client/profile", map[string]any{"birthdate": nil}},
		{"UpdateProfileImage", func(a *AccountService) (json.RawMessage, error) { return a.UpdateProfileImage(ctx, img) },
			"/client/profile/image", map[string]any{"media": b64}},
		{"UpdateProfileBanner", func(a *AccountService) (json.RawMessage, error) { return a.UpdateProfileBanner(ctx, img) },
			"/client/profile/banner", map[string]any{"media": b64}},

		// Direct messages
		{"DMInbox", func(a *AccountService) (json.RawMessage, error) { return a.DMInbox(ctx, DMInboxParams{Count: 50}) },
			"/client/dm/inbox", map[string]any{"count": float64(50), "passcode": "0420"}},
		{"DMInbox without passcode", func(a *AccountService) (json.RawMessage, error) {
			return a.WithDMPasscode("").DMInbox(ctx, DMInboxParams{})
		}, "/client/dm/inbox", map[string]any{}},
		{"DMConversation", func(a *AccountService) (json.RawMessage, error) {
			return a.DMConversation(ctx, DMConversationParams{UserID: "12", Cursor: "123"})
		}, "/client/dm/conversation", map[string]any{"user_id": "12", "cursor": "123", "passcode": "0420"}},
		{"DMConversationInfo", func(a *AccountService) (json.RawMessage, error) {
			return a.DMConversationInfo(ctx, DMChat{ConversationID: "12:42", UserID: "99"})
		}, "/client/dm/conversation/info", map[string]any{"conversation_id": "12:42"}},
		{"SetDMDisappearing", func(a *AccountService) (json.RawMessage, error) {
			return a.SetDMDisappearing(ctx, DMDisappearingParams{ConversationID: "12:42", TTL: 24 * time.Hour})
		}, "/client/dm/conversation/disappearing", map[string]any{"conversation_id": "12:42", "ttl_seconds": float64(86400), "passcode": "0420"}},
		{"SetDMDisappearing off", func(a *AccountService) (json.RawMessage, error) {
			return a.SetDMDisappearing(ctx, DMDisappearingParams{UserID: "12"})
		}, "/client/dm/conversation/disappearing", map[string]any{"user_id": "12", "ttl_seconds": float64(0), "passcode": "0420"}},
		{"SendDM", func(a *AccountService) (json.RawMessage, error) {
			return a.SendDM(ctx, SendDMParams{UserID: "12", Text: "hi"})
		}, "/client/dm/send", map[string]any{"user_id": "12", "text": "hi", "passcode": "0420"}},
		{"DMSettings", func(a *AccountService) (json.RawMessage, error) { return a.DMSettings(ctx) },
			"/client/dm/settings", map[string]any{}},
		{"UpdateDMSettings", func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateDMSettings(ctx, DMSettingsUpdate{AllowDMsFrom: Ptr("verified"), AVCallsEnabled: Ptr(false), AcceptCallsFromFollowing: Ptr(true)})
		}, "/client/dm/settings/update", map[string]any{"allow_dms_from": "verified", "av_calls_enabled": false, "accept_calls_from_following": true}},
	}
}

func TestAccountEndpoints(t *testing.T) {
	srv, reqs := accountServer(t)
	c, _ := newTestClient(t, srv)
	acct := c.Account(XSession{Cookie: " " + testCookie + " ", Proxy: testProxy}).WithDMPasscode("0420")
	for _, tc := range accountCases() {
		before := len(reqs())
		if _, err := tc.call(acct); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		all := reqs()
		if len(all) != before+1 {
			t.Errorf("%s: expected one request, got %d", tc.name, len(all)-before)
			continue
		}
		r := all[len(all)-1]
		if r.method != http.MethodPost || r.path != tc.path || r.query != "" {
			t.Errorf("%s: %s %s?%s, want POST %s", tc.name, r.method, r.path, r.query, tc.path)
		}
		want := map[string]any{"cookie": testCookie, "proxy": testProxy}
		for k, v := range tc.body {
			want[k] = v
		}
		if !reflect.DeepEqual(r.body, want) {
			t.Errorf("%s: body = %v\nwant %v", tc.name, r.body, want)
		}
	}
}

// TestAccountCoverage makes sure every exported AccountService method has a row in
// accountCases.
func TestAccountCoverage(t *testing.T) {
	skip := map[string]bool{"String": true, "GoString": true, "WithDMPasscode": true}
	covered := map[string]bool{}
	for _, tc := range accountCases() {
		covered[strings.Fields(tc.name)[0]] = true
	}
	typ := reflect.TypeOf(&AccountService{})
	total := 0
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if skip[name] {
			continue
		}
		total++
		if !covered[name] {
			t.Errorf("no endpoint test for AccountService.%s", name)
		}
	}
	if total != 64 {
		t.Fatalf("expected 64 account methods, found %d", total)
	}
}

func TestAccountValidation(t *testing.T) {
	ctx := context.Background()
	long := strings.Repeat("a", 26)
	cases := []struct {
		name string
		sess XSession
		call func(a *AccountService) (json.RawMessage, error)
		want string
	}{
		{"bad folder ID", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.BookmarkFolder(ctx, BookmarkFolderParams{FolderID: "abc"})
		}, "numeric ID"},
		{"empty bookmark search", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: " "})
		}, "Query must be"},
		{"long bookmark search", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: strings.Repeat("é", 501)})
		}, "longer than 500"},
		{"bookmark search count", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: "go", Count: 101})
		}, "count must be"},
		{"bad lifecycle", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Articles(ctx, ArticlesParams{Lifecycle: "archived"})
		}, "Lifecycle must be one of"},
		{"articles bad user ID", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Articles(ctx, ArticlesParams{UserID: "me"})
		}, "numeric ID"},
		{"articles count", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Articles(ctx, ArticlesParams{Count: 101})
		}, "count must be"},
		{"bad article ID", testSession, func(a *AccountService) (json.RawMessage, error) { return a.Article(ctx, "") }, "numeric ID"},
		{"create article text and state", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{Text: "x", ContentState: json.RawMessage(`{"blocks":[],"entity_map":[]}`)})
		}, "not both"},
		{"create article bad state", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{ContentState: json.RawMessage(`{"blocks":{}}`)})
		}, "ContentState must be"},
		{"create article state without entity map", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{ContentState: json.RawMessage(`{"blocks":[]}`)})
		}, "ContentState must be"},
		{"update article no ID", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{Title: Ptr("x")})
		}, "numeric ID"},
		{"update article nothing", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1"})
		}, "at least one"},
		{"update article text and state", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", Text: Ptr(""), ContentState: json.RawMessage(`{"blocks":[],"entity_map":[]}`)})
		}, "not both"},
		{"update article bad cover", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", CoverMediaID: "abc"})
		}, "numeric ID"},
		{"update article bad cover category", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", CoverMediaID: "6", CoverMediaCategory: "Photo"})
		}, "CoverMediaCategory must be one of"},
		{"update article category without cover", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", CoverMediaCategory: ArticleCoverGIF})
		}, "needs CoverMediaID"},
		{"update article cover and remove", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", CoverMediaID: "6", RemoveCover: true})
		}, "not both"},
		{"publish article bad visibility", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1", Visibility: "private"})
		}, "Visibility must be one of"},
		{"publish article bad reply control", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1", ReplyControl: "only_follows"})
		}, "ReplyControl must be one of"},
		{"publish article long text", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1", TweetText: strings.Repeat("ü", 257)})
		}, "longer than 256"},
		{"unpublish article bad ID", testSession, func(a *AccountService) (json.RawMessage, error) { return a.UnpublishArticle(ctx, "x") }, "numeric ID"},
		{"delete article bad ID", testSession, func(a *AccountService) (json.RawMessage, error) { return a.DeleteArticle(ctx, "1 2") }, "numeric ID"},
		{"no cookie", XSession{Proxy: testProxy}, func(a *AccountService) (json.RawMessage, error) { return a.Like(ctx, "20") }, "Cookie must be"},
		{"no proxy", XSession{Cookie: testCookie, Proxy: " "}, func(a *AccountService) (json.RawMessage, error) { return a.DMSettings(ctx) }, "Proxy must be"},
		{"bad tweet ID", testSession, func(a *AccountService) (json.RawMessage, error) { return a.Like(ctx, "abc") }, "numeric ID"},
		{"no user", testSession, func(a *AccountService) (json.RawMessage, error) { return a.Follow(ctx, UserRef{}) }, "user ID or username"},
		{"bad user ID", testSession, func(a *AccountService) (json.RawMessage, error) { return a.Block(ctx, UserByID("jack")) }, "numeric ID"},
		{"empty tweet", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: " "})
		}, "Text must be"},
		{"five media", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x", MediaIDs: []string{"1", "2", "3", "4", "5"}})
		}, "at most 4"},
		{"bad media ID", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Reply(ctx, ReplyParams{TweetID: "1", Text: "x", MediaIDs: []string{"m"}})
		}, "numeric ID"},
		{"bad reply control", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x", ReplyControl: "friends"})
		}, "ReplyControl must be one of"},
		{"poll one choice", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x", Poll: &Poll{Choices: []string{"a"}}})
		}, "2 to 4 choices"},
		{"poll with media", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x", MediaIDs: []string{"1"}, Poll: &Poll{Choices: []string{"a", "b"}}})
		}, "cannot be combined with MediaIDs"},
		{"poll duration", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x", Poll: &Poll{Choices: []string{"a", "b"}, DurationMinutes: 4}})
		}, "DurationMinutes"},
		{"schedule without time", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.ScheduleTweet(ctx, ScheduleTweetParams{Text: "x"})
		}, "At must be set"},
		{"report reason", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Report(ctx, ReportParams{TweetID: "1", UserID: "2", Reason: "Rude"})
		}, "Reason must be one of"},
		{"report username", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Report(ctx, ReportParams{TweetID: "1", UserID: "jack", Reason: ReportSpam})
		}, "numeric ID"},
		{"page count", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.Bookmarks(ctx, AccountPageParams{Count: 101})
		}, "count must be"},
		{"list name too long", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateList(ctx, CreateListParams{Name: long})
		}, "longer than 25"},
		{"list update empty", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateList(ctx, UpdateListParams{ListID: "1"})
		}, "at least one"},
		{"community without name", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.DeleteCommunity(ctx, "1", "")
		}, "name must be"},
		{"empty draft", testSession, func(a *AccountService) (json.RawMessage, error) { return a.CreateDraft(ctx, DraftParams{}) }, "Text must be"},
		{"empty image", testSession, func(a *AccountService) (json.RawMessage, error) { return a.UploadImage(ctx, ImageUpload{}) }, "Data must not be empty"},
		{"video duration", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UploadVideo(ctx, VideoUpload{Data: []byte("v")})
		}, "Duration must be"},
		{"empty profile", testSession, func(a *AccountService) (json.RawMessage, error) { return a.UpdateProfile(ctx, ProfileUpdate{}) }, "at least one"},
		{"profile empty name", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateProfile(ctx, ProfileUpdate{Name: Ptr(" ")})
		}, "Name cannot be empty"},
		{"profile bad visibility", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateProfile(ctx, ProfileUpdate{Birthdate: &Birthdate{Day: 1, Month: 1, Year: 2000, YearVisibility: "everyone"}})
		}, "year_visibility must be one of"},
		{"send DM without passcode", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.SendDM(ctx, SendDMParams{UserID: "1", Text: "hi"})
		}, "WithDMPasscode"},
		{"DM without chat", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.DMConversationInfo(ctx, DMChat{})
		}, "ConversationID or UserID"},
		{"DM bad conversation ID", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.DMConversationInfo(ctx, DMChat{ConversationID: "a b"})
		}, "ConversationID may hold"},
		{"DM TTL too short", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.WithDMPasscode("1").SetDMDisappearing(ctx, DMDisappearingParams{UserID: "1", TTL: time.Second})
		}, "TTL must be"},
		{"DM inbox count", testSession, func(a *AccountService) (json.RawMessage, error) { return a.DMInbox(ctx, DMInboxParams{Count: 51}) }, "Count must be"},
		{"DM settings empty", testSession, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateDMSettings(ctx, DMSettingsUpdate{})
		}, "at least one"},
	}
	srv, reqs := accountServer(t)
	c, _ := newTestClient(t, srv)
	for _, tc := range cases {
		_, err := tc.call(c.Account(tc.sess))
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("%s: want ErrInvalidParams, got %v", tc.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Errorf("%s: error leaks the cookie: %q", tc.name, err)
		}
	}
	if n := len(reqs()); n != 0 {
		t.Fatalf("invalid params sent %d requests", n)
	}
}

func TestXSessionMasking(t *testing.T) {
	secrets := []string{"secret-token", "secret-csrf", "u%3D42", "puser", "ppass"}
	acct := (&Client{}).Account(testSession).WithDMPasscode("0420")
	outputs := map[string]string{
		"String":            testSession.String(),
		"%v":                fmt.Sprintf("%v", testSession),
		"%+v":               fmt.Sprintf("%+v", testSession),
		"%#v":               fmt.Sprintf("%#v", testSession),
		"%s":                fmt.Sprintf("%s", testSession),
		"pointer %v":        fmt.Sprintf("%v", &testSession),
		"nested %+v":        fmt.Sprintf("%+v", struct{ S XSession }{testSession}),
		"service %v":        fmt.Sprintf("%v", acct),
		"service %+v":       fmt.Sprintf("%+v", acct),
		"service %#v":       fmt.Sprintf("%#v", acct),
		"service String":    acct.String(),
		"service in struct": fmt.Sprintf("%+v", struct{ A *AccountService }{acct}),
	}
	for name, out := range outputs {
		for _, s := range secrets {
			if strings.Contains(out, s) {
				t.Errorf("%s leaks %q: %s", name, s, out)
			}
		}
		if strings.Contains(out, "0420") {
			t.Errorf("%s leaks the passcode: %s", name, out)
		}
	}
	want := `XSession{Cookie: "auth_token=***; ct0=***; twid=***", Proxy: "http://***@proxy.example:8080"}`
	if got := testSession.String(); got != want {
		t.Errorf("String() = %s\nwant %s", got, want)
	}
	if got := acct.String(); !strings.Contains(got, `DMPasscode: "***"`) {
		t.Errorf("service String() = %s", got)
	}

	masks := []struct{ in, cookie, proxy string }{
		{"", "", ""},
		{"lonevalue", "***", "***"},
		{"a=1;;b=", "a=***; b=***", "***"},
		{"socks5://host:1080", "***", "socks5://host:1080"},
		{"http://user@host:1", "***", "http://***@host:1"},
	}
	for _, m := range masks {
		if got := maskCookie(m.in); got != m.cookie {
			t.Errorf("maskCookie(%q) = %q, want %q", m.in, got, m.cookie)
		}
		if got := maskProxy(m.in); got != m.proxy {
			t.Errorf("maskProxy(%q) = %q, want %q", m.in, got, m.proxy)
		}
	}
}

func TestAccountRetries(t *testing.T) {
	ctx := context.Background()
	fail := response{status: 500, body: `{"error":"boom"}`}
	limited := response{status: 429, body: `{"error":"slow down"}`}
	ok := response{status: 200, body: `{"status_code":200,"response":"{}"}`}
	cases := []struct {
		name  string
		resps []response
		call  func(a *AccountService) (json.RawMessage, error)
		want  int // requests
		isErr bool
	}{
		{"create not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x"})
		}, 1, true},
		{"send DM not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.WithDMPasscode("1").SendDM(ctx, SendDMParams{UserID: "1", Text: "x"})
		}, 1, true},
		{"create retried on 429", []response{limited, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateTweet(ctx, CreateTweetParams{Text: "x"})
		}, 2, false},
		{"like retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.Like(ctx, "1")
		}, 2, false},
		{"create article not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.CreateArticle(ctx, CreateArticleParams{Title: "x"})
		}, 1, true},
		{"update article not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.UpdateArticle(ctx, UpdateArticleParams{ArticleID: "1", Title: Ptr("x")})
		}, 1, true},
		{"publish article not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1"})
		}, 1, true},
		{"unpublish article not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.UnpublishArticle(ctx, "1")
		}, 1, true},
		{"delete article not retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.DeleteArticle(ctx, "1")
		}, 1, true},
		{"publish article retried on 429", []response{limited, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.PublishArticle(ctx, PublishArticleParams{ArticleID: "1"})
		}, 2, false},
		{"article read retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.Article(ctx, "1")
		}, 2, false},
		{"articles list retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.Articles(ctx, ArticlesParams{})
		}, 2, false},
		{"bookmark folders retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.BookmarkFolders(ctx, AccountCursorParams{})
		}, 2, false},
		{"bookmark search retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.SearchBookmarks(ctx, SearchBookmarksParams{Query: "go"})
		}, 2, false},
		{"community memberships retried on 5xx", []response{fail, ok}, func(a *AccountService) (json.RawMessage, error) {
			return a.CommunityMemberships(ctx, AccountCursorParams{})
		}, 2, false},
	}
	for _, tc := range cases {
		srv, reqs := accountServer(t, tc.resps...)
		c, _ := newTestClient(t, srv)
		_, err := tc.call(c.Account(testSession))
		if (err != nil) != tc.isErr {
			t.Errorf("%s: err = %v", tc.name, err)
		}
		if n := len(reqs()); n != tc.want {
			t.Errorf("%s: %d requests, want %d", tc.name, n, tc.want)
		}
	}
}

func TestAccountErrorPassThrough(t *testing.T) {
	wrapper := `{"status_code":403,"response":"{\"errors\":[{\"code\":139}]}"}`
	srv, _ := accountServer(t, response{status: 403, body: wrapper})
	c, _ := newTestClient(t, srv)
	_, err := c.Account(testSession).Like(context.Background(), "20")
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) || string(authErr.Body) != wrapper {
		t.Fatalf("err = %v", err)
	}
	status, body, ok := UnwrapAccountResponse(authErr.Body)
	if !ok || status != 403 || string(body) != `{"errors":[{"code":139}]}` {
		t.Fatalf("unwrap = %d %s %v", status, body, ok)
	}
}

func TestAccountSimpleFormat(t *testing.T) {
	srv, reqs := accountServer(t)
	c, _ := newTestClient(t, srv)
	ctx := ContextWithFormat(context.Background(), FormatSimple)
	if _, err := c.Account(testSession).Like(ctx, "20"); err != nil {
		t.Fatal(err)
	}
	if q := reqs()[0].query; q != "format=simple" {
		t.Fatalf("query = %q", q)
	}
}

func TestUnwrapAccountResponse(t *testing.T) {
	cases := []struct {
		in     string
		status int
		body   string
		ok     bool
	}{
		{`{"status_code":200,"response":"{\"data\":{}}"}`, 200, `{"data":{}}`, true},
		{`{"status_code":200,"response":"not json"}`, 200, "not json", true},
		{`{"response":"{}"}`, 0, "", false},
		{`{"status_code":200,"response":{}}`, 0, "", false},
		{`{"data":{"success":true}}`, 0, "", false},
		{`[]`, 0, "", false},
	}
	for _, tc := range cases {
		status, body, ok := UnwrapAccountResponse([]byte(tc.in))
		if status != tc.status || string(body) != tc.body || ok != tc.ok {
			t.Errorf("%s: got %d %q %v", tc.in, status, body, ok)
		}
	}
}

func TestPaginateAccountBookmarks(t *testing.T) {
	wrap := func(inner string) string {
		s, _ := json.Marshal(inner)
		return fmt.Sprintf(`{"status_code":200,"response":%s}`, s)
	}
	srv, reqs := accountServer(t,
		response{status: 200, body: wrap(timelinePage(2, "B1"))},
		response{status: 200, body: wrap(timelinePage(1, ""))},
	)
	c, _ := newTestClient(t, srv)
	it := Paginate(context.Background(), c.Account(testSession).Bookmarks, AccountPageParams{Count: 20}, nil)
	var nexts []string
	for it.Next() {
		nexts = append(nexts, it.Page().NextCursor)
	}
	if it.Err() != nil {
		t.Fatal(it.Err())
	}
	all := reqs()
	if len(all) != 2 || all[0].body["cursor"] != nil || all[1].body["cursor"] != "B1" || nexts[0] != "B1" {
		t.Fatalf("requests=%v nexts=%v", all, nexts)
	}
}
