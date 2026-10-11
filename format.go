package twittrapi

import (
	"context"
	"net/url"
	"strings"
)

// Format selects the response format of the API.
type Format string

const (
	// FormatRaw returns X's raw JSON unchanged. This is the default.
	FormatRaw Format = "raw"
	// FormatSimple returns the opt-in simple format: flat camelCase objects with a fixed
	// schema in a {"data", "nextCursor", "previousCursor", "hasNextPage"} envelope (lists)
	// or a {"data"} envelope (single objects). It is sent as format=simple. See
	// https://twittrapi.com/docs/simple-format.
	FormatSimple Format = "simple"
)

// WithFormat sets the default response format for every request of the client. The default
// is [FormatRaw], which sends no format parameter. With [FormatSimple] every request carries
// format=simple. A single call can override it with [ContextWithFormat]. Unknown values are
// treated as [FormatRaw].
func WithFormat(f Format) Option {
	return func(c *clientConfig) { c.format = normalizeFormat(f) }
}

type formatKey struct{}

// ContextWithFormat returns a copy of ctx that makes requests sent with it use format f,
// whatever the client default is. Use it to request the simple format for one call:
//
//	body, err := client.Users.Tweets(twittrapi.ContextWithFormat(ctx, twittrapi.FormatSimple), params)
//
// [FetchSimple], [FetchSimplePage] and [PaginateSimple] do this for you.
func ContextWithFormat(ctx context.Context, f Format) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, formatKey{}, normalizeFormat(f))
}

func normalizeFormat(f Format) Format {
	if strings.EqualFold(strings.TrimSpace(string(f)), string(FormatSimple)) {
		return FormatSimple
	}
	return FormatRaw
}

// Format returns the client's default response format.
func (c *Client) Format() Format { return c.format }

// requestFormat returns the format for a request: the one set on ctx, else the client
// default.
func (c *Client) requestFormat(ctx context.Context) Format {
	if f, ok := ctx.Value(formatKey{}).(Format); ok {
		return f
	}
	return c.format
}

// applyFormat adds format=simple to query when the request asks for the simple format and
// the caller did not set a format parameter itself. The raw format sends nothing, so raw
// requests are exactly what they were before the option existed.
func (c *Client) applyFormat(ctx context.Context, query url.Values) url.Values {
	if c.requestFormat(ctx) != FormatSimple || query.Get("format") != "" {
		return query
	}
	out := make(url.Values, len(query)+1)
	for k, v := range query {
		out[k] = v
	}
	out.Set("format", string(FormatSimple))
	return out
}
