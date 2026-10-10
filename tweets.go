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

// TweetPageParams selects a tweet and a page for retweeters, likers and quotes.
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

// Likers returns users who liked a tweet. X hides likers for many tweets, so the result can
// be empty. Paginated.
func (s *TweetsService) Likers(ctx context.Context, p TweetPageParams) (json.RawMessage, error) {
	return s.list(ctx, "favorites", p)
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
