package twittrapi

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

// Articles are X's long-form posts. Every article call needs X Premium: without it X
// refuses with error code 37 ("Twitter Blue claim is not enabled").

const articleTweetTextMax = 256

// ArticleLifecycle selects drafts or published articles in [ArticlesParams].
type ArticleLifecycle string

const (
	// ArticleDrafts lists draft articles. This is the API default.
	ArticleDrafts ArticleLifecycle = "draft"
	// ArticlePublished lists published articles.
	ArticlePublished ArticleLifecycle = "published"
)

var articleLifecycles = []string{string(ArticleDrafts), string(ArticlePublished)}

// ArticleVisibility sets who can read an article published with
// [AccountService.PublishArticle].
type ArticleVisibility string

const (
	// ArticlePublic makes the article visible to everyone. This is the API default.
	ArticlePublic ArticleVisibility = "public"
	// ArticleSubscribers makes the article visible to the account's subscribers only.
	ArticleSubscribers ArticleVisibility = "subscribers"
)

var articleVisibilities = []string{string(ArticlePublic), string(ArticleSubscribers)}

// ArticleReplyControl sets who can reply to the post of an article published with
// [AccountService.PublishArticle].
type ArticleReplyControl string

const (
	// ArticleRepliesAll lets everyone reply. This is the API default.
	ArticleRepliesAll ArticleReplyControl = "all"
	// ArticleRepliesCommunity lets only the communities you are in reply.
	ArticleRepliesCommunity ArticleReplyControl = "community"
	// ArticleRepliesVerified lets only verified accounts reply.
	ArticleRepliesVerified ArticleReplyControl = "verified"
	// ArticleRepliesByInvitation lets only accounts you mention reply.
	ArticleRepliesByInvitation ArticleReplyControl = "byinvitation"
)

var articleReplyControls = []string{
	string(ArticleRepliesAll), string(ArticleRepliesCommunity), string(ArticleRepliesVerified), string(ArticleRepliesByInvitation),
}

// ArticleCoverCategory is the kind of media used as an article cover in
// [UpdateArticleParams].
type ArticleCoverCategory string

const (
	// ArticleCoverImage is an image from [AccountService.UploadImage]. This is the API
	// default.
	ArticleCoverImage ArticleCoverCategory = "DraftTweetImage"
	// ArticleCoverGIF is a GIF from [AccountService.UploadImage].
	ArticleCoverGIF ArticleCoverCategory = "DraftTweetGif"
	// ArticleCoverVideo is a video from [AccountService.UploadVideo].
	ArticleCoverVideo ArticleCoverCategory = "AmplifyVideo"
)

var articleCoverCategories = []string{string(ArticleCoverImage), string(ArticleCoverGIF), string(ArticleCoverVideo)}

// ArticlesParams selects articles and a page for [AccountService.Articles].
type ArticlesParams struct {
	// Lifecycle selects drafts or published articles. Empty uses the API default
	// ([ArticleDrafts]).
	Lifecycle ArticleLifecycle
	// UserID is your numeric user ID. Needed only when the cookie has no twid.
	UserID string
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p ArticlesParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p ArticlesParams) WithCursor(cursor string) ArticlesParams {
	p.Cursor = cursor
	return p
}

// CreateArticleParams is the content of a new draft article for
// [AccountService.CreateArticle]. All fields are optional; without them the draft starts
// empty. Set Text or ContentState, not both.
type CreateArticleParams struct {
	// Title is the article title.
	Title string
	// Text is the body as plain text, one paragraph per line.
	Text string
	// ContentState is the body in X's format, {"blocks": [...], "entity_map": [...]}, as
	// [AccountService.Article] returns it. Use it for formatting that plain text cannot
	// express.
	ContentState json.RawMessage
}

// UpdateArticleParams describes changes for [AccountService.UpdateArticle]. Unset fields
// stay as they are; set at least one of Title, Text, ContentState, CoverMediaID or
// RemoveCover. [Ptr] helps fill the pointer fields.
type UpdateArticleParams struct {
	// ArticleID is the numeric article ID. Required.
	ArticleID string
	// Title is the new title.
	Title *string
	// Text replaces the whole body with plain text, one paragraph per line. It cannot be
	// combined with ContentState.
	Text *string
	// ContentState replaces the whole body with content in X's format, {"blocks": [...],
	// "entity_map": [...]}. It cannot be combined with Text.
	ContentState json.RawMessage
	// CoverMediaID is a numeric media ID from [AccountService.UploadImage] or
	// [AccountService.UploadVideo] to use as the cover.
	CoverMediaID string
	// CoverMediaCategory is the kind of cover media. Empty uses the API default
	// ([ArticleCoverImage]). Only used with CoverMediaID.
	CoverMediaCategory ArticleCoverCategory
	// RemoveCover removes the cover. It cannot be combined with CoverMediaID.
	RemoveCover bool
}

// PublishArticleParams describes how [AccountService.PublishArticle] publishes a draft.
type PublishArticleParams struct {
	// ArticleID is the numeric article ID. Required.
	ArticleID string
	// Visibility sets who can read the article. Empty uses the API default
	// ([ArticlePublic]).
	Visibility ArticleVisibility
	// ReplyControl sets who can reply to the article's post. Empty uses the API default
	// ([ArticleRepliesAll]). Ignored for [ArticleSubscribers].
	ReplyControl ArticleReplyControl
	// TweetText is the text of the post that shares the article, up to 256 characters.
	TweetText string
}

// choice checks an optional value against allowed, ignoring case, and returns the allowed
// spelling.
func choice(name, value string, allowed []string) (string, error) {
	v := strings.TrimSpace(value)
	for _, a := range allowed {
		if strings.EqualFold(v, a) {
			return a, nil
		}
	}
	return oneOf(name, v, allowed)
}

// contentState checks that raw is an object with a "blocks" array and an "entity_map"
// array.
func contentState(raw json.RawMessage) (json.RawMessage, error) {
	var cs struct {
		Blocks    json.RawMessage `json:"blocks"`
		EntityMap json.RawMessage `json:"entity_map"`
	}
	isArray := func(v json.RawMessage) bool {
		t := bytes.TrimSpace(v)
		return len(t) > 0 && t[0] == '['
	}
	t := bytes.TrimSpace(raw)
	if len(t) == 0 || t[0] != '{' || json.Unmarshal(t, &cs) != nil || !isArray(cs.Blocks) || !isArray(cs.EntityMap) {
		return nil, invalidParams(`ContentState must be a JSON object {"blocks": [...], "entity_map": [...]}`)
	}
	return json.RawMessage(t), nil
}

// articleContent adds text or content_state to f.
func articleContent(f fields, text *string, state json.RawMessage) error {
	hasState := len(bytes.TrimSpace(state)) > 0
	if text != nil && hasState {
		return invalidParams("set Text or ContentState, not both")
	}
	if hasState {
		cs, err := contentState(state)
		if err != nil {
			return err
		}
		f["content_state"] = cs
	}
	if text != nil {
		f["text"] = *text
	}
	return nil
}

// Articles lists the account's own articles, drafts by default. Paginated.
// POST /client/articles.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) Articles(ctx context.Context, p ArticlesParams) (json.RawMessage, error) {
	f := fields{}
	lifecycle, err := choice("Lifecycle", string(p.Lifecycle), articleLifecycles)
	if err != nil {
		return nil, err
	}
	if lifecycle != "" {
		f["lifecycle"] = lifecycle
	}
	if strings.TrimSpace(p.UserID) != "" {
		id, err := numericID("UserID", p.UserID)
		if err != nil {
			return nil, err
		}
		f["user_id"] = id
	}
	if err := addPage(f, p.Count, p.Cursor); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/articles", f, true)
}

// Article returns one of the account's articles: title, content_state, cover, lifecycle
// and metadata. POST /client/articles/get.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) Article(ctx context.Context, articleID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/articles/get", "article_id", "articleID", articleID, true)
}

// CreateArticle creates a draft article. The answer holds the new draft and its ID.
// POST /client/articles/create. Not retried after a timeout or 5xx.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) CreateArticle(ctx context.Context, p CreateArticleParams) (json.RawMessage, error) {
	f := fields{}
	if p.Title != "" {
		f["title"] = p.Title
	}
	var text *string
	if p.Text != "" {
		text = &p.Text
	}
	if err := articleContent(f, text, p.ContentState); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/articles/create", f, false)
}

// UpdateArticle changes the title, the body and/or the cover of an article. Text or
// ContentState replaces the whole body. The changes are saved one after another and billed
// as one call. POST /client/articles/update. Not retried after a timeout or 5xx.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) UpdateArticle(ctx context.Context, p UpdateArticleParams) (json.RawMessage, error) {
	id, err := numericID("ArticleID", p.ArticleID)
	if err != nil {
		return nil, err
	}
	f := fields{"article_id": id}
	if p.Title != nil {
		f["title"] = *p.Title
	}
	if err := articleContent(f, p.Text, p.ContentState); err != nil {
		return nil, err
	}
	cover := strings.TrimSpace(p.CoverMediaID)
	switch {
	case cover != "" && p.RemoveCover:
		return nil, invalidParams("set CoverMediaID or RemoveCover, not both")
	case cover != "":
		mediaID, err := numericID("CoverMediaID", cover)
		if err != nil {
			return nil, err
		}
		f["cover_media_id"] = mediaID
		kind, err := choice("CoverMediaCategory", string(p.CoverMediaCategory), articleCoverCategories)
		if err != nil {
			return nil, err
		}
		if kind != "" {
			f["cover_media_category"] = kind
		}
	case p.CoverMediaCategory != "":
		return nil, invalidParams("CoverMediaCategory needs CoverMediaID")
	case p.RemoveCover:
		f["remove_cover"] = true
	}
	if len(f) == 1 {
		return nil, invalidParams("set at least one of Title, Text, ContentState, CoverMediaID or RemoveCover")
	}
	return a.send(ctx, "/client/articles/update", f, false)
}

// PublishArticle publishes a draft article as a post. POST /client/articles/publish. Not
// retried after a timeout or 5xx.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) PublishArticle(ctx context.Context, p PublishArticleParams) (json.RawMessage, error) {
	id, err := numericID("ArticleID", p.ArticleID)
	if err != nil {
		return nil, err
	}
	f := fields{"article_id": id}
	visibility, err := choice("Visibility", string(p.Visibility), articleVisibilities)
	if err != nil {
		return nil, err
	}
	if visibility != "" {
		f["visibility"] = visibility
	}
	mode, err := choice("ReplyControl", string(p.ReplyControl), articleReplyControls)
	if err != nil {
		return nil, err
	}
	if mode != "" {
		f["reply_control"] = mode
	}
	if err := maxText("TweetText", &p.TweetText, articleTweetTextMax); err != nil {
		return nil, err
	}
	if t := strings.TrimSpace(p.TweetText); t != "" {
		f["tweet_text"] = t
	}
	return a.send(ctx, "/client/articles/publish", f, false)
}

// UnpublishArticle turns a published article back into a draft.
// POST /client/articles/unpublish. Not retried after a timeout or 5xx.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) UnpublishArticle(ctx context.Context, articleID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/articles/unpublish", "article_id", "articleID", articleID, false)
}

// DeleteArticle deletes an article. POST /client/articles/delete. Not retried after a
// timeout or 5xx.
//
// Needs X Premium: without it X refuses with error code 37 ("Twitter Blue claim is not
// enabled").
func (a *AccountService) DeleteArticle(ctx context.Context, articleID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/articles/delete", "article_id", "articleID", articleID, false)
}
