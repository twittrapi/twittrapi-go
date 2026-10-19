package twittrapi

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
)

// UsersService groups user profile, timeline and follower endpoints. Use it through
// [Client.Users].
type UsersService struct{ client *Client }

// UserTimelineParams selects a user and a page for user timelines and follower lists.
//
// Set either UserID or Username. UserID is slightly faster; with only Username the API
// resolves the handle itself. When both are set, UserID is used.
type UserTimelineParams struct {
	// UserID is the numeric user ID.
	UserID string
	// Username is the handle, with or without a leading "@".
	Username string
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p UserTimelineParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p UserTimelineParams) WithCursor(cursor string) UserTimelineParams {
	p.Cursor = cursor
	return p
}

// GetByUsername returns a profile by handle. A leading "@" is fine. The response includes
// the numeric user ID (rest_id).
func (s *UsersService) GetByUsername(ctx context.Context, username string) (json.RawMessage, error) {
	name, err := cleanUsername(username)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/user/by/username/"+url.PathEscape(name), nil)
}

// GetByID returns a profile by numeric user ID.
func (s *UsersService) GetByID(ctx context.Context, userID string) (json.RawMessage, error) {
	id, err := numericID("userID", userID)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/user/"+id, nil)
}

// UsernameToID resolves a handle to its numeric user ID. It is smaller and cheaper than
// [UsersService.GetByUsername].
func (s *UsersService) UsernameToID(ctx context.Context, username string) (json.RawMessage, error) {
	name, err := cleanUsername(username)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/username/to/id/"+url.PathEscape(name), nil)
}

// GetMany returns up to [MaxUsersPerLookup] (500) profiles in one call by numeric user ID.
func (s *UsersService) GetMany(ctx context.Context, userIDs []string) (json.RawMessage, error) {
	if len(userIDs) == 0 {
		return nil, invalidParams("userIDs must contain at least one ID")
	}
	if len(userIDs) > MaxUsersPerLookup {
		return nil, invalidParams("userIDs accepts at most %d IDs per call, got %d", MaxUsersPerLookup, len(userIDs))
	}
	ids := make([]string, len(userIDs))
	for i, raw := range userIDs {
		id, err := numericID("each user ID", raw)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return s.client.Get(ctx, "/user/users/by/ids", url.Values{"ids": {strings.Join(ids, ",")}})
}

// Tweets returns a user's tweets (profile timeline without replies), newest first.
// Paginated.
func (s *UsersService) Tweets(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "tweets", p)
}

// TweetsAndReplies returns a user's tweets including replies, newest first. Paginated.
func (s *UsersService) TweetsAndReplies(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "tweetsAndReplies", p)
}

// Media returns a user's tweets that contain photos or videos. Paginated.
func (s *UsersService) Media(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "media", p)
}

// Followers returns accounts following a user, newest first. Paginated.
func (s *UsersService) Followers(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "followers", p)
}

// Following returns accounts a user follows. Paginated.
func (s *UsersService) Following(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "followings", p)
}

// VerifiedFollowers returns a user's followers that have a verified badge. Paginated.
func (s *UsersService) VerifiedFollowers(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "verifiedFollowers", p)
}

// Affiliates returns the accounts affiliated with an organization (the Affiliates tab of a
// verified organization's profile). Paginated. In the simple format it is a list of users,
// like [UsersService.Followers].
func (s *UsersService) Affiliates(ctx context.Context, p UserTimelineParams) (json.RawMessage, error) {
	return s.timeline(ctx, "affiliates", p)
}

// FriendshipsBatch returns the relationship between source and up to
// [MaxFriendshipTargets] (50) other accounts in one call. Handles may have a leading "@";
// duplicates are dropped (case-insensitive). Each target is billed as one call.
//
// The answer is built by the API: {"source_screen_name": "...", "relationships":
// [{"target_screen_name": "...", "relationship": {...}}], "errors": [{"username", "error",
// "message"}]}, with relationships in input order and "errors" only when a lookup failed.
// Each relationship is the object GET /user/friendships/show returns. There is no simple
// format.
func (s *UsersService) FriendshipsBatch(ctx context.Context, source string, targets []string) (json.RawMessage, error) {
	src, err := handle("source", source)
	if err != nil {
		return nil, err
	}
	names, err := handleList("targets", targets, MaxFriendshipTargets)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/user/friendships/batch", url.Values{
		"source_screen_name":  {src},
		"target_screen_names": {strings.Join(names, ",")},
	})
}

// AboutBatch returns the "About this account" details of up to [MaxAboutUsernames] (20)
// accounts in one call. Handles may have a leading "@"; duplicates are dropped
// (case-insensitive). Each handle is billed as one call.
//
// The answer is built by the API: {"users": [{"username": "...", "about": {...}}],
// "errors": [{"username", "error": "not_found" | "upstream_error", "message"}]}, with
// "errors" only when a lookup failed. Each about is the object GET /user/about/{username}
// returns. There is no simple format.
func (s *UsersService) AboutBatch(ctx context.Context, usernames []string) (json.RawMessage, error) {
	names, err := handleList("usernames", usernames, MaxAboutUsernames)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/user/about/batch", url.Values{"usernames": {strings.Join(names, ",")}})
}

// timeline builds /user/{id}/{suffix}. With only a username the path uses -1 and the API
// resolves ?username=.
func (s *UsersService) timeline(ctx context.Context, suffix string, p UserTimelineParams) (json.RawMessage, error) {
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	var id string
	switch {
	case strings.TrimSpace(p.UserID) != "":
		if id, err = numericID("UserID", p.UserID); err != nil {
			return nil, err
		}
	case strings.TrimSpace(p.Username) != "":
		name, err := cleanUsername(p.Username)
		if err != nil {
			return nil, err
		}
		id = "-1"
		q.Set("username", name)
	default:
		return nil, invalidParams("provide either UserID or Username")
	}
	return s.client.Get(ctx, "/user/"+id+"/"+suffix, q)
}
