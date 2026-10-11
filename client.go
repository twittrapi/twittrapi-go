package twittrapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the base URL of the direct TwittrAPI API.
	DefaultBaseURL = "https://api.twittrapi.com"
	// RapidAPIHost is the host of TwittrAPI on RapidAPI, sent as x-rapidapi-host.
	RapidAPIHost = "twitter-aio.p.rapidapi.com"
	// RapidAPIBaseURL is the base URL used with a RapidAPI key.
	RapidAPIBaseURL = "https://" + RapidAPIHost
	// DefaultTimeout is the default timeout for a single attempt of a request.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxRetries is the default number of retries for 429, 5xx, timeouts and
	// network errors.
	DefaultMaxRetries = 2
	// DefaultRetryDelay is the default base delay for exponential backoff between retries.
	DefaultRetryDelay = 500 * time.Millisecond

	// EnvAPIKey is the environment variable read for a direct TwittrAPI key.
	EnvAPIKey = "TWITTRAPI_API_KEY"
	// EnvRapidAPIKey is the environment variable read for a RapidAPI key.
	EnvRapidAPIKey = "RAPIDAPI_KEY"
	// EnvBaseURL is the environment variable read for a base URL override in direct mode.
	EnvBaseURL = "TWITTRAPI_BASE_URL"

	maxRetryDelay = 8 * time.Second
	maxRetryAfter = 60 * time.Second
)

// AuthMode says which kind of API key a [Client] uses.
type AuthMode string

const (
	// AuthDirect sends the key in the X-API-KEY header to the TwittrAPI API.
	AuthDirect AuthMode = "direct"
	// AuthRapidAPI sends the key in the x-rapidapi-key header to RapidAPI.
	AuthRapidAPI AuthMode = "rapidapi"
)

// ErrNoAPIKey is returned by [NewClient] when no API key was passed and none was found in
// the TWITTRAPI_API_KEY or RAPIDAPI_KEY environment variables.
var ErrNoAPIKey = errors.New("twittrapi: no API key found: use WithAPIKey or WithRapidAPIKey, " +
	"or set TWITTRAPI_API_KEY or RAPIDAPI_KEY (get a key at https://twittrapi.com)")

// Client is a TwittrAPI client. Create one with [NewClient]. A Client is safe for concurrent
// use by multiple goroutines.
type Client struct {
	// Users groups user profile, timeline and follower endpoints.
	Users *UsersService
	// Tweets groups single tweet and engagement endpoints.
	Tweets *TweetsService
	// Search groups search endpoints.
	Search *SearchService
	// Lists groups X list endpoints.
	Lists *ListsService
	// Communities groups X Communities endpoints.
	Communities *CommunitiesService
	// Trends groups trending topic endpoints.
	Trends *TrendsService

	authMode   AuthMode
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration
	userAgent  string
	format     Format

	// sleep waits for d or until ctx is done. Replaced in tests.
	sleep func(ctx context.Context, d time.Duration) error
}

// Option configures a [Client]. Pass options to [NewClient].
type Option func(*clientConfig)

type clientConfig struct {
	apiKey      string
	rapidAPIKey string
	baseURL     string
	httpClient  *http.Client
	timeout     time.Duration
	maxRetries  int
	retryDelay  time.Duration
	userAgent   string
	format      Format
}

// WithAPIKey sets a direct TwittrAPI key, sent in the X-API-KEY header to
// https://api.twittrapi.com. Get a key at https://twittrapi.com.
func WithAPIKey(key string) Option {
	return func(c *clientConfig) { c.apiKey = strings.TrimSpace(key) }
}

// WithRapidAPIKey sets a RapidAPI key, sent in the x-rapidapi-key header (with
// x-rapidapi-host) to https://twitter-aio.p.rapidapi.com.
func WithRapidAPIKey(key string) Option {
	return func(c *clientConfig) { c.rapidAPIKey = strings.TrimSpace(key) }
}

// WithBaseURL overrides the API base URL. Without it the client uses [DefaultBaseURL] in
// direct mode (or TWITTRAPI_BASE_URL, if set) and [RapidAPIBaseURL] in RapidAPI mode.
func WithBaseURL(baseURL string) Option {
	return func(c *clientConfig) { c.baseURL = strings.TrimSpace(baseURL) }
}

// WithHTTPClient sets the [http.Client] used to send requests. The default is a new client
// with no timeout of its own; the per-attempt timeout comes from [WithTimeout].
func WithHTTPClient(hc *http.Client) Option {
	return func(c *clientConfig) { c.httpClient = hc }
}

// WithTimeout sets the timeout for each attempt of a request. The default is
// [DefaultTimeout]. A value of 0 or less disables the per-attempt timeout; the request
// context still applies.
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) { c.timeout = d }
}

// WithMaxRetries sets how often a request is retried after a 429, a 5xx status, a timeout or
// a network error. The default is [DefaultMaxRetries]. Use 0 to disable retries. Negative
// values are treated as 0.
func WithMaxRetries(n int) Option {
	return func(c *clientConfig) { c.maxRetries = n }
}

// WithRetryDelay sets the base delay for exponential backoff between retries. The delay
// doubles after each attempt, up to 8 seconds, with a little random jitter. The default is
// [DefaultRetryDelay]. A Retry-After header on a 429 response takes precedence (capped at
// 60 seconds).
func WithRetryDelay(d time.Duration) Option {
	return func(c *clientConfig) { c.retryDelay = d }
}

// WithUserAgent replaces the User-Agent header. The default is "twittrapi-go/<version>".
func WithUserAgent(ua string) Option {
	return func(c *clientConfig) { c.userAgent = ua }
}

func envValue(name string) string { return strings.TrimSpace(os.Getenv(name)) }

// NewClient returns a new client configured by opts.
//
// The API key is taken from, in order: [WithAPIKey], [WithRapidAPIKey], the
// TWITTRAPI_API_KEY environment variable and the RAPIDAPI_KEY environment variable. Passing
// both WithAPIKey and WithRapidAPIKey is an error. When no key is found, NewClient returns
// [ErrNoAPIKey].
func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
		retryDelay: DefaultRetryDelay,
		userAgent:  "twittrapi-go/" + Version,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	var mode AuthMode
	var key string
	switch {
	case cfg.apiKey != "" && cfg.rapidAPIKey != "":
		return nil, errors.New("twittrapi: pass either WithAPIKey or WithRapidAPIKey, not both")
	case cfg.apiKey != "":
		mode, key = AuthDirect, cfg.apiKey
	case cfg.rapidAPIKey != "":
		mode, key = AuthRapidAPI, cfg.rapidAPIKey
	case envValue(EnvAPIKey) != "":
		mode, key = AuthDirect, envValue(EnvAPIKey)
	case envValue(EnvRapidAPIKey) != "":
		mode, key = AuthRapidAPI, envValue(EnvRapidAPIKey)
	default:
		return nil, ErrNoAPIKey
	}

	baseURL := cfg.baseURL
	if baseURL == "" && mode == AuthDirect {
		baseURL = envValue(EnvBaseURL)
	}
	if baseURL == "" {
		if mode == AuthRapidAPI {
			baseURL = RapidAPIBaseURL
		} else {
			baseURL = DefaultBaseURL
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if u, err := url.Parse(baseURL); err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("twittrapi: invalid base URL %q", baseURL)
	}

	hc := cfg.httpClient
	if hc == nil {
		hc = &http.Client{}
	}
	if cfg.maxRetries < 0 {
		cfg.maxRetries = 0
	}
	if cfg.retryDelay < 0 {
		cfg.retryDelay = 0
	}

	c := &Client{
		authMode:   mode,
		apiKey:     key,
		baseURL:    baseURL,
		httpClient: hc,
		timeout:    cfg.timeout,
		maxRetries: cfg.maxRetries,
		retryDelay: cfg.retryDelay,
		userAgent:  cfg.userAgent,
		format:     normalizeFormat(cfg.format),
		sleep:      sleepContext,
	}
	c.Users = &UsersService{client: c}
	c.Tweets = &TweetsService{client: c}
	c.Search = &SearchService{client: c}
	c.Lists = &ListsService{client: c}
	c.Communities = &CommunitiesService{client: c}
	c.Trends = &TrendsService{client: c}
	return c, nil
}

// AuthMode reports whether the client uses a direct TwittrAPI key or a RapidAPI key.
func (c *Client) AuthMode() AuthMode { return c.authMode }

// BaseURL returns the base URL requests are sent to, without a trailing slash.
func (c *Client) BaseURL() string { return c.baseURL }

// MaxRetries returns the number of retries configured for this client.
func (c *Client) MaxRetries() int { return c.maxRetries }

func (c *Client) setHeaders(h http.Header) {
	h.Set("Accept", "application/json")
	h.Set("User-Agent", c.userAgent)
	if c.authMode == AuthRapidAPI {
		h.Set("x-rapidapi-key", c.apiKey)
		h.Set("x-rapidapi-host", RapidAPIHost)
	} else {
		h.Set("X-API-KEY", c.apiKey)
	}
}

// buildURL joins the base URL, an already escaped path and the query. Empty query values
// are dropped.
func (c *Client) buildURL(path string, query url.Values) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := c.baseURL + path
	clean := url.Values{}
	for k, vs := range query {
		for _, v := range vs {
			if v != "" {
				clean.Add(k, v)
			}
		}
	}
	if len(clean) > 0 {
		u += "?" + clean.Encode()
	}
	return u
}

// Get sends an authenticated GET request to path (relative to the base URL, already
// escaped) with the given query parameters and returns the raw JSON body. It applies the
// same retries and error mapping as the service methods, so it can be used for endpoints
// this package does not wrap yet. Empty query values are dropped.
//
// When the request uses the simple format (see [WithFormat] and [ContextWithFormat]),
// format=simple is added to the query unless it already has a format parameter.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	target := c.buildURL(path, c.applyFormat(ctx, query))
	for attempt := 0; ; attempt++ {
		body, err := c.attempt(ctx, target)
		if err == nil {
			return body, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt >= c.maxRetries || !isRetryable(err) {
			return nil, err
		}
		if serr := c.sleep(ctx, c.backoff(attempt, err)); serr != nil {
			return nil, serr
		}
	}
}

func isRetryable(err error) bool {
	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
	}
	return false
}

func (c *Client) backoff(attempt int, err error) time.Duration {
	var rl *RateLimitError
	if errors.As(err, &rl) && rl.RetryAfter >= 0 {
		if rl.RetryAfter > maxRetryAfter {
			return maxRetryAfter
		}
		return rl.RetryAfter
	}
	d := c.retryDelay << uint(attempt)
	if d > maxRetryDelay || d < 0 {
		d = maxRetryDelay
	}
	// Jitter: 75% to 100% of the base delay.
	return time.Duration(float64(d) * (0.75 + rand.Float64()*0.25))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) attempt(ctx context.Context, target string) (json.RawMessage, error) {
	attemptCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		attemptCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("twittrapi: building request: %w", err)
	}
	c.setHeaders(req.Header)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(ctx, attemptCtx, err, "Could not reach TwittrAPI")
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, c.transportError(ctx, attemptCtx, err, "Connection lost while reading the response")
	}

	if resp.StatusCode >= 400 {
		return nil, errorFromResponse(resp.StatusCode, data, resp.Header)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(data) {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("%d Response was not valid JSON", resp.StatusCode),
			Body:       data,
			Header:     resp.Header,
		}
	}
	return json.RawMessage(data), nil
}

// transportError turns an error from the HTTP client into a ConnectionError or TimeoutError.
// When the caller's context is done, its error is returned unchanged.
func (c *Client) transportError(ctx, attemptCtx context.Context, err error, msg string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var netErr net.Error
	if errors.Is(attemptCtx.Err(), context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return &TimeoutError{ConnectionError: &ConnectionError{
			Message: fmt.Sprintf("Request timed out after %s", c.timeout),
			Err:     err,
		}}
	}
	return &ConnectionError{Message: msg, Err: err}
}
