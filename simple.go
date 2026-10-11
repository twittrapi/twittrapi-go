package twittrapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Types of the opt-in simple response format (format=simple). Field names follow the API's
// documented schema (https://twittrapi.com/docs/simple-format): every field is always
// present, unknown values are null (nil pointers here) and lists are [] rather than null.
// New fields may be added to the API; they are ignored until the SDK adds them.

// SimplePage is the envelope of a list or timeline response in the simple format.
type SimplePage[T any] struct {
	// Data holds the items of this page.
	Data []T `json:"data"`
	// NextCursor is the cursor of the next page, or nil when there is nothing more to fetch.
	// Pass it as Cursor to get the next page.
	NextCursor *string `json:"nextCursor"`
	// PreviousCursor is X's top cursor, or nil. Use it to poll for newer items.
	PreviousCursor *string `json:"previousCursor"`
	// HasNextPage is false on the last page.
	HasNextPage bool `json:"hasNextPage"`
}

// Next returns the cursor of the next page, or "" when HasNextPage is false or there is no
// cursor.
func (p *SimplePage[T]) Next() string {
	if p == nil || !p.HasNextPage || p.NextCursor == nil {
		return ""
	}
	return *p.NextCursor
}

// SimpleTweet is a post in the simple format.
//
// A tweet X did not return (deleted, withheld, protected or a tombstone) whose ID is known
// is sent as {"id": "...", "unavailable": true}: then only ID and Unavailable are set. The
// same placeholder is used for Quoted and RetweetOf.
type SimpleTweet struct {
	ID             string            `json:"id"`
	Unavailable    bool              `json:"unavailable,omitempty"`
	URL            string            `json:"url"`
	Text           string            `json:"text"`
	CreatedAt      *string           `json:"createdAt"`
	Lang           *string           `json:"lang"`
	Author         *SimpleUser       `json:"author"`
	Counts         SimpleTweetCounts `json:"counts"`
	Media          []SimpleMedia     `json:"media"`
	URLs           []string          `json:"urls"`
	Hashtags       []string          `json:"hashtags"`
	Mentions       []SimpleUserRef   `json:"mentions"`
	ReplyTo        *SimpleReplyTo    `json:"replyTo"`
	Quoted         *SimpleTweet      `json:"quoted"`
	RetweetOf      *SimpleTweet      `json:"retweetOf"`
	ConversationID *string           `json:"conversationId"`
	IsPinned       bool              `json:"isPinned"`
	Source         *string           `json:"source"`
	Article        *SimpleArticle    `json:"article"`
}

// SimpleTweetCounts are the public metrics of a tweet. Views is nil when X hides the count.
type SimpleTweetCounts struct {
	Likes     int64  `json:"likes"`
	Retweets  int64  `json:"retweets"`
	Replies   int64  `json:"replies"`
	Quotes    int64  `json:"quotes"`
	Bookmarks int64  `json:"bookmarks"`
	Views     *int64 `json:"views"`
}

// SimpleMedia is a photo, video or GIF attached to a tweet. For videos and GIFs, URL is the
// highest-bitrate MP4 and PreviewURL the poster image.
type SimpleMedia struct {
	Type       string  `json:"type"` // photo, video or gif
	URL        string  `json:"url"`
	PreviewURL string  `json:"previewUrl"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	AltText    *string `json:"altText"`
	DurationMs *int64  `json:"durationMs"`
}

// SimpleReplyTo names the tweet (and its author) a tweet answers.
type SimpleReplyTo struct {
	TweetID  *string `json:"tweetId"`
	UserID   *string `json:"userId"`
	Username *string `json:"username"`
}

// SimpleUserRef is a short reference to an account.
type SimpleUserRef struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// SimpleArticle is the long-form X Article attached to a tweet.
type SimpleArticle struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	PreviewText   string  `json:"previewText"`
	CoverImageURL *string `json:"coverImageUrl"`
	PublishedAt   *string `json:"publishedAt"`
	Text          *string `json:"text"`
}

// SimpleUser is an account in the simple format.
type SimpleUser struct {
	ID             string           `json:"id"`
	Username       string           `json:"username"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	Location       string           `json:"location"`
	URL            *string          `json:"url"`
	CreatedAt      *string          `json:"createdAt"`
	AvatarURL      *string          `json:"avatarUrl"`
	BannerURL      *string          `json:"bannerUrl"`
	Verified       bool             `json:"verified"`
	VerifiedType   *string          `json:"verifiedType"`
	Protected      bool             `json:"protected"`
	Counts         SimpleUserCounts `json:"counts"`
	PinnedTweetIDs []string         `json:"pinnedTweetIds"`
	IsAutomated    bool             `json:"isAutomated"`
}

// SimpleUserCounts are the profile counters.
type SimpleUserCounts struct {
	Followers int64 `json:"followers"`
	Following int64 `json:"following"`
	Tweets    int64 `json:"tweets"`
	Likes     int64 `json:"likes"`
	Media     int64 `json:"media"`
	Listed    int64 `json:"listed"`
}

// SimpleUserAbout is GET /user/about/{username} in the simple format: the account plus the
// "About this account" details.
type SimpleUserAbout struct {
	SimpleUser
	About SimpleAbout `json:"about"`
}

// SimpleAbout holds the "About this account" details.
type SimpleAbout struct {
	AccountBasedIn         *string `json:"accountBasedIn"`
	LocationAccurate       *bool   `json:"locationAccurate"`
	CreatedCountryAccurate *bool   `json:"createdCountryAccurate"`
	Source                 *string `json:"source"`
	UsernameChanges        int64   `json:"usernameChanges"`
	LastUsernameChangeAt   *string `json:"lastUsernameChangeAt"`
	AffiliateUsername      *string `json:"affiliateUsername"`
	IdentityVerified       bool    `json:"identityVerified"`
	VerifiedSince          *string `json:"verifiedSince"`
}

// SimpleList is an X List in the simple format.
type SimpleList struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	CreatedAt     *string        `json:"createdAt"`
	MemberCount   int64          `json:"memberCount"`
	FollowerCount int64          `json:"followerCount"`
	Private       bool           `json:"private"`
	BannerURL     *string        `json:"bannerUrl"`
	Owner         *SimpleUserRef `json:"owner"`
}

// SimpleCommunity is an X Community in the simple format.
type SimpleCommunity struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	CreatedAt      *string               `json:"createdAt"`
	MemberCount    int64                 `json:"memberCount"`
	ModeratorCount int64                 `json:"moderatorCount"`
	JoinPolicy     *string               `json:"joinPolicy"`
	NSFW           bool                  `json:"nsfw"`
	Topic          *string               `json:"topic"`
	BannerURL      *string               `json:"bannerUrl"`
	Creator        *SimpleUserRef        `json:"creator"`
	Rules          []SimpleCommunityRule `json:"rules"`
}

// SimpleCommunityRule is one rule of a community.
type SimpleCommunityRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SimpleTrend is one trending topic in the simple format.
type SimpleTrend struct {
	Name        string `json:"name"`
	Query       string `json:"query"`
	URL         string `json:"url"`
	TweetVolume *int64 `json:"tweetVolume"`
}

// SimpleTrendLocation is a place with trends (a WOEID) in the simple format.
type SimpleTrendLocation struct {
	Woeid       int64   `json:"woeid"`
	Name        string  `json:"name"`
	Country     string  `json:"country"`
	CountryCode *string `json:"countryCode"`
	PlaceType   string  `json:"placeType"`
	ParentID    int64   `json:"parentId"`
}

// ErrNotSimpleFormat is returned by the simple-format decoders when a response is not in the
// simple format, for example because the endpoint does not support it or the API returned
// its raw body.
var ErrNotSimpleFormat = errors.New("twittrapi: response is not in the simple format")

type simpleEnvelope struct {
	Data           json.RawMessage `json:"data"`
	NextCursor     *string         `json:"nextCursor"`
	PreviousCursor *string         `json:"previousCursor"`
	HasNextPage    *bool           `json:"hasNextPage"`
}

// simplePageEnvelope reports whether data is a simple-format list envelope and returns its
// next cursor ("" when hasNextPage is false).
func simplePageEnvelope(data []byte) (ok bool, next string) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' || !bytes.Contains(trimmed, []byte(`"hasNextPage"`)) {
		return false, ""
	}
	var env simpleEnvelope
	if err := json.Unmarshal(trimmed, &env); err != nil || env.HasNextPage == nil || env.Data == nil {
		return false, ""
	}
	if *env.HasNextPage && env.NextCursor != nil {
		next = *env.NextCursor
	}
	return true, next
}

// DecodeSimplePage decodes a simple-format list response ({"data": [...], "nextCursor",
// "previousCursor", "hasNextPage"}) into a [SimplePage]. It returns an error wrapping
// [ErrNotSimpleFormat] when data is not such an envelope.
func DecodeSimplePage[T any](data []byte) (*SimplePage[T], error) {
	var env simpleEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("twittrapi: decoding simple page: %w", err)
	}
	if env.HasNextPage == nil || env.Data == nil {
		return nil, ErrNotSimpleFormat
	}
	page := &SimplePage[T]{
		NextCursor:     env.NextCursor,
		PreviousCursor: env.PreviousCursor,
		HasNextPage:    *env.HasNextPage,
	}
	if err := json.Unmarshal(env.Data, &page.Data); err != nil {
		return nil, fmt.Errorf("twittrapi: decoding simple page data: %w", err)
	}
	if page.Data == nil {
		page.Data = []T{}
	}
	return page, nil
}

// DecodeSimple decodes a simple-format single object response ({"data": {...}}). It returns
// nil and no error when data is null (X returned nothing for the ID or handle), and an error
// wrapping [ErrNotSimpleFormat] when the body is not a {"data": ...} envelope or is a list
// envelope.
func DecodeSimple[T any](data []byte) (*T, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("twittrapi: decoding simple response: %w", err)
	}
	raw, ok := top["data"]
	if !ok || len(top) != 1 {
		return nil, ErrNotSimpleFormat
	}
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	v := new(T)
	if err := json.Unmarshal(raw, v); err != nil {
		return nil, fmt.Errorf("twittrapi: decoding simple response data: %w", err)
	}
	return v, nil
}

// FetchSimple calls fn (a service method such as client.Users.GetByUsername or
// client.Lists.Details) with the simple format and decodes the single object it returns.
// The result is nil when X returned nothing for the ID or handle.
//
//	user, err := twittrapi.FetchSimple[twittrapi.SimpleUser](ctx, client.Users.GetByUsername, "jack")
func FetchSimple[T any, A any](ctx context.Context, fn func(context.Context, A) (json.RawMessage, error), arg A) (*T, error) {
	data, err := fn(ContextWithFormat(ctx, FormatSimple), arg)
	if err != nil {
		return nil, err
	}
	return DecodeSimple[T](data)
}

// FetchSimplePage calls fn (a list or timeline method such as client.Users.Tweets or
// client.Search.Tweets) with the simple format and decodes the page it returns.
//
//	page, err := twittrapi.FetchSimplePage[twittrapi.SimpleTweet](ctx, client.Users.Tweets,
//		twittrapi.UserTimelineParams{Username: "XDevelopers"})
func FetchSimplePage[T any, A any](ctx context.Context, fn func(context.Context, A) (json.RawMessage, error), arg A) (*SimplePage[T], error) {
	data, err := fn(ContextWithFormat(ctx, FormatSimple), arg)
	if err != nil {
		return nil, err
	}
	return DecodeSimplePage[T](data)
}

// SimplePageIterator walks the pages of a paginated method in the simple format. Create one
// with [PaginateSimple]. Use it like [PageIterator]. It is not safe for concurrent use.
type SimplePageIterator[T any] struct {
	it   *PageIterator
	page *SimplePage[T]
	err  error
}

// PaginateSimple is [Paginate] for the simple format: it requests every page with
// format=simple, follows nextCursor while hasNextPage is true and decodes each page into a
// [SimplePage] of T. The [PaginateOptions] limits apply (10 pages by default).
//
//	it := twittrapi.PaginateSimple[twittrapi.SimpleUser](ctx, client.Users.Followers,
//		twittrapi.UserTimelineParams{Username: "jack", Count: 100}, nil)
//	for it.Next() {
//		for _, u := range it.Page().Data {
//			fmt.Println(u.Username)
//		}
//	}
//	if err := it.Err(); err != nil {
//		log.Fatal(err)
//	}
func PaginateSimple[T any, P Pageable[P]](ctx context.Context, fn PageFunc[P], params P, opts *PaginateOptions) *SimplePageIterator[T] {
	return &SimplePageIterator[T]{it: Paginate(ContextWithFormat(ctx, FormatSimple), fn, params, opts)}
}

// Next fetches and decodes the next page. It returns false when iteration is over; check
// [SimplePageIterator.Err].
func (s *SimplePageIterator[T]) Next() bool {
	if s.err != nil || !s.it.Next() {
		return false
	}
	page, err := DecodeSimplePage[T](s.it.Page().Data)
	if err != nil {
		s.err = err
		s.it.done = true
		return false
	}
	s.page = page
	return true
}

// Page returns the page decoded by the last successful call to [SimplePageIterator.Next].
func (s *SimplePageIterator[T]) Page() *SimplePage[T] { return s.page }

// Raw returns the raw page (JSON, page number and cursors) behind [SimplePageIterator.Page].
func (s *SimplePageIterator[T]) Raw() Page { return s.it.Page() }

// Err returns the error that stopped the iteration, or nil.
func (s *SimplePageIterator[T]) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.it.Err()
}

// ForEach calls fn for every remaining page. It stops at the first error from a request,
// from decoding or from fn and returns it.
func (s *SimplePageIterator[T]) ForEach(fn func(*SimplePage[T]) error) error {
	for s.Next() {
		if err := fn(s.Page()); err != nil {
			s.it.done = true
			return err
		}
	}
	return s.Err()
}
