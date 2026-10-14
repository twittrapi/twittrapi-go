package twittrapi

import (
	"context"
	"encoding/json"
	"net/url"
)

// RankingMode sets the sort order of replies returned by [TweetsService.Get].
type RankingMode string

const (
	// RankingRelevance sorts replies by relevance. This is the API default.
	RankingRelevance RankingMode = "Relevance"
	// RankingRecency sorts replies newest first.
	RankingRecency RankingMode = "Recency"
	// RankingLikes sorts replies by like count.
	RankingLikes RankingMode = "Likes"
)

var rankingModes = []string{string(RankingRelevance), string(RankingRecency), string(RankingLikes)}

// TweetsService groups single tweet and engagement endpoints. Use it through
// [Client.Tweets].
type TweetsService struct{ client *Client }

// TweetParams selects a tweet for [TweetsService.Get].
type TweetParams struct {
	// TweetID is the numeric tweet ID. Required.
	TweetID string
	// RankingMode sets the sort order of the replies. Empty uses the API default
	// (Relevance).
	RankingMode RankingMode
	// Cursor loads more replies. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p TweetParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p TweetParams) WithCursor(cursor string) TweetParams {
	p.Cursor = cursor
	return p
}

// TweetPageParams selects a tweet and a page for retweeters and quotes.
type TweetPageParams struct {
	// TweetID is the numeric tweet ID. Required.
	TweetID string
	// Count is the page size, from 1 to 100. 0 uses the API default.
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p TweetPageParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p TweetPageParams) WithCursor(cursor string) TweetPageParams {
	p.Cursor = cursor
	return p
}

// LikersParams selects one of your own tweets and a page for [TweetsService.Likers].
type LikersParams struct {
	// TweetID is the numeric tweet ID. Required.
	TweetID string
	// Cookie is the Cookie header of the X account that posted the tweet. It must contain
	// auth_token and ct0. Required.
	Cookie string
	// Proxy is the proxy the request to X goes out through, as http://user:pass@host:port.
	// Required.
	Proxy string
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p LikersParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p LikersParams) WithCursor(cursor string) LikersParams {
	p.Cursor = cursor
	return p
}

// Get returns a tweet with its author, media, counts and reply thread. Paginated: the
// cursor loads more replies.
func (s *TweetsService) Get(ctx context.Context, p TweetParams) (json.RawMessage, error) {
	id, err := numericID("TweetID", p.TweetID)
	if err != nil {
		return nil, err
	}
	mode, err := oneOf("RankingMode", string(p.RankingMode), rankingModes)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/tweet/"+id, url.Values{"rankingMode": {mode}, "cursor": {p.Cursor}})
}

// Retweeters returns users who retweeted a tweet. Paginated.
func (s *TweetsService) Retweeters(ctx context.Context, p TweetPageParams) (json.RawMessage, error) {
	return s.list(ctx, "retweets", p)
}

// Likers returns users who liked a tweet. It works only for your own tweets: X shows who
// liked a tweet only to its author, so p.Cookie must belong to the account that posted it.
// For anyone else's tweet the list is empty. The request is a POST; cookie, proxy, count
// and cursor go in the JSON body. Paginated.
func (s *TweetsService) Likers(ctx context.Context, p LikersParams) (json.RawMessage, error) {
	id, err := numericID("TweetID", p.TweetID)
	if err != nil {
		return nil, err
	}
	cookie, err := nonEmpty("Cookie", p.Cookie)
	if err != nil {
		return nil, err
	}
	proxy, err := nonEmpty("Proxy", p.Proxy)
	if err != nil {
		return nil, err
	}
	if err := validCount(p.Count); err != nil {
		return nil, err
	}
	body := likersBody{Cookie: cookie, Proxy: proxy, Count: p.Count, Cursor: p.Cursor}
	return s.client.Post(ctx, "/tweet/"+id+"/favorites", nil, body)
}

type likersBody struct {
	Cookie string `json:"cookie"`
	Proxy  string `json:"proxy"`
	Count  int    `json:"count,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// Quotes returns tweets quoting a tweet. Paginated.
func (s *TweetsService) Quotes(ctx context.Context, p TweetPageParams) (json.RawMessage, error) {
	return s.list(ctx, "quotes", p)
}

func (s *TweetsService) list(ctx context.Context, suffix string, p TweetPageParams) (json.RawMessage, error) {
	id, err := numericID("TweetID", p.TweetID)
	if err != nil {
		return nil, err
	}
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/tweet/"+id+"/"+suffix, q)
}
