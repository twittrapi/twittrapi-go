package twittrapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

// CreateListParams describes a list for [AccountService.CreateList].
type CreateListParams struct {
	// Name is the list name, up to 25 characters. Required.
	Name string
	// Description is up to 100 characters.
	Description string
	// Private makes the list visible only to you.
	Private bool
}

// UpdateListParams describes changes for [AccountService.UpdateList]. Nil fields stay as
// they are; set at least one. [Ptr] helps fill them.
type UpdateListParams struct {
	// ListID is the numeric list ID. Required.
	ListID string
	// Name is the new name, up to 25 characters. Cannot be empty.
	Name *string
	// Description is the new description, up to 100 characters. "" clears it.
	Description *string
	// Private changes the visibility.
	Private *bool
}

// CreateCommunityParams describes a community for [AccountService.CreateCommunity].
type CreateCommunityParams struct {
	// Name is the community name. Required.
	Name string
	// Description is the community description.
	Description string
	// JoinPolicy is passed to X as is. Empty uses "Open".
	JoinPolicy string
	// InvitesPolicy is passed to X as is. Empty uses "MemberInvitesAllowed".
	InvitesPolicy string
	// Question is shown to people who ask to join.
	Question string
}

// DraftsParams sets the order for [AccountService.Drafts].
type DraftsParams struct {
	// Ascending lists the oldest draft first. The default is newest first.
	Ascending bool
}

// DraftParams is the content of a draft for [AccountService.CreateDraft] and
// [AccountService.EditDraft]. Text or MediaIDs is required.
type DraftParams struct {
	// Text is the draft text, up to 25,000 characters.
	Text string
	// MediaIDs are up to 4 media IDs from [AccountService.UploadImage] or
	// [AccountService.UploadVideo].
	MediaIDs []string
}

// ImageUpload is an image or GIF for [AccountService.UploadImage].
type ImageUpload struct {
	// Data is the file content. JPEG, PNG or WebP up to 5 MB, GIF up to 15 MB, at most
	// 8192 x 8192 pixels. Required.
	Data []byte
	// MediaType is the MIME type, such as "image/png". Optional: the API detects the type
	// from the bytes.
	MediaType string
}

// VideoUpload is a video for [AccountService.UploadVideo].
type VideoUpload struct {
	// Data is the file content, MP4 or MOV. Required. Request bodies are limited to 100 MB
	// at the gateway; with base64 overhead that is a video of about 75 MB.
	Data []byte
	// MediaType is the MIME type, such as "video/mp4". Optional: the API detects the type
	// from the bytes.
	MediaType string
	// Duration is the length of the video. Required, under 4 hours. X allows 140 seconds
	// without Premium.
	Duration time.Duration
}

// BirthdateVisibility sets who can see a birth date in [Birthdate].
type BirthdateVisibility string

// Birth date visibilities.
const (
	VisibilityPublic       BirthdateVisibility = "public"
	VisibilityFollowers    BirthdateVisibility = "followers"
	VisibilityFollowing    BirthdateVisibility = "following"
	VisibilityMutualFollow BirthdateVisibility = "mutualfollow"
	VisibilitySelf         BirthdateVisibility = "self"
)

var birthdateVisibilities = []string{
	string(VisibilityPublic), string(VisibilityFollowers), string(VisibilityFollowing),
	string(VisibilityMutualFollow), string(VisibilitySelf),
}

// Birthdate is a birth date for [ProfileUpdate]. The date must be real, not in the future
// and at most 120 years back (checked by the API).
type Birthdate struct {
	Day   int
	Month int
	Year  int
	// Visibility sets who sees the day and month. Empty keeps the current setting.
	Visibility BirthdateVisibility
	// YearVisibility sets who sees the year. Empty keeps the current setting.
	YearVisibility BirthdateVisibility
}

// ProfileUpdate describes changes for [AccountService.UpdateProfile]. Nil fields stay as
// they are; set at least one. [Ptr] helps fill them.
type ProfileUpdate struct {
	// Name is the display name, up to 50 characters. Cannot be empty.
	Name *string
	// Description is the bio, up to 160 characters. "" clears it.
	Description *string
	// Location is up to 30 characters. "" clears it.
	Location *string
	// URL is the website, up to 100 characters. "" clears it.
	URL *string
	// Birthdate sets the birth date.
	Birthdate *Birthdate
	// RemoveBirthdate removes the birth date. It cannot be combined with Birthdate.
	RemoveBirthdate bool
}

// maxText checks the length of an optional text field.
func maxText(name string, v *string, max int) error {
	if v != nil && utf8.RuneCountInString(*v) > max {
		return invalidParams("%s is longer than %d characters", name, max)
	}
	return nil
}

// CreateList creates a list. POST /client/lists/create. Not retried after a timeout or
// 5xx.
func (a *AccountService) CreateList(ctx context.Context, p CreateListParams) (json.RawMessage, error) {
	name, err := nonEmpty("Name", p.Name)
	if err != nil {
		return nil, err
	}
	if err := maxText("Name", &p.Name, listNameMax); err != nil {
		return nil, err
	}
	if err := maxText("Description", &p.Description, listDescMax); err != nil {
		return nil, err
	}
	f := fields{"name": name}
	if p.Description != "" {
		f["description"] = p.Description
	}
	if p.Private {
		f["private"] = true
	}
	return a.send(ctx, "/client/lists/create", f, false)
}

// UpdateList changes a list's name, description or visibility. POST /client/lists/update.
func (a *AccountService) UpdateList(ctx context.Context, p UpdateListParams) (json.RawMessage, error) {
	id, err := numericID("ListID", p.ListID)
	if err != nil {
		return nil, err
	}
	if p.Name == nil && p.Description == nil && p.Private == nil {
		return nil, invalidParams("set at least one of Name, Description or Private")
	}
	f := fields{"list_id": id}
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return nil, invalidParams("Name cannot be empty")
		}
		f["name"] = *p.Name
	}
	if p.Description != nil {
		f["description"] = *p.Description
	}
	if p.Private != nil {
		f["private"] = *p.Private
	}
	if err := maxText("Name", p.Name, listNameMax); err != nil {
		return nil, err
	}
	if err := maxText("Description", p.Description, listDescMax); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/lists/update", f, true)
}

// DeleteList deletes one of the account's lists. POST /client/lists/delete.
func (a *AccountService) DeleteList(ctx context.Context, listID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/lists/delete", "list_id", "listID", listID, true)
}

// AddListMember adds a user to one of the account's lists. POST /client/lists/members/add.
func (a *AccountService) AddListMember(ctx context.Context, listID string, user UserRef) (json.RawMessage, error) {
	return a.listMember(ctx, "/client/lists/members/add", listID, user)
}

// RemoveListMember removes a user from one of the account's lists.
// POST /client/lists/members/remove.
func (a *AccountService) RemoveListMember(ctx context.Context, listID string, user UserRef) (json.RawMessage, error) {
	return a.listMember(ctx, "/client/lists/members/remove", listID, user)
}

func (a *AccountService) listMember(ctx context.Context, path, listID string, user UserRef) (json.RawMessage, error) {
	id, err := numericID("listID", listID)
	if err != nil {
		return nil, err
	}
	f := fields{"list_id": id}
	if err := user.addTo(f); err != nil {
		return nil, err
	}
	return a.send(ctx, path, f, true)
}

// CreateCommunity creates a community owned by the account. POST /client/communities/create.
// Not retried after a timeout or 5xx.
func (a *AccountService) CreateCommunity(ctx context.Context, p CreateCommunityParams) (json.RawMessage, error) {
	name, err := nonEmpty("Name", p.Name)
	if err != nil {
		return nil, err
	}
	f := fields{"name": name}
	for k, v := range map[string]string{
		"description": p.Description, "joinPolicy": p.JoinPolicy, "invitesPolicy": p.InvitesPolicy, "question": p.Question,
	} {
		if v != "" {
			f[k] = v
		}
	}
	return a.send(ctx, "/client/communities/create", f, false)
}

// DeleteCommunity deletes a community the account owns. X needs the community's name
// together with its ID. POST /client/communities/remove.
func (a *AccountService) DeleteCommunity(ctx context.Context, communityID, name string) (json.RawMessage, error) {
	id, err := numericID("communityID", communityID)
	if err != nil {
		return nil, err
	}
	n, err := nonEmpty("name", name)
	if err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/communities/remove", fields{"community_id": id, "name": n}, true)
}

// JoinCommunity joins a community. POST /client/communities/join.
func (a *AccountService) JoinCommunity(ctx context.Context, communityID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/communities/join", "community_id", "communityID", communityID, true)
}

// LeaveCommunity leaves a community. POST /client/communities/leave.
func (a *AccountService) LeaveCommunity(ctx context.Context, communityID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/communities/leave", "community_id", "communityID", communityID, true)
}

// CommunityMemberships returns the communities the account belongs to. Paginated.
// POST /client/communities/memberships.
func (a *AccountService) CommunityMemberships(ctx context.Context, p AccountCursorParams) (json.RawMessage, error) {
	return a.send(ctx, "/client/communities/memberships", cursorFields(p.Cursor), true)
}

// Drafts lists the account's draft tweets. POST /client/drafts.
func (a *AccountService) Drafts(ctx context.Context, p DraftsParams) (json.RawMessage, error) {
	f := fields{}
	if p.Ascending {
		f["ascending"] = true
	}
	return a.send(ctx, "/client/drafts", f, true)
}

func draftFields(p DraftParams) (fields, error) {
	media, err := mediaIDs(p.MediaIDs)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Text) == "" && len(media) == 0 {
		return nil, invalidParams("Text must be non-empty unless MediaIDs is set")
	}
	if err := maxText("Text", &p.Text, draftTextMax); err != nil {
		return nil, err
	}
	f := fields{}
	if p.Text != "" {
		f["text"] = p.Text
	}
	if len(media) > 0 {
		f["media_ids"] = media
	}
	return f, nil
}

// CreateDraft saves a draft tweet. POST /client/drafts/create. Not retried after a timeout
// or 5xx.
func (a *AccountService) CreateDraft(ctx context.Context, p DraftParams) (json.RawMessage, error) {
	f, err := draftFields(p)
	if err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/drafts/create", f, false)
}

// EditDraft replaces the content of a draft. POST /client/drafts/edit.
func (a *AccountService) EditDraft(ctx context.Context, draftID string, p DraftParams) (json.RawMessage, error) {
	id, err := numericID("draftID", draftID)
	if err != nil {
		return nil, err
	}
	f, err := draftFields(p)
	if err != nil {
		return nil, err
	}
	f["draft_tweet_id"] = id
	return a.send(ctx, "/client/drafts/edit", f, true)
}

// DeleteDraft deletes a draft. POST /client/drafts/delete.
func (a *AccountService) DeleteDraft(ctx context.Context, draftID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/drafts/delete", "draft_tweet_id", "draftID", draftID, true)
}

// UploadImage uploads an image or GIF and returns X's answer with the media ID
// (media_id_string inside response) for [CreateTweetParams.MediaIDs]. The file is sent
// base64-encoded. POST /client/media. Not retried after a timeout or 5xx.
func (a *AccountService) UploadImage(ctx context.Context, p ImageUpload) (json.RawMessage, error) {
	if len(p.Data) == 0 {
		return nil, invalidParams("Data must not be empty")
	}
	f := fields{"media": base64.StdEncoding.EncodeToString(p.Data)}
	if p.MediaType != "" {
		f["media_type"] = p.MediaType
	}
	return a.send(ctx, "/client/media", f, false)
}

// UploadVideo uploads a video and waits for X to process it. The answer holds the media
// ID for [CreateTweetParams.MediaIDs]. POST /client/media/video. Not retried after a
// timeout or 5xx.
func (a *AccountService) UploadVideo(ctx context.Context, p VideoUpload) (json.RawMessage, error) {
	if len(p.Data) == 0 {
		return nil, invalidParams("Data must not be empty")
	}
	if p.Duration < time.Millisecond || p.Duration >= maxVideoLength {
		return nil, invalidParams("Duration must be positive and under %s, got %s", maxVideoLength, p.Duration)
	}
	f := fields{
		"media":             base64.StdEncoding.EncodeToString(p.Data),
		"video_duration_ms": p.Duration.Milliseconds(),
	}
	if p.MediaType != "" {
		f["media_type"] = p.MediaType
	}
	return a.send(ctx, "/client/media/video", f, false)
}

// MediaStatus returns the processing status of an uploaded media ID.
// POST /client/media/status.
func (a *AccountService) MediaStatus(ctx context.Context, mediaID string) (json.RawMessage, error) {
	return a.idAction(ctx, "/client/media/status", "media_id", "mediaID", mediaID, true)
}

// UpdateProfile changes the name, bio, location, website or birth date. Fields left nil
// stay as they are. POST /client/profile.
func (a *AccountService) UpdateProfile(ctx context.Context, p ProfileUpdate) (json.RawMessage, error) {
	f := fields{}
	for _, t := range []struct {
		name, key string
		v         *string
		max       int
	}{
		{"Name", "name", p.Name, profileNameMax},
		{"Description", "description", p.Description, profileDescMax},
		{"Location", "location", p.Location, profileLocMax},
		{"URL", "url", p.URL, profileURLMax},
	} {
		if err := maxText(t.name, t.v, t.max); err != nil {
			return nil, err
		}
		if t.v != nil {
			f[t.key] = *t.v
		}
	}
	if p.Name != nil && strings.TrimSpace(*p.Name) == "" {
		return nil, invalidParams("Name cannot be empty")
	}
	switch {
	case p.Birthdate != nil && p.RemoveBirthdate:
		return nil, invalidParams("set Birthdate or RemoveBirthdate, not both")
	case p.RemoveBirthdate:
		f["birthdate"] = nil
	case p.Birthdate != nil:
		b := p.Birthdate
		if b.Day == 0 || b.Month == 0 || b.Year == 0 {
			return nil, invalidParams("Birthdate needs Day, Month and Year")
		}
		bd := fields{"day": b.Day, "month": b.Month, "year": b.Year}
		for k, v := range map[string]BirthdateVisibility{"visibility": b.Visibility, "year_visibility": b.YearVisibility} {
			vis, err := oneOf("Birthdate "+k, string(v), birthdateVisibilities)
			if err != nil {
				return nil, err
			}
			if vis != "" {
				bd[k] = vis
			}
		}
		f["birthdate"] = bd
	}
	if len(f) == 0 {
		return nil, invalidParams("set at least one of Name, Description, Location, URL, Birthdate or RemoveBirthdate")
	}
	return a.send(ctx, "/client/profile", f, true)
}

// UpdateProfileImage replaces the profile picture with data (JPEG, PNG, GIF or WebP; the
// API checks X's limits). POST /client/profile/image.
func (a *AccountService) UpdateProfileImage(ctx context.Context, data []byte) (json.RawMessage, error) {
	return a.profileMedia(ctx, "/client/profile/image", data)
}

// UpdateProfileBanner replaces the profile banner with data (the API checks X's limits).
// POST /client/profile/banner.
func (a *AccountService) UpdateProfileBanner(ctx context.Context, data []byte) (json.RawMessage, error) {
	return a.profileMedia(ctx, "/client/profile/banner", data)
}

func (a *AccountService) profileMedia(ctx context.Context, path string, data []byte) (json.RawMessage, error) {
	if len(data) == 0 {
		return nil, invalidParams("data must not be empty")
	}
	return a.send(ctx, path, fields{"media": base64.StdEncoding.EncodeToString(data)}, true)
}
