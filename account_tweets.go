package twittrapi

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxMediaPerTweet is the largest number of media IDs a tweet, reply or draft accepts.
const MaxMediaPerTweet = 4

const (
	pollMinChoices  = 2
	pollMaxChoices  = 4
	pollChoiceMax   = 25
	pollMinMinutes  = 5
	pollMaxMinutes  = 7 * 24 * 60
	pollTextMax     = 280
	listNameMax     = 25
	listDescMax     = 100
	draftTextMax    = 25000
	profileNameMax  = 50
	profileDescMax  = 160
	profileLocMax   = 30
	profileURLMax   = 100
	maxVideoLength  = 4 * time.Hour
	dmInboxMaxCount = 50
)

// ReplyControl sets who can reply to a tweet posted with [AccountService.CreateTweet].
type ReplyControl string

const (
	// ReplyFollows lets only accounts you follow reply.
	ReplyFollows ReplyControl = "only_follows"
	// ReplyVerified lets only verified accounts reply.
	ReplyVerified ReplyControl = "only_verified"
	// ReplyMentioned lets only accounts mentioned in the tweet reply.
	ReplyMentioned ReplyControl = "only_mentioned"
)

var replyControls = []string{string(ReplyFollows), string(ReplyVerified), string(ReplyMentioned)}

// Poll is a text poll attached to a tweet in [CreateTweetParams].
type Poll struct {
	// Choices holds 2 to 4 answers of up to 25 characters each.
	Choices []string
	// DurationMinutes is how long the poll runs, from 5 minutes to 7 days (10080). 0 uses
	// X's default of one day.
	DurationMinutes int
}

// CreateTweetParams describes a tweet for [AccountService.CreateTweet].
type CreateTweetParams struct {
	// Text is the tweet text. Over 280 characters it is posted as a long post (note tweet).
	// Required unless MediaIDs is set.
	Text string
	// MediaIDs are up to 4 media IDs from [AccountService.UploadImage] or
	// [AccountService.UploadVideo].
	MediaIDs []string
	// PossiblySensitive marks the media as sensitive.
	PossiblySensitive bool
	// ReplyControl limits who can reply. Empty lets everyone reply. Not used for community
	// tweets.
	ReplyControl ReplyControl
	// QuoteURL is the URL of a tweet to quote.
	QuoteURL string
	// CommunityID posts the tweet into this community.
	CommunityID string
	// NoteTweet forces a long post even for short text.
	NoteTweet bool
	// Poll attaches a text poll. It cannot be combined with media, a quote or a community,
	// and the text is limited to 280 characters.
	Poll *Poll
}

// ScheduleTweetParams describes a scheduled tweet for [AccountService.ScheduleTweet].
type ScheduleTweetParams struct {
	// Text is the tweet text. Required unless MediaIDs is set.
	Text string
	// MediaIDs are up to 4 media IDs from [AccountService.UploadImage] or
	// [AccountService.UploadVideo].
	MediaIDs []string
	// At is when X posts the tweet. Required. It is sent as a Unix time in seconds.
	At time.Time
}

// ScheduledTweetsParams sets the order for [AccountService.ScheduledTweets].
type ScheduledTweetsParams struct {
	// Descending lists the latest scheduled time first. The default is soonest first.
	Descending bool
}

// ReplyParams describes a reply for [AccountService.Reply].
type ReplyParams struct {
	// TweetID is the numeric ID of the tweet to reply to. Required.
	TweetID string
	// Text is the reply text. Required unless MediaIDs is set.
	Text string
	// MediaIDs are up to 4 media IDs from [AccountService.UploadImage] or
	// [AccountService.UploadVideo].
	MediaIDs []string
	// QuoteURL is the URL of a tweet to quote in the reply (attachment_url).
	QuoteURL string
	// PossiblySensitive marks the media as sensitive.
	PossiblySensitive bool
}

// ReportReason is the reason sent with [AccountService.Report].
type ReportReason string

// Report reasons accepted by [AccountService.Report].
const (
	ReportSpam            ReportReason = "SpamSimpleOption"
	ReportHateOrAbuse     ReportReason = "HateOrAbuseSimpleOption"
	ReportChildSafety     ReportReason = "ChildSafetySimpleOption"
	ReportViolentSpeech   ReportReason = "ViolentSpeechSimpleOption"
	ReportViolentMedia    ReportReason = "ViolentMediaSimpleOption"
	ReportIllegalBehavior ReportReason = "IRBSimpleOption" // illegal and regulated behaviors
	ReportImpersonation   ReportReason = "ImpersonationSimpleOption"
	ReportAdultContent    ReportReason = "AdultContentSimpleOption"
	ReportPrivateContent  ReportReason = "PrivateContentSimpleOption"
	ReportSuicideSelfHarm ReportReason = "SuicideSelfHarmSimpleOption"
	ReportTerrorism       ReportReason = "TerrorismSimpleOption"
	ReportCivicIntegrity  ReportReason = "CivicIntegritySimpleOption"
)

var reportReasons = []string{
	string(ReportSpam), string(ReportHateOrAbuse), string(ReportChildSafety), string(ReportViolentSpeech),
	string(ReportViolentMedia), string(ReportIllegalBehavior), string(ReportImpersonation), string(ReportAdultContent),
	string(ReportPrivateContent), string(ReportSuicideSelfHarm), string(ReportTerrorism), string(ReportCivicIntegrity),
}

// ReportParams describes a report for [AccountService.Report].
type ReportParams struct {
	// TweetID is the numeric ID of the tweet to report. Required.
	TweetID string
	// UserID is the numeric ID of the tweet's author. Required.
	UserID string
	// Reason is one of the Report* constants. Required.
	Reason ReportReason
}

// HomeTimelineParams configures [AccountService.HomeTimeline].
type HomeTimelineParams struct {
	// Count is the number of tweets to ask X for. 0 uses the API default (20).
	Count int
	// ExcludePromoted leaves out promoted tweets (ads).
	ExcludePromoted bool
}

// LikesParams selects a page for [AccountService.Likes].
type LikesParams struct {
	// UserID is your numeric user ID. Needed only when the cookie has no twid; X shows likes
	// only to the account itself.
	UserID string
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p LikesParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p LikesParams) WithCursor(cursor string) LikesParams {
	p.Cursor = cursor
	return p
}

// FollowersYouKnowParams selects a user and a page for [AccountService.FollowersYouKnow].
type FollowersYouKnowParams struct {
	// User is the account whose followers to check. Required.
	User UserRef
	// Count is the page size, from 1 to 100. 0 uses the API default (20).
	Count int
	// Cursor requests a specific page. Leave empty for the first page.
	Cursor string
}

// PageCursor returns p.Cursor. It implements [Pageable].
func (p FollowersYouKnowParams) PageCursor() string { return p.Cursor }

// WithCursor returns a copy of p with Cursor set. It implements [Pageable].
func (p FollowersYouKnowParams) WithCursor(cursor string) FollowersYouKnowParams {
	p.Cursor = cursor
	return p
}

// mediaIDs checks up to MaxMediaPerTweet numeric IDs.
func mediaIDs(ids []string) ([]string, error) {
	if len(ids) > MaxMediaPerTweet {
		return nil, invalidParams("MediaIDs accepts at most %d IDs, got %d", MaxMediaPerTweet, len(ids))
	}
	out := make([]string, len(ids))
	for i, raw := range ids {
		id, err := numericID("each media ID", raw)
		if err != nil {
			return nil, err
		}
		out[i] = id
	}
	return out, nil
}

// addTextAndMedia adds text (always) and media_ids (when set). Text or media is required.
func addTextAndMedia(f fields, text string, ids []string) error {
	media, err := mediaIDs(ids)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" && len(media) == 0 {
		return invalidParams("Text must be non-empty unless MediaIDs is set")
	}
	f["text"] = text
	if len(media) > 0 {
		f["media_ids"] = media
	}
	return nil
}

func validatePoll(p *Poll, t CreateTweetParams) (fields, error) {
	if len(p.Choices) < pollMinChoices || len(p.Choices) > pollMaxChoices {
		return nil, invalidParams("Poll needs %d to %d choices, got %d", pollMinChoices, pollMaxChoices, len(p.Choices))
	}
	for i, c := range p.Choices {
		if strings.TrimSpace(c) == "" {
			return nil, invalidParams("Poll choice %d is empty", i+1)
		}
		if utf8.RuneCountInString(c) > pollChoiceMax {
			return nil, invalidParams("Poll choice %d is longer than %d characters", i+1, pollChoiceMax)
		}
	}
	if p.DurationMinutes != 0 && (p.DurationMinutes < pollMinMinutes || p.DurationMinutes > pollMaxMinutes) {
		return nil, invalidParams("Poll.DurationMinutes must be 0 or from %d to %d, got %d", pollMinMinutes, pollMaxMinutes, p.DurationMinutes)
	}
	switch {
	case len(t.MediaIDs) > 0:
		return nil, invalidParams("Poll cannot be combined with MediaIDs")
	case t.QuoteURL != "":
		return nil, invalidParams("Poll cannot be combined with QuoteURL")
	case t.CommunityID != "":
		return nil, invalidParams("Poll cannot be combined with CommunityID")
	case t.NoteTweet || utf8.RuneCountInString(t.Text) > pollTextMax:
		return nil, invalidParams("a tweet with a poll is limited to %d characters", pollTextMax)
	}
	poll := fields{"choices": p.Choices}
	if p.DurationMinutes != 0 {
		poll["duration_minutes"] = p.DurationMinutes
	}
	return poll, nil
}

// CreateTweet posts a tweet: a plain tweet, a quote tweet (QuoteURL), a long post (over 280
// characters or NoteTweet), a community tweet (CommunityID) or a poll (Poll). To schedule a
// tweet use [AccountService.ScheduleTweet]. POST /client/create. Not retried after a
// timeout or 5xx.
func (a *AccountService) CreateTweet(ctx context.Context, p CreateTweetParams) (json.RawMessage, error) {
	f := fields{}
	if err := addTextAndMedia(f, p.Text, p.MediaIDs); err != nil {
		return nil, err
	}
	mode, err := oneOf("ReplyControl", string(p.ReplyControl), replyControls)
	if err != nil {
		return nil, err
	}
	if mode != "" {
		f["conversation_control_mode"] = mode
	}
	if p.CommunityID != "" {
		id, err := numericID("CommunityID", p.CommunityID)
		if err != nil {
			return nil, err
		}
		f["community_id"] = id
	}
	if p.Poll != nil {
		poll, err := validatePoll(p.Poll, p)
		if err != nil {
			return nil, err
		}
		f["poll"] = poll
	}
	if p.PossiblySensitive {
		f["possibly_sensitive"] = true
	}
	if q := strings.TrimSpace(p.QuoteURL); q != "" {
		f["quote_url"] = q
	}
	if p.NoteTweet {
		f["is_note_tweet"] = true
	}
	return a.send(ctx, "/client/create", f, false)
}

// ScheduleTweet schedules a tweet with text and media for p.At. POST /client/create with
// schedule_time. Not retried after a timeout or 5xx.
func (a *AccountService) ScheduleTweet(ctx context.Context, p ScheduleTweetParams) (json.RawMessage, error) {
	f := fields{}
	if err := addTextAndMedia(f, p.Text, p.MediaIDs); err != nil {
		return nil, err
	}
	if p.At.IsZero() {
		return nil, invalidParams("At must be set")
	}
	f["schedule_time"] = p.At.Unix()
	return a.send(ctx, "/client/create", f, false)
}

// ScheduledTweets lists the account's scheduled tweets. POST /client/schedule-overview.
func (a *AccountService) ScheduledTweets(ctx context.Context, p ScheduledTweetsParams) (json.RawMessage, error) {
	f := fields{}
	if p.Descending {
		f["ascending"] = false
	}
	return a.send(ctx, "/client/schedule-overview", f, true)
}

// DeleteScheduledTweet deletes a scheduled tweet. POST /client/schedule-delete.
func (a *AccountService) DeleteScheduledTweet(ctx context.Context, scheduledTweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/schedule-delete", "scheduled_tweet_id", "scheduledTweetID", scheduledTweetID, true)
}

// Reply replies to a tweet. POST /client/reply. Not retried after a timeout or 5xx.
func (a *AccountService) Reply(ctx context.Context, p ReplyParams) (json.RawMessage, error) {
	id, err := numericID("TweetID", p.TweetID)
	if err != nil {
		return nil, err
	}
	f := fields{"id": id}
	if err := addTextAndMedia(f, p.Text, p.MediaIDs); err != nil {
		return nil, err
	}
	if q := strings.TrimSpace(p.QuoteURL); q != "" {
		f["attachment_url"] = q
	}
	if p.PossiblySensitive {
		f["possibly_sensitive"] = true
	}
	return a.send(ctx, "/client/reply", f, false)
}

// DeleteTweet deletes one of the account's tweets. POST /client/deleteTweet.
func (a *AccountService) DeleteTweet(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/deleteTweet", "tweet_id", "tweetID", tweetID, true)
}

// Like likes a tweet. POST /client/like.
func (a *AccountService) Like(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/like", "tweet_id", "tweetID", tweetID, true)
}

// Unlike removes a like. POST /client/unlike.
func (a *AccountService) Unlike(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/unlike", "tweet_id", "tweetID", tweetID, true)
}

// Retweet retweets a tweet. POST /client/retweet.
func (a *AccountService) Retweet(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/retweet", "tweet_id", "tweetID", tweetID, true)
}

// Unretweet removes a retweet. tweetID is the original tweet's ID. POST /client/unretweet.
func (a *AccountService) Unretweet(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/unretweet", "tweet_id", "tweetID", tweetID, true)
}

// Pin pins one of the account's tweets to its profile. POST /client/pin.
func (a *AccountService) Pin(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/pin", "tweet_id", "tweetID", tweetID, true)
}

// Unpin unpins a tweet from the account's profile. POST /client/unpin.
func (a *AccountService) Unpin(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/unpin", "tweet_id", "tweetID", tweetID, true)
}

// Bookmark adds a tweet to the account's bookmarks. POST /client/bookmark.
func (a *AccountService) Bookmark(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/bookmark", "tweet_id", "tweetID", tweetID, true)
}

// Unbookmark removes a tweet from the account's bookmarks. POST /client/unbookmark.
func (a *AccountService) Unbookmark(ctx context.Context, tweetID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/unbookmark", "tweet_id", "tweetID", tweetID, true)
}

// Report reports a tweet and its author to X. POST /client/report. Not retried after a
// timeout or 5xx.
func (a *AccountService) Report(ctx context.Context, p ReportParams) (json.RawMessage, error) {
	tweetID, err := numericID("TweetID", p.TweetID)
	if err != nil {
		return nil, err
	}
	userID, err := numericID("UserID", p.UserID)
	if err != nil {
		return nil, err
	}
	if p.Reason == "" {
		return nil, invalidParams("Reason must be set")
	}
	reason, err := oneOf("Reason", string(p.Reason), reportReasons)
	if err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/report", fields{"tweet_id": tweetID, "user_id": userID, "reason": reason}, false)
}

// Follow follows a user. POST /client/follow.
func (a *AccountService) Follow(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/follow", user)
}

// Unfollow unfollows a user. POST /client/unfollow.
func (a *AccountService) Unfollow(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/unfollow", user)
}

// Mute mutes a user. POST /client/mute.
func (a *AccountService) Mute(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/mute", user)
}

// Unmute unmutes a user. POST /client/unmute.
func (a *AccountService) Unmute(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/unmute", user)
}

// Block blocks a user. POST /client/block.
func (a *AccountService) Block(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/block", user)
}

// Unblock unblocks a user. POST /client/unblock.
func (a *AccountService) Unblock(ctx context.Context, user UserRef) (json.RawMessage, error) {
	return a.userAction(ctx, "/client/unblock", user)
}

// HomeTimeline returns the account's home timeline (For you). POST /client/home-timeline.
func (a *AccountService) HomeTimeline(ctx context.Context, p HomeTimelineParams) (json.RawMessage, error) {
	if p.Count < 0 {
		return nil, invalidParams("Count must not be negative, got %d", p.Count)
	}
	f := fields{}
	if p.Count != 0 {
		f["count"] = p.Count
	}
	if p.ExcludePromoted {
		f["include_promoted_content"] = false
	}
	return a.send(ctx, "/client/home-timeline", f, true)
}

// Bookmarks returns the account's bookmarks. Paginated. POST /client/bookmarks.
func (a *AccountService) Bookmarks(ctx context.Context, p AccountPageParams) (json.RawMessage, error) {
	return a.page(ctx, "/client/bookmarks", p)
}

// Blocked returns the accounts this account blocks. Paginated. POST /client/blocked.
func (a *AccountService) Blocked(ctx context.Context, p AccountPageParams) (json.RawMessage, error) {
	return a.page(ctx, "/client/blocked", p)
}

// Muted returns the accounts this account mutes. Paginated. POST /client/muted.
func (a *AccountService) Muted(ctx context.Context, p AccountPageParams) (json.RawMessage, error) {
	return a.page(ctx, "/client/muted", p)
}

func (a *AccountService) page(ctx context.Context, path string, p AccountPageParams) (json.RawMessage, error) {
	f := fields{}
	if err := addPage(f, p.Count, p.Cursor); err != nil {
		return nil, err
	}
	return a.send(ctx, path, f, true)
}

// Likes returns the tweets this account liked. X shows likes only to the account itself.
// Paginated. POST /client/likes.
func (a *AccountService) Likes(ctx context.Context, p LikesParams) (json.RawMessage, error) {
	f := fields{}
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
	return a.send(ctx, "/client/likes", f, true)
}

// FollowersYouKnow returns followers of a user that this account follows. Paginated.
// POST /client/followers-you-know.
func (a *AccountService) FollowersYouKnow(ctx context.Context, p FollowersYouKnowParams) (json.RawMessage, error) {
	f := fields{}
	if err := p.User.addTo(f); err != nil {
		return nil, err
	}
	if err := addPage(f, p.Count, p.Cursor); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/followers-you-know", f, true)
}
