package twittrapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// DefaultMaxPages is the number of pages [Paginate] fetches when
// [PaginateOptions.MaxPages] is 0. Every page is a billed request.
const DefaultMaxPages = 10

// NoPageLimit can be used as [PaginateOptions.MaxPages] to fetch pages until the cursor runs
// out.
const NoPageLimit = -1

// maxNodes guards against pathological input; real responses are far smaller.
const maxNodes = 500_000

// Cursors holds the pagination cursors found in a raw X timeline response.
type Cursors struct {
	// Next is the bottom cursor. Pass it as Cursor to get the next (older) page.
	Next string
	// Previous is the top cursor. It points at newer items.
	Previous string
}

// node is a JSON value that keeps object keys in document order, which encoding/json maps
// do not.
type node struct {
	kind byte // 'o' object, 'a' array, 's' string, 'x' anything else
	keys []string
	vals []*node
	str  string
}

func (n *node) get(key string) *node {
	if n == nil || n.kind != 'o' {
		return nil
	}
	for i, k := range n.keys {
		if k == key {
			return n.vals[i]
		}
	}
	return nil
}

func (n *node) stringValue() (string, bool) {
	if n == nil || n.kind != 's' {
		return "", false
	}
	return n.str, true
}

func parseOrdered(data []byte) (*node, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	count := 0
	var parse func() (*node, error)
	parse = func() (*node, error) {
		count++
		if count > maxNodes {
			return nil, errors.New("twittrapi: JSON too large to scan")
		}
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				n := &node{kind: 'o'}
				for dec.More() {
					kt, err := dec.Token()
					if err != nil {
						return nil, err
					}
					key, _ := kt.(string)
					v, err := parse()
					if err != nil {
						return nil, err
					}
					n.keys = append(n.keys, key)
					n.vals = append(n.vals, v)
				}
				if _, err := dec.Token(); err != nil {
					return nil, err
				}
				return n, nil
			case '[':
				n := &node{kind: 'a'}
				for dec.More() {
					v, err := parse()
					if err != nil {
						return nil, err
					}
					n.vals = append(n.vals, v)
				}
				if _, err := dec.Token(); err != nil {
					return nil, err
				}
				return n, nil
			}
			return nil, errors.New("twittrapi: unexpected JSON delimiter")
		case string:
			return &node{kind: 's', str: t}, nil
		default:
			return &node{kind: 'x'}, nil
		}
	}
	root, err := parse()
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("twittrapi: trailing data after JSON value")
	}
	return root, nil
}

// walk calls fn for every object in n in document order (pre-order, depth first), without
// recursion. It stops early when fn returns false.
func walk(n *node, fn func(obj *node) bool) {
	stack := []*node{n}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == nil {
			continue
		}
		switch cur.kind {
		case 'o':
			if !fn(cur) {
				return
			}
			fallthrough
		case 'a':
			for i := len(cur.vals) - 1; i >= 0; i-- {
				if k := cur.vals[i].kind; k == 'o' || k == 'a' {
					stack = append(stack, cur.vals[i])
				}
			}
		}
	}
}

// cursorEntryValue reads content.value or content.itemContent.value from a timeline entry.
func cursorEntryValue(entry *node) string {
	content := entry.get("content")
	if s, ok := content.get("value").stringValue(); ok && s != "" {
		return s
	}
	if s, ok := content.get("itemContent").get("value").stringValue(); ok && s != "" {
		return s
	}
	return ""
}

// ExtractCursors finds the pagination cursors in X's raw GraphQL JSON.
//
// X puts cursors in timeline entries shaped like {"cursorType": "Bottom" | "Top", "value":
// "..."}, usually inside an entry whose entryId starts with "cursor-bottom-" or
// "cursor-top-". Later pages often carry them in TimelineReplaceEntry instructions instead
// of the entry list; the scan covers both. Plain next_cursor / nextCursor fields (other than
// "0") are used as a fallback. The first match in document order wins.
//
// This is best-effort: it depends on X's response shape, which can change without notice.
// Invalid JSON gives empty Cursors.
func ExtractCursors(data []byte) Cursors {
	var out Cursors
	root, err := parseOrdered(data)
	if err != nil {
		return out
	}
	fallback := ""
	walk(root, func(obj *node) bool {
		typ, ok := obj.get("cursorType").stringValue()
		if !ok {
			typ, _ = obj.get("cursor_type").stringValue()
		}
		if v, ok := obj.get("value").stringValue(); ok && v != "" {
			if typ == "Bottom" && out.Next == "" {
				out.Next = v
			}
			if typ == "Top" && out.Previous == "" {
				out.Previous = v
			}
		}
		if id, ok := obj.get("entryId").stringValue(); ok {
			if v := cursorEntryValue(obj); v != "" {
				if strings.HasPrefix(id, "cursor-bottom-") && out.Next == "" {
					out.Next = v
				}
				if strings.HasPrefix(id, "cursor-top-") && out.Previous == "" {
					out.Previous = v
				}
			}
		}
		if fallback == "" {
			for _, key := range []string{"next_cursor", "nextCursor"} {
				if v, ok := obj.get(key).stringValue(); ok && v != "" && v != "0" {
					fallback = v
					break
				}
			}
		}
		return out.Next == "" || out.Previous == ""
	})
	if out.Next == "" {
		out.Next = fallback
	}
	return out
}

// ExtractBottomCursor returns the bottom ("next page") cursor of a raw X timeline response,
// or "" if none was found. See [ExtractCursors].
func ExtractBottomCursor(data []byte) string {
	return ExtractCursors(data).Next
}

// CountTimelineEntries counts timeline entries (objects with a string entryId) in a raw
// response. Entries whose entryId starts with "cursor-" are counted as cursors, all others
// as items. [Paginate] uses this to stop when X keeps returning a cursor but no results.
func CountTimelineEntries(data []byte) (items, cursors int) {
	root, err := parseOrdered(data)
	if err != nil {
		return 0, 0
	}
	stack := []*node{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur.kind == 'o' {
			if id, ok := cur.get("entryId").stringValue(); ok {
				if strings.HasPrefix(id, "cursor-") {
					cursors++
				} else {
					items++
				}
				continue // entries do not nest other top-level entries
			}
		}
		for _, v := range cur.vals {
			if v.kind == 'o' || v.kind == 'a' {
				stack = append(stack, v)
			}
		}
	}
	return items, cursors
}

// sliceItems reports whether a raw response uses X's slice shape (an "items_results" array,
// as in community search, community members and job search) and how many items those arrays
// hold in total.
func sliceItems(data []byte) (isSlice bool, items int) {
	root, err := parseOrdered(data)
	if err != nil {
		return false, 0
	}
	walk(root, func(obj *node) bool {
		if v := obj.get("items_results"); v != nil && v.kind == 'a' {
			isSlice = true
			items += len(v.vals)
		}
		return true
	})
	return isSlice, items
}

// Page is one page returned by a [PageIterator].
type Page struct {
	// Data is the raw JSON of this page.
	Data json.RawMessage
	// Number is the 1-based page number within this iteration.
	Number int
	// Cursor is the cursor used to request this page ("" for the first page unless one was
	// passed in the params).
	Cursor string
	// NextCursor is the bottom cursor found in this page, or "" if none was found.
	NextCursor string
}

// PaginateOptions configures [Paginate].
type PaginateOptions struct {
	// MaxPages is the maximum number of pages to fetch. Every page is a billed request, so 0
	// means [DefaultMaxPages] (10). Use [NoPageLimit] (or any negative value) to keep going
	// until the cursor runs out.
	MaxPages int
}

// PageFunc is a paginated method such as client.Users.Followers or client.Search.Tweets.
type PageFunc[P any] func(ctx context.Context, params P) (json.RawMessage, error)

// PageIterator walks the pages of a paginated method. Create one with [Paginate], then call
// Next until it returns false and check Err:
//
//	for it.Next() {
//		page := it.Page()
//		// ...
//	}
//	if err := it.Err(); err != nil {
//		// handle error
//	}
//
// A PageIterator is not safe for concurrent use.
type PageIterator struct {
	fetch    func(ctx context.Context, cursor string) (json.RawMessage, error)
	ctx      context.Context
	maxPages int
	cursor   string
	seen     map[string]bool
	page     Page
	err      error
	done     bool
}

// Paginate returns an iterator over the pages of fn, starting with params and following the
// bottom cursor of each response. fn is a paginated method value such as
// client.Users.Followers.
//
// Iteration stops when any of these is true:
//   - MaxPages pages have been fetched (default 10),
//   - no bottom cursor is found,
//   - the cursor repeats,
//   - a page contains timeline entries but all of them are cursors (no results),
//   - a slice-shaped page (items_results + slice_info.next_cursor) has no items,
//   - a request fails (see [PageIterator.Err]).
//
// Cursor detection is best-effort; see [ExtractCursors]. opts may be nil.
func Paginate[P Pageable[P]](ctx context.Context, fn PageFunc[P], params P, opts *PaginateOptions) *PageIterator {
	maxPages := DefaultMaxPages
	if opts != nil && opts.MaxPages != 0 {
		maxPages = opts.MaxPages
	}
	if ctx == nil {
		ctx = context.Background()
	}
	it := &PageIterator{
		fetch: func(ctx context.Context, cursor string) (json.RawMessage, error) {
			return fn(ctx, params.WithCursor(cursor))
		},
		ctx:      ctx,
		maxPages: maxPages,
		cursor:   params.PageCursor(),
		seen:     map[string]bool{},
	}
	if it.cursor != "" {
		it.seen[it.cursor] = true
	}
	return it
}

// Next fetches the next page. It returns false when iteration is over, either because a
// stop condition was reached or because of an error; check [PageIterator.Err].
func (it *PageIterator) Next() bool {
	if it.done {
		return false
	}
	if it.maxPages > 0 && it.page.Number >= it.maxPages {
		it.done = true
		return false
	}
	if it.page.Number > 0 {
		// Decide whether the previous page allows another request.
		next := it.page.NextCursor
		if next == "" || it.seen[next] {
			it.done = true
			return false
		}
		if items, cursors := CountTimelineEntries(it.page.Data); items == 0 && cursors > 0 {
			it.done = true
			return false
		}
		if isSlice, items := sliceItems(it.page.Data); isSlice && items == 0 {
			it.done = true
			return false
		}
		it.seen[next] = true
		it.cursor = next
	}
	data, err := it.fetch(it.ctx, it.cursor)
	if err != nil {
		it.err = err
		it.done = true
		return false
	}
	it.page = Page{
		Data:       data,
		Number:     it.page.Number + 1,
		Cursor:     it.cursor,
		NextCursor: ExtractBottomCursor(data),
	}
	return true
}

// Page returns the page fetched by the last successful call to [PageIterator.Next].
func (it *PageIterator) Page() Page { return it.page }

// Err returns the error that stopped the iteration, or nil.
func (it *PageIterator) Err() error { return it.err }

// ForEach calls fn for every remaining page. It stops at the first error from a request or
// from fn and returns it.
func (it *PageIterator) ForEach(fn func(Page) error) error {
	for it.Next() {
		if err := fn(it.Page()); err != nil {
			it.done = true
			return err
		}
	}
	return it.Err()
}
