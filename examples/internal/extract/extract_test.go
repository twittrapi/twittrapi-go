package extract

// Tests against trimmed real TwittrAPI responses in ../../../testdata/live (the SDK fixtures).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "live", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func usernames(users []User) string {
	var names []string
	for _, u := range users {
		names = append(names, u.Username)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

func TestUsersFromLiveFixtures(t *testing.T) {
	cases := map[string]string{
		"user-userid-followers":           "example_user_1,example_user_2", // legacy.screen_name
		"lists-members":                   "TwitterAble,TwitterThailand",   // core.screen_name
		"tweet-tweetid-retweets":          "example_user_1,example_user_2",
		"communities-communityid-members": "example_user_1,example_user_2", // items_results[].result
		"user-users-by-ids":               "XDevelopers,jack",              // data.users[].result
		"user-userid-likes":               "",
	}
	for name, want := range cases {
		users := Users(fixture(t, name))
		if got := usernames(users); got != want {
			t.Errorf("%s: users = %q, want %q", name, got, want)
		}
		for _, u := range users {
			if u.ID == "" {
				t.Errorf("%s: user without ID: %+v", name, u)
			}
		}
	}
}

func TestTweetsFromLiveFixtures(t *testing.T) {
	cases := map[string][]string{ // id:author, newest first
		"user-userid-tweets":     {"2019881223666233717:XDevelopers"},
		"tweet-tweetid":          {"2019881223666233717:XDevelopers", "2000000000000000001:example_user_1"},
		"search":                 {"2000000000000000001:example_user_1"},
		"user-userid-articles":   {"2041295840325636551:XDevelopers"},
		"tweet-tweetid-retweets": nil, // users only
		"user-userid-likes":      nil,
	}
	for name, want := range cases {
		var got []string
		for _, tw := range Tweets(fixture(t, name)) {
			got = append(got, tw.ID+":"+tw.Author)
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%s: tweets = %q, want %q", name, got, want)
		}
	}
}

func TestNoteTweetLongText(t *testing.T) {
	tweets := Tweets(fixture(t, "user-userid-media"))
	if len(tweets) != 1 {
		t.Fatalf("tweets = %d", len(tweets))
	}
	if text := tweets[0].Text; !strings.HasSuffix(text, "Explore it now: https://t.co/XorO5ldGTN") {
		t.Fatalf("expected the note_tweet text, got %q", text)
	}
}

// realTweet returns the XDevelopers tweet object from the user tweets fixture.
func realTweet(t *testing.T) map[string]any {
	var found map[string]any
	data, _ := Decode(fixture(t, "user-userid-tweets"))
	Walk(data, func(m map[string]any) {
		if r := obj(m, "tweet_results", "result"); str(r, "rest_id") == "2019881223666233717" {
			found = r
		}
	})
	if found == nil {
		t.Fatal("tweet not found in fixture")
	}
	return found
}

func TestVisibilityRetweetAndEmptyResults(t *testing.T) {
	orig := realTweet(t)
	retweet := map[string]any{
		"__typename": "Tweet",
		"rest_id":    "2100000000000000001",
		"core": map[string]any{"user_results": map[string]any{"result": map[string]any{
			"__typename": "User", "rest_id": "1000000000000000001",
			"core": map[string]any{"screen_name": "example_user_1"},
		}}},
		"legacy": map[string]any{
			"full_text":               "RT @XDevelopers: Officially launching X API Pay…",
			"retweeted_status_result": map[string]any{"result": map[string]any{"__typename": "TweetWithVisibilityResults", "tweet": orig}},
		},
	}
	body := map[string]any{"entries": []any{
		map[string]any{"entryId": "tweet-1", "content": map[string]any{"itemContent": map[string]any{"tweet_results": map[string]any{}}}},
		map[string]any{"entryId": "tweet-2", "content": map[string]any{"itemContent": map[string]any{"tweet_results": map[string]any{"result": retweet}}}},
		map[string]any{"entryId": "tweet-3", "content": map[string]any{"itemContent": map[string]any{"tweet_results": map[string]any{
			"result": map[string]any{"__typename": "TweetWithVisibilityResults", "tweet": orig}}}}},
		map[string]any{"entryId": "cursor-bottom-1", "content": map[string]any{"cursorType": "Bottom", "value": "x"}},
	}}
	raw, _ := json.Marshal(body)
	tweets := Tweets(raw)
	if len(tweets) != 2 {
		t.Fatalf("tweets = %+v", tweets)
	}
	rt, plain := tweets[0], tweets[1]
	if rt.Author != "example_user_1" || rt.RetweetOf != "XDevelopers" || !strings.HasPrefix(rt.Text, "RT @XDevelopers: ") || strings.HasSuffix(rt.Text, "…") {
		t.Errorf("retweet = %+v", rt)
	}
	if plain.ID != "2019881223666233717" || plain.Author != "XDevelopers" || plain.RetweetOf != "" {
		t.Errorf("visibility-wrapped tweet = %+v", plain)
	}
	if !strings.HasSuffix(rt.Text, plain.Text) {
		t.Errorf("retweet text %q does not carry the original %q", rt.Text, plain.Text)
	}
}
