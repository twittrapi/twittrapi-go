package twittrapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// clearEnv unsets the key environment variables for the duration of the test.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvAPIKey, "")
	t.Setenv(EnvRapidAPIKey, "")
	t.Setenv(EnvRapidAPIHost, "")
	t.Setenv(EnvBaseURL, "")
}

// recorder captures the requests a test server receives.
type recorder struct {
	mu   sync.Mutex
	reqs []*http.Request
}

func (r *recorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, req)
}

func (r *recorder) all() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Request(nil), r.reqs...)
}

func (r *recorder) last(t *testing.T) *http.Request {
	t.Helper()
	all := r.all()
	if len(all) == 0 {
		t.Fatal("no request received")
	}
	return all[len(all)-1]
}

type response struct {
	status int
	body   string
	header map[string]string
}

// newServer returns a test server that answers with the given responses in order (the
// last one repeats) and records every request.
func newServer(t *testing.T, responses ...response) (*httptest.Server, *recorder) {
	t.Helper()
	if len(responses) == 0 {
		responses = []response{{status: 200, body: `{"ok":true}`}}
	}
	rec := &recorder{}
	var mu sync.Mutex
	i := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		mu.Lock()
		resp := responses[i]
		if i < len(responses)-1 {
			i++
		}
		mu.Unlock()
		for k, v := range resp.header {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

// newTestClient returns a direct-mode client pointed at srv with instant retries. The
// returned slice records the requested sleep durations.
func newTestClient(t *testing.T, srv *httptest.Server, opts ...Option) (*Client, *[]time.Duration) {
	t.Helper()
	clearEnv(t)
	all := append([]Option{WithAPIKey("test-key"), WithBaseURL(srv.URL)}, opts...)
	c, err := NewClient(all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var mu sync.Mutex
	sleeps := &[]time.Duration{}
	c.sleep = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		*sleeps = append(*sleeps, d)
		mu.Unlock()
		return ctx.Err()
	}
	return c, sleeps
}

func TestNewClientDirectKey(t *testing.T) {
	clearEnv(t)
	c, err := NewClient(WithAPIKey("  abc  "))
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMode() != AuthDirect || c.BaseURL() != DefaultBaseURL || c.apiKey != "abc" {
		t.Fatalf("got mode=%s base=%s key=%q", c.AuthMode(), c.BaseURL(), c.apiKey)
	}
	if c.MaxRetries() != DefaultMaxRetries {
		t.Fatalf("MaxRetries = %d", c.MaxRetries())
	}
}

func TestNewClientRapidAPIKey(t *testing.T) {
	clearEnv(t)
	t.Setenv(EnvBaseURL, "https://ignored.example.com")
	c, err := NewClient(WithRapidAPIKey("rk"))
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMode() != AuthRapidAPI || c.BaseURL() != RapidAPIBaseURL {
		t.Fatalf("got mode=%s base=%s", c.AuthMode(), c.BaseURL())
	}
}

func TestNewClientEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv(EnvRapidAPIKey, "rapid")
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMode() != AuthRapidAPI || c.apiKey != "rapid" {
		t.Fatalf("got mode=%s key=%q", c.AuthMode(), c.apiKey)
	}

	t.Setenv(EnvAPIKey, "direct")
	t.Setenv(EnvBaseURL, "https://staging.example.com/")
	c, err = NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMode() != AuthDirect || c.apiKey != "direct" || c.BaseURL() != "https://staging.example.com" {
		t.Fatalf("got mode=%s key=%q base=%s", c.AuthMode(), c.apiKey, c.BaseURL())
	}

	// Explicit options win over the environment.
	c, err = NewClient(WithRapidAPIKey("explicit"))
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthMode() != AuthRapidAPI || c.apiKey != "explicit" {
		t.Fatalf("got mode=%s key=%q", c.AuthMode(), c.apiKey)
	}
}

func TestNewClientErrors(t *testing.T) {
	clearEnv(t)
	if _, err := NewClient(); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("want ErrNoAPIKey, got %v", err)
	}
	if _, err := NewClient(WithAPIKey("   ")); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("blank key: want ErrNoAPIKey, got %v", err)
	}
	if _, err := NewClient(WithAPIKey("a"), WithRapidAPIKey("b")); err == nil {
		t.Fatal("want error for both keys")
	}
	if _, err := NewClient(WithAPIKey("a"), WithBaseURL("not a url")); err == nil {
		t.Fatal("want error for invalid base URL")
	}
}

func TestDirectModeRequest(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv, WithBaseURL(srv.URL+"/"))
	body, err := c.Users.GetByUsername(context.Background(), "@jack")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", body)
	}
	r := rec.last(t)
	if r.Method != http.MethodGet || r.URL.Path != "/user/by/username/jack" {
		t.Fatalf("got %s %s", r.Method, r.URL.Path)
	}
	if got := r.Header.Get("X-API-KEY"); got != "test-key" {
		t.Fatalf("X-API-KEY = %q", got)
	}
	if r.Header.Get("x-rapidapi-key") != "" || r.Header.Get("x-rapidapi-host") != "" {
		t.Fatal("RapidAPI headers sent in direct mode")
	}
	if r.Header.Get("Accept") != "application/json" {
		t.Fatalf("Accept = %q", r.Header.Get("Accept"))
	}
	if ua := r.Header.Get("User-Agent"); ua != "twittrapi-go/"+Version {
		t.Fatalf("User-Agent = %q", ua)
	}
}

func TestRapidAPIModeRequest(t *testing.T) {
	srv, rec := newServer(t)
	clearEnv(t)
	c, err := NewClient(WithRapidAPIKey("rk"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Users.GetByID(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	r := rec.last(t)
	if r.URL.Path != "/user/12" {
		t.Fatalf("path = %s", r.URL.Path)
	}
	if r.Header.Get("x-rapidapi-key") != "rk" || r.Header.Get("x-rapidapi-host") != RapidAPIHost {
		t.Fatalf("rapidapi headers = %q / %q", r.Header.Get("x-rapidapi-key"), r.Header.Get("x-rapidapi-host"))
	}
	if r.Header.Get("X-API-KEY") != "" {
		t.Fatal("X-API-KEY sent in RapidAPI mode")
	}
}

func TestRapidAPIHost(t *testing.T) {
	clearEnv(t)
	c, err := NewClient(WithRapidAPIKey("rk"))
	if err != nil {
		t.Fatal(err)
	}
	if c.RapidAPIHost() != RapidAPIHost {
		t.Fatalf("default host = %q", c.RapidAPIHost())
	}

	for _, in := range []string{
		"twttr.p.rapidapi.com",
		"https://twttr.p.rapidapi.com",
		"https://twttr.p.rapidapi.com/",
		"  HTTPS://twttr.p.rapidapi.com/ ",
		"http://twttr.p.rapidapi.com",
	} {
		c, err := NewClient(WithRapidAPIKey("rk"), WithRapidAPIHost(in))
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if c.RapidAPIHost() != "twttr.p.rapidapi.com" || c.BaseURL() != "https://twttr.p.rapidapi.com" {
			t.Fatalf("%q: host=%q base=%s", in, c.RapidAPIHost(), c.BaseURL())
		}
	}

	for _, in := range []string{
		"twttr.p.rapidapi.com/v1",
		"https://twttr.p.rapidapi.com/v1/",
		"twttr p.rapidapi.com",
		"https://",
		"user@twttr.p.rapidapi.com",
		"twttr.p.rapidapi.com?x=1",
	} {
		if _, err := NewClient(WithRapidAPIKey("rk"), WithRapidAPIHost(in)); err == nil {
			t.Fatalf("%q: want error", in)
		}
	}
}

func TestRapidAPIHostEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv(EnvRapidAPIKey, "rk")
	t.Setenv(EnvRapidAPIHost, "https://twttr.p.rapidapi.com/")
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.RapidAPIHost() != "twttr.p.rapidapi.com" || c.BaseURL() != "https://twttr.p.rapidapi.com" {
		t.Fatalf("host=%q base=%s", c.RapidAPIHost(), c.BaseURL())
	}

	// The option wins over the environment.
	c, err = NewClient(WithRapidAPIHost("other.p.rapidapi.com"))
	if err != nil {
		t.Fatal(err)
	}
	if c.RapidAPIHost() != "other.p.rapidapi.com" {
		t.Fatalf("host = %q", c.RapidAPIHost())
	}

	t.Setenv(EnvRapidAPIHost, "twttr.p.rapidapi.com/path")
	if _, err := NewClient(); err == nil {
		t.Fatal("want error for invalid RAPIDAPI_HOST")
	}
}

func TestRapidAPIHostWithBaseURL(t *testing.T) {
	srv, rec := newServer(t)
	clearEnv(t)
	c, err := NewClient(WithRapidAPIKey("rk"), WithRapidAPIHost("twttr.p.rapidapi.com"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != srv.URL {
		t.Fatalf("base = %s", c.BaseURL())
	}
	if _, err := c.Users.GetByID(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if got := rec.last(t).Header.Get("x-rapidapi-host"); got != "twttr.p.rapidapi.com" {
		t.Fatalf("x-rapidapi-host = %q", got)
	}
}

func TestRapidAPIHostIgnoredInDirectMode(t *testing.T) {
	srv, rec := newServer(t)
	clearEnv(t)
	t.Setenv(EnvRapidAPIHost, "not a host/")
	c, err := NewClient(WithAPIKey("dk"), WithRapidAPIHost("twttr.p.rapidapi.com"))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL() != DefaultBaseURL || c.RapidAPIHost() != "" {
		t.Fatalf("base=%s host=%q", c.BaseURL(), c.RapidAPIHost())
	}

	c, err = NewClient(WithAPIKey("dk"), WithRapidAPIHost("twttr.p.rapidapi.com"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Users.GetByID(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if got := rec.last(t).Header.Get("x-rapidapi-host"); got != "" {
		t.Fatalf("x-rapidapi-host = %q in direct mode", got)
	}
}

func TestCustomHTTPClientAndUserAgent(t *testing.T) {
	srv, rec := newServer(t)
	used := false
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used = true
		return http.DefaultTransport.RoundTrip(r)
	})}
	c, _ := newTestClient(t, srv, WithHTTPClient(hc), WithUserAgent("my-app/1.0"))
	if _, err := c.Trends.Locations(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !used {
		t.Fatal("custom HTTP client not used")
	}
	if ua := rec.last(t).Header.Get("User-Agent"); ua != "my-app/1.0" {
		t.Fatalf("User-Agent = %q", ua)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetDropsEmptyQueryValues(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	_, err := c.Get(context.Background(), "custom/endpoint", map[string][]string{"a": {"1"}, "b": {""}})
	if err != nil {
		t.Fatal(err)
	}
	r := rec.last(t)
	if r.URL.Path != "/custom/endpoint" || r.URL.RawQuery != "a=1" {
		t.Fatalf("got %s?%s", r.URL.Path, r.URL.RawQuery)
	}
}

func TestEmptyBodyIsNull(t *testing.T) {
	srv, _ := newServer(t, response{status: 200, body: ""})
	c, _ := newTestClient(t, srv)
	body, err := c.Trends.Locations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "null" {
		t.Fatalf("body = %s", body)
	}
}

func TestInvalidJSONBody(t *testing.T) {
	srv, _ := newServer(t, response{status: 200, body: "<html>oops</html>"})
	c, _ := newTestClient(t, srv)
	_, err := c.Trends.Locations(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 200 || !strings.Contains(apiErr.Message, "not valid JSON") {
		t.Fatalf("got %v", err)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status     int
		body       string
		check      func(error) bool
		wantDetail string
		wantMsg    string
	}{
		{400, `{"error":"bad id"}`, func(e error) bool { var x *ValidationError; return errors.As(e, &x) }, "bad id", "400 bad id"},
		{422, `{}`, func(e error) bool { var x *ValidationError; return errors.As(e, &x) }, "", "422 Invalid parameters"},
		{401, `{"message":"invalid key"}`, func(e error) bool { var x *AuthenticationError; return errors.As(e, &x) }, "invalid key", "401 invalid key"},
		{403, ``, func(e error) bool { var x *AuthenticationError; return errors.As(e, &x) }, "", "403 Access denied: the key may lack access to this endpoint"},
		{402, `{"error":"no balance"}`, func(e error) bool { var x *InsufficientBalanceError; return errors.As(e, &x) }, "no balance", "402 no balance"},
		{404, `{"detail":"gone"}`, func(e error) bool { var x *NotFoundError; return errors.As(e, &x) }, "gone", "404 gone"},
		{429, `{}`, func(e error) bool { var x *RateLimitError; return errors.As(e, &x) && x.RetryAfter == -1 }, "", "429 Rate limit exceeded"},
		{500, `upstream broke`, func(e error) bool { var x *ServerError; return errors.As(e, &x) }, "upstream broke", "500 upstream broke"},
		{503, `{}`, func(e error) bool { var x *ServerError; return errors.As(e, &x) }, "", "503 Server error"},
		{418, `{}`, func(e error) bool {
			var x *APIError
			var s *ServerError
			return errors.As(e, &x) && !errors.As(e, &s)
		}, "", "418 Request failed"},
	}
	for _, tc := range cases {
		srv, rec := newServer(t, response{status: tc.status, body: tc.body})
		c, _ := newTestClient(t, srv, WithMaxRetries(0))
		_, err := c.Users.GetByID(context.Background(), "12")
		if err == nil || !tc.check(err) {
			t.Errorf("%d: wrong error type %T: %v", tc.status, err, err)
			continue
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Errorf("%d: not an *APIError", tc.status)
			continue
		}
		if apiErr.StatusCode != tc.status || apiErr.Detail != tc.wantDetail || apiErr.Message != tc.wantMsg {
			t.Errorf("%d: got status=%d detail=%q msg=%q", tc.status, apiErr.StatusCode, apiErr.Detail, apiErr.Message)
		}
		if string(apiErr.Body) != tc.body {
			t.Errorf("%d: body = %q", tc.status, apiErr.Body)
		}
		if err.Error() != "twittrapi: "+tc.wantMsg {
			t.Errorf("%d: Error() = %q", tc.status, err.Error())
		}
		if n := len(rec.all()); n != 1 {
			t.Errorf("%d: %d requests with retries disabled", tc.status, n)
		}
	}
}

func TestLongTextBodyIsTruncated(t *testing.T) {
	long := strings.Repeat("x", 600)
	srv, _ := newServer(t, response{status: 400, body: long})
	c, _ := newTestClient(t, srv)
	_, err := c.Users.GetByID(context.Background(), "12")
	var v *ValidationError
	if !errors.As(err, &v) || len(v.Detail) != 503 || !strings.HasSuffix(v.Detail, "...") {
		t.Fatalf("got %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 0, false},
		{"  ", 0, false},
		{"5", 5 * time.Second, true},
		{"1.5", 1500 * time.Millisecond, true},
		{"-3", 0, false},
		{"soon", 0, false},
		{"Thu, 01 Jan 2026 12:00:30 GMT", 30 * time.Second, true},
		{"Thu, 01 Jan 2026 11:00:00 GMT", 0, true},
	}
	for _, tc := range cases {
		got, ok := ParseRetryAfter(tc.in, now)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ParseRetryAfter(%q) = %v, %v; want %v, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRetriesOn429And5xx(t *testing.T) {
	srv, rec := newServer(t,
		response{status: 429, body: `{}`, header: map[string]string{"Retry-After": "3"}},
		response{status: 502, body: `{}`},
		response{status: 200, body: `{"ok":1}`},
	)
	c, sleeps := newTestClient(t, srv)
	body, err := c.Users.GetByID(context.Background(), "12")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":1}` || len(rec.all()) != 3 {
		t.Fatalf("body=%s requests=%d", body, len(rec.all()))
	}
	if len(*sleeps) != 2 || (*sleeps)[0] != 3*time.Second {
		t.Fatalf("sleeps = %v", *sleeps)
	}
	// Second delay: base 500ms << 1 = 1s with 75-100% jitter.
	if d := (*sleeps)[1]; d < 750*time.Millisecond || d > time.Second {
		t.Fatalf("backoff = %v", d)
	}
}

func TestRetryAfterIsCapped(t *testing.T) {
	srv, _ := newServer(t,
		response{status: 429, body: `{}`, header: map[string]string{"Retry-After": "3600"}},
		response{status: 200, body: `{}`},
	)
	c, sleeps := newTestClient(t, srv)
	if _, err := c.Users.GetByID(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if (*sleeps)[0] != 60*time.Second {
		t.Fatalf("sleep = %v", (*sleeps)[0])
	}
}

func TestRetriesExhausted(t *testing.T) {
	srv, rec := newServer(t, response{status: 503, body: `{"error":"down"}`})
	c, sleeps := newTestClient(t, srv)
	_, err := c.Users.GetByID(context.Background(), "12")
	var s *ServerError
	if !errors.As(err, &s) || s.Detail != "down" {
		t.Fatalf("got %v", err)
	}
	if n := len(rec.all()); n != 3 {
		t.Fatalf("requests = %d, want 3 (1 + 2 retries)", n)
	}
	if len(*sleeps) != 2 {
		t.Fatalf("sleeps = %v", *sleeps)
	}
}

func TestWithMaxRetries(t *testing.T) {
	srv, rec := newServer(t, response{status: 500, body: `{}`})
	c, _ := newTestClient(t, srv, WithMaxRetries(4))
	_, _ = c.Users.GetByID(context.Background(), "12")
	if n := len(rec.all()); n != 5 {
		t.Fatalf("requests = %d, want 5", n)
	}
	c, _ = newTestClient(t, srv, WithMaxRetries(-1))
	if c.MaxRetries() != 0 {
		t.Fatalf("negative retries = %d", c.MaxRetries())
	}
}

func TestNoRetryOnOther4xx(t *testing.T) {
	for _, status := range []int{400, 401, 402, 403, 404, 422} {
		srv, rec := newServer(t, response{status: status, body: `{}`})
		c, sleeps := newTestClient(t, srv)
		if _, err := c.Users.GetByID(context.Background(), "12"); err == nil {
			t.Fatalf("%d: want error", status)
		}
		if n := len(rec.all()); n != 1 || len(*sleeps) != 0 {
			t.Fatalf("%d: requests=%d sleeps=%v", status, n, *sleeps)
		}
	}
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	c := &Client{retryDelay: DefaultRetryDelay}
	err := &ServerError{&APIError{StatusCode: 500}}
	for attempt, base := range []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		d := c.backoff(attempt, err)
		if d < base*3/4 || d > base {
			t.Errorf("attempt %d: %v not in [%v, %v]", attempt, d, base*3/4, base)
		}
	}
	if d := c.backoff(100, err); d > maxRetryDelay {
		t.Errorf("attempt 100: %v", d)
	}
}

func TestConnectionErrorIsRetried(t *testing.T) {
	// Grab a free port and close it so connections are refused.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	clearEnv(t)
	c, err := NewClient(WithAPIKey("k"), WithBaseURL("http://"+addr))
	if err != nil {
		t.Fatal(err)
	}
	var sleeps int
	c.sleep = func(ctx context.Context, d time.Duration) error { sleeps++; return nil }
	_, err = c.Users.GetByID(context.Background(), "12")
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatalf("want *ConnectionError, got %T: %v", err, err)
	}
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		t.Fatal("refused connection reported as timeout")
	}
	if connErr.Unwrap() == nil {
		t.Fatal("ConnectionError without cause")
	}
	if sleeps != 2 {
		t.Fatalf("sleeps = %d, want 2", sleeps)
	}
}

func TestTimeoutError(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(block); srv.Close() })

	c, sleeps := newTestClient(t, srv, WithTimeout(20*time.Millisecond), WithMaxRetries(1))
	_, err := c.Users.GetByID(context.Background(), "12")
	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("want *TimeoutError, got %T: %v", err, err)
	}
	var connErr *ConnectionError
	if !errors.As(err, &connErr) {
		t.Fatal("TimeoutError does not match *ConnectionError")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout does not wrap DeadlineExceeded: %v", err)
	}
	if len(*sleeps) != 1 {
		t.Fatalf("sleeps = %v, want 1 retry", *sleeps)
	}
}

func TestContextCanceled(t *testing.T) {
	srv, rec := newServer(t)
	c, _ := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Users.GetByID(ctx, "12")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		t.Fatal("canceled context reported as connection error")
	}
	if n := len(rec.all()); n != 0 {
		t.Fatalf("requests = %d", n)
	}
}

func TestContextCanceledDuringBackoff(t *testing.T) {
	srv, rec := newServer(t, response{status: 500, body: `{}`})
	clearEnv(t)
	c, err := NewClient(WithAPIKey("k"), WithBaseURL(srv.URL), WithRetryDelay(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = c.Users.GetByID(ctx, "12")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("backoff ignored context")
	}
	if n := len(rec.all()); n != 1 {
		t.Fatalf("requests = %d", n)
	}
}
