package twittrapi

import (
	"context"
	"encoding/json"
	"net/url"
)

// CommunitiesService groups X Communities endpoints. Use it through [Client.Communities].
type CommunitiesService struct{ client *Client }

// CommunityPageParams selects a community and a page for [CommunitiesService.Tweets].
type CommunityPageParams struct {
	// CommunityID is the numeric community ID. Required.
	CommunityID string
	// Count is the page size, from 1 to 100. 0 uses the API default.
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p CommunityPageParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p CommunityPageParams) WithCursor(cursor string) CommunityPageParams {
	p.Cursor = cursor
	return p
}

// CommunitySearchParams describes a search for [CommunitiesService.Search].
type CommunitySearchParams struct {
	// Query is the keyword to search for. Required.
	Query string
	// Count is the page size, from 1 to 100. 0 uses the API default.
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p CommunitySearchParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p CommunitySearchParams) WithCursor(cursor string) CommunitySearchParams {
	p.Cursor = cursor
	return p
}

// Get returns a community's name, description, rules, member count and admins.
func (s *CommunitiesService) Get(ctx context.Context, communityID string) (json.RawMessage, error) {
	id, err := numericID("communityID", communityID)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/communities/"+id, nil)
}

// Tweets returns tweets posted in a community. Paginated.
func (s *CommunitiesService) Tweets(ctx context.Context, p CommunityPageParams) (json.RawMessage, error) {
	id, err := numericID("CommunityID", p.CommunityID)
	if err != nil {
		return nil, err
	}
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/communities/"+id+"/tweets", q)
}

// Search searches communities by keyword. Paginated.
func (s *CommunitiesService) Search(ctx context.Context, p CommunitySearchParams) (json.RawMessage, error) {
	query, err := nonEmpty("Query", p.Query)
	if err != nil {
		return nil, err
	}
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/communities/search/"+url.PathEscape(query), q)
}
