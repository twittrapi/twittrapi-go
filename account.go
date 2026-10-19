package twittrapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// XSession is a logged-in X session used by [AccountService]. Account actions run on X as
// this account.
//
// The cookie and proxy are secrets. [XSession.String] and [XSession.GoString] mask them, so
// printing a session with fmt or a logger does not reveal them.
type XSession struct {
	// Cookie is the Cookie header of a logged-in x.com session, such as
	// "auth_token=...; ct0=...; twid=u%3D...". auth_token and ct0 are required; twid (the
	// account ID) is required for direct messages and likes. Required.
	Cookie string
	// Proxy is the proxy the requests to X go out through, as http://user:pass@host:port.
	// Use one proxy per account. Required.
	Proxy string
}

// String returns the session with every cookie value and the proxy credentials masked,
// for example XSession{Cookie: "auth_token=***; ct0=***", Proxy: "http://***@host:8080"}.
func (s XSession) String() string {
	return fmt.Sprintf("XSession{Cookie: %q, Proxy: %q}", maskCookie(s.Cookie), maskProxy(s.Proxy))
}

// GoString returns the same masked text as [XSession.String], so %#v masks too.
func (s XSession) GoString() string { return s.String() }

// maskCookie keeps the cookie names and replaces every value with ***.
func maskCookie(cookie string) string {
	if strings.TrimSpace(cookie) == "" {
		return ""
	}
	var parts []string
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, _, found := strings.Cut(part, "=")
		if !found {
			parts = append(parts, "***")
			continue
		}
		parts = append(parts, strings.TrimSpace(name)+"=***")
	}
	return strings.Join(parts, "; ")
}

// maskProxy keeps the scheme, host and port and replaces the user info with ***. A value
// that does not parse as a URL with a host is masked completely.
func maskProxy(proxy string) string {
	p := strings.TrimSpace(proxy)
	if p == "" {
		return ""
	}
	u, err := url.Parse(p)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "***"
	}
	masked := u.Scheme + "://"
	if u.User != nil {
		masked += "***@"
	}
	return masked + u.Host
}

// AccountService groups account actions (write endpoints): posting, liking, following,
// bookmarks, lists, communities, drafts, articles, media uploads, profile edits and direct
// messages, run as the X account of an [XSession]. Get one with [Client.Account].
//
// Every call sends a POST to /client/... with the session's cookie and proxy in the JSON
// body. Responses are returned as received; most are the wrapper {"status_code": ...,
// "response": "<X body as a string>"}, which [UnwrapAccountResponse] decodes. When X
// answers 4xx or 5xx, the API uses X's status and the call returns the matching typed
// error with the wrapper in [APIError.Body].
//
// Actions that create something (a tweet, reply, upload, list, community, draft, report or
// message) and every article change (create, update, publish, unpublish, delete) are
// retried only after a 429, never after a timeout, network error or 5xx, because the
// action may already have run. Other actions are retried as usual.
//
// An AccountService is safe for concurrent use. It never prints the session: String and
// GoString are masked.
type AccountService struct {
	client     *Client
	session    XSession
	dmPasscode string
}

// Account returns an [AccountService] that runs account actions as the X account of
// session. The session is checked on each call: an empty Cookie or Proxy returns
// [ErrInvalidParams] without sending a request.
//
//	acct := client.Account(twittrapi.XSession{
//		Cookie: os.Getenv("X_COOKIE"),
//		Proxy:  os.Getenv("X_PROXY"),
//	})
//	_, err := acct.Like(ctx, "1790000000000000000")
func (c *Client) Account(session XSession) *AccountService {
	return &AccountService{client: c, session: session}
}

// WithDMPasscode returns a copy of the service that sends passcode, the account's X Chat
// passcode, with direct message calls. Reading a conversation, sending a message and
// changing disappearing messages need it; [AccountService.DMInbox] uses it to include the
// latest messages. The passcode is masked like the session when the service is printed.
func (a *AccountService) WithDMPasscode(passcode string) *AccountService {
	cp := *a
	cp.dmPasscode = passcode
	return &cp
}

// String describes the service with the session and passcode masked.
func (a *AccountService) String() string {
	if a == nil {
		return "AccountService(nil)"
	}
	passcode := ""
	if a.dmPasscode != "" {
		passcode = "***"
	}
	return fmt.Sprintf("AccountService{Session: %s, DMPasscode: %q}", a.session, passcode)
}

// GoString returns the same masked text as [AccountService.String].
func (a *AccountService) GoString() string { return a.String() }

// UnwrapAccountResponse decodes the wrapper most account actions return,
// {"status_code": <X status>, "response": "<X body as a string>"}. It returns X's status and
// X's body (usually JSON; decode it a second time). ok is false when data is not such a
// wrapper, for example a direct-message answer or a simple-format body.
func UnwrapAccountResponse(data []byte) (statusCode int, body []byte, ok bool) {
	var w struct {
		StatusCode *int            `json:"status_code"`
		Response   json.RawMessage `json:"response"`
	}
	if json.Unmarshal(data, &w) != nil || w.StatusCode == nil {
		return 0, nil, false
	}
	var s string
	if json.Unmarshal(w.Response, &s) != nil {
		return 0, nil, false
	}
	return *w.StatusCode, []byte(s), true
}

// pageBody returns X's body inside an account wrapper, or data unchanged.
func pageBody(data []byte) []byte {
	if _, inner, ok := UnwrapAccountResponse(data); ok && json.Valid(inner) {
		return inner
	}
	return data
}

// Ptr returns a pointer to v. It helps fill optional fields such as
// [UpdateListParams.Name] or [ProfileUpdate.Description]:
//
//	acct.UpdateProfile(ctx, twittrapi.ProfileUpdate{Description: twittrapi.Ptr("Hello")})
func Ptr[T any](v T) *T { return &v }

// fields is a request body without the session.
type fields map[string]any

// send validates the session and POSTs fields plus cookie and proxy to path.
func (a *AccountService) send(ctx context.Context, path string, f fields, idempotent bool) (json.RawMessage, error) {
	cookie, err := nonEmpty("Cookie", a.session.Cookie)
	if err != nil {
		return nil, err
	}
	proxy, err := nonEmpty("Proxy", a.session.Proxy)
	if err != nil {
		return nil, err
	}
	body := fields{"cookie": cookie, "proxy": proxy}
	for k, v := range f {
		body[k] = v
	}
	return a.client.postAction(ctx, path, body, idempotent)
}

// UserRef selects a user by numeric ID or by username for account actions. Build one with
// [UserByID] or [UserByUsername]. When both fields are set, ID is used.
type UserRef struct {
	// ID is the numeric user ID.
	ID string
	// Username is the handle, with or without a leading "@". The API resolves it to the ID.
	Username string
}

// UserByID returns a [UserRef] for a numeric user ID.
func UserByID(id string) UserRef { return UserRef{ID: id} }

// UserByUsername returns a [UserRef] for a handle. A leading "@" is fine.
func UserByUsername(username string) UserRef { return UserRef{Username: username} }

// addTo adds user_id or username to f.
func (u UserRef) addTo(f fields) error {
	if strings.TrimSpace(u.ID) != "" {
		id, err := numericID("user ID", u.ID)
		if err != nil {
			return err
		}
		f["user_id"] = id
		return nil
	}
	if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(u.Username), "@")) != "" {
		name, err := cleanUsername(u.Username)
		if err != nil {
			return err
		}
		f["username"] = name
		return nil
	}
	return invalidParams("provide a user ID or username (UserByID or UserByUsername)")
}

// idAction sends {key: id} after checking that id is numeric.
func (a *AccountService) idAction(ctx context.Context, path, key, label, id string, idempotent bool) (json.RawMessage, error) {
	v, err := numericID(label, id)
	if err != nil {
		return nil, err
	}
	return a.send(ctx, path, fields{key: v}, idempotent)
}

// userAction sends user_id or username.
func (a *AccountService) userAction(ctx context.Context, path string, user UserRef) (json.RawMessage, error) {
	f := fields{}
	if err := user.addTo(f); err != nil {
		return nil, err
	}
	return a.send(ctx, path, f, true)
}

// AccountPageParams selects a page for paginated account reads such as
// [AccountService.Bookmarks].
type AccountPageParams struct {
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p AccountPageParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p AccountPageParams) WithCursor(cursor string) AccountPageParams {
	p.Cursor = cursor
	return p
}

// AccountCursorParams selects a page for account reads that take only a cursor, such as
// [AccountService.BookmarkFolders] and [AccountService.CommunityMemberships].
type AccountCursorParams struct {
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p AccountCursorParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p AccountCursorParams) WithCursor(cursor string) AccountCursorParams {
	p.Cursor = cursor
	return p
}

// cursorFields returns fields with cursor, if set.
func cursorFields(cursor string) fields {
	f := fields{}
	if cursor != "" {
		f["cursor"] = cursor
	}
	return f
}

// addPage validates count and adds count and cursor to f.
func addPage(f fields, count int, cursor string) error {
	if err := validCount(count); err != nil {
		return err
	}
	if count != 0 {
		f["count"] = count
	}
	if cursor != "" {
		f["cursor"] = cursor
	}
	return nil
}
