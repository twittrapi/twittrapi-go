package twittrapi

import (
	"context"
	"encoding/json"
	"net/url"
)

// ListsService groups X list endpoints. Use it through [Client.Lists].
type ListsService struct{ client *Client }

// ListPageParams selects a list and a page for [ListsService.Tweets] and
// [ListsService.Members].
type ListPageParams struct {
	// ListID is the numeric list ID. Required.
	ListID string
	// Count is the page size, from 1 to 100. 0 uses the API default.
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p ListPageParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p ListPageParams) WithCursor(cursor string) ListPageParams {
	p.Cursor = cursor
	return p
}

// Details returns a list's name, description, owner and counts.
func (s *ListsService) Details(ctx context.Context, listID string) (json.RawMessage, error) {
	id, err := numericID("listID", listID)
	if err != nil {
		return nil, err
	}
	return s.client.Get(ctx, "/lists/details", url.Values{"listId": {id}})
}

// Tweets returns tweets from a list's members, newest first. Paginated.
func (s *ListsService) Tweets(ctx context.Context, p ListPageParams) (json.RawMessage, error) {
	return s.page(ctx, "/lists/tweets", p)
}

// Members returns accounts that are members of a list. Paginated.
func (s *ListsService) Members(ctx context.Context, p ListPageParams) (json.RawMessage, error) {
	return s.page(ctx, "/lists/members", p)
}

func (s *ListsService) page(ctx context.Context, path string, p ListPageParams) (json.RawMessage, error) {
	id, err := numericID("ListID", p.ListID)
	if err != nil {
		return nil, err
	}
	q, err := pageQuery(p.Count, p.Cursor)
	if err != nil {
		return nil, err
	}
	q.Set("listId", id)
	return s.client.Get(ctx, path, q)
}
