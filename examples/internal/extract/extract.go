// Package extract has small helpers for pulling users and tweets out of X's raw GraphQL
// JSON. They walk the whole response, so they keep working when X moves things around a
// little. They are for the examples only; the SDK itself returns the raw JSON unchanged.
package extract

import (
	"encoding/json"
	"sort"
	"strconv"
)

// User is a flattened user profile.
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Followers int64  `json:"followers"`
	Following int64  `json:"following"`
	Verified  bool   `json:"verified"`
	Bio       string `json:"bio"`
}

// Tweet is a flattened tweet.
type Tweet struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	Author    string `json:"author"`
	Text      string `json:"text"`
	Likes     int64  `json:"likes"`
	Retweets  int64  `json:"retweets"`
	Replies   int64  `json:"replies"`
	Views     int64  `json:"views"`
}

// Walk calls fn for every JSON object in v, depth first.
func Walk(v any, fn func(map[string]any)) {
	switch t := v.(type) {
	case map[string]any:
		fn(t)
		for _, child := range t {
			Walk(child, fn)
		}
	case []any:
		for _, child := range t {
			Walk(child, fn)
		}
	}
}

// Decode parses raw JSON into a generic value. Numbers become float64.
func Decode(raw []byte) (any, error) {
	var v any
	err := json.Unmarshal(raw, &v)
	return v, err
}

func obj(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		next, ok := m[k].(map[string]any)
		if !ok {
			return nil
		}
		m = next
	}
	return m
}

func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func num(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Users returns the users in a response, de-duplicated by ID, in no particular order.
func Users(raw []byte) []User {
	data, err := Decode(raw)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []User
	Walk(data, func(m map[string]any) {
		u := obj(m, "user_results", "result")
		if u == nil || str(u, "__typename") != "User" || seen[str(u, "rest_id")] {
			return
		}
		seen[str(u, "rest_id")] = true
		legacy := obj(u, "legacy")
		if legacy == nil {
			legacy = map[string]any{}
		}
		core := obj(u, "core")
		if core == nil {
			core = map[string]any{}
		}
		verified, _ := u["is_blue_verified"].(bool)
		if lv, ok := legacy["verified"].(bool); ok && lv {
			verified = true
		}
		bio := str(legacy, "description")
		if bio == "" {
			bio = str(obj(u, "profile_bio"), "description")
		}
		out = append(out, User{
			ID:        str(u, "rest_id"),
			Username:  firstNonEmpty(str(core, "screen_name"), str(legacy, "screen_name")),
			Name:      firstNonEmpty(str(core, "name"), str(legacy, "name")),
			Followers: num(legacy, "followers_count"),
			Following: num(legacy, "friends_count"),
			Verified:  verified,
			Bio:       bio,
		})
	})
	return out
}

// Tweets returns the tweets in a response, de-duplicated by ID, newest first (by ID).
func Tweets(raw []byte) []Tweet {
	data, err := Decode(raw)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []Tweet
	Walk(data, func(m map[string]any) {
		t := obj(m, "tweet_results", "result")
		if t != nil && str(t, "__typename") == "TweetWithVisibilityResults" {
			t = obj(t, "tweet")
		}
		legacy := obj(t, "legacy")
		if legacy == nil || seen[str(t, "rest_id")] {
			return
		}
		seen[str(t, "rest_id")] = true
		author := obj(t, "core", "user_results", "result")
		text := str(obj(t, "note_tweet", "note_tweet_results", "result"), "text")
		if text == "" {
			text = str(legacy, "full_text")
		}
		out = append(out, Tweet{
			ID:        str(t, "rest_id"),
			CreatedAt: str(legacy, "created_at"),
			Author:    firstNonEmpty(str(obj(author, "core"), "screen_name"), str(obj(author, "legacy"), "screen_name")),
			Text:      text,
			Likes:     num(legacy, "favorite_count"),
			Retweets:  num(legacy, "retweet_count"),
			Replies:   num(legacy, "reply_count"),
			Views:     num(obj(t, "views"), "count"),
		})
	})
	// Tweet IDs grow over time, so sorting by ID sorts by time.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].ID, out[j].ID
		if len(a) != len(b) {
			return len(a) > len(b)
		}
		return a > b
	})
	return out
}
