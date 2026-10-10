package twittrapi

import (
	"context"
	"encoding/json"
)

// SearchCategory is the result tab for [SearchService.Tweets].
type SearchCategory string

const (
	// CategoryTop returns top results. This is the API default.
	CategoryTop SearchCategory = "Top"
	// CategoryLatest returns tweets newest first.
	CategoryLatest SearchCategory = "Latest"
	// CategoryPeople returns accounts instead of tweets.
	CategoryPeople SearchCategory = "People"
	// CategoryPhotos returns tweets with photos.
	CategoryPhotos SearchCategory = "Photos"
	// CategoryVideos returns tweets with videos.
	CategoryVideos SearchCategory = "Videos"
)

var searchCategories = []string{
	string(CategoryTop), string(CategoryLatest), string(CategoryPeople),
	string(CategoryPhotos), string(CategoryVideos),
}

// SearchService groups search endpoints. Use it through [Client.Search].
type SearchService struct{ client *Client }

// SearchParams describes a search for [SearchService.Tweets].
type SearchParams struct {
	// Query is the search query. Required. X search operators such as from:, lang:, since:,
	// min_faves: and -filter:replies work.
	Query string
	// Category is the result tab. Empty uses the API default (Top).
	Category SearchCategory
	// Count is the page size, from 1 to 100. 0 uses the API default.
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p SearchParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p SearchParams) WithCursor(cursor string) SearchParams {
	p.Cursor = cursor
	return p
}

// Tweets searches tweets, like the X search box (or people, with [CategoryPeople]).
// Paginated.
func (s *SearchService) Tweets(ctx context.Context, p SearchParams) (json.RawMessage, error) {
	query, err := nonEmpty("Query", p.Query)
	if err != nil {
		return nil, err
	}
	category, err := oneOf("Category", string(p.Category), searchCategories)
	if err != nil {
		return nil, err
	}
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	q.Set("searchItem", query)
	if category != "" {
		q.Set("category", category)
	}
	return s.client.Get(ctx, "/search", q)
}
