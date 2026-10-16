package twittrapi

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	dmPasscodeMax = 64
	dmTextMax     = 10000
	dmTTLMin      = time.Minute
	dmTTLMax      = 365 * 24 * time.Hour
)

var (
	conversationIDPattern = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,128}$`)
	dmUserIDPattern       = regexp.MustCompile(`^\d{1,20}$`)
)

// DMChat selects a direct message conversation: by ConversationID, or for a one-to-one
// chat by the other person's UserID. When both are set, ConversationID is used.
type DMChat struct {
	// ConversationID is the ID from [AccountService.DMInbox]. A one-to-one chat has the ID
	// <smaller user ID>:<larger user ID>.
	ConversationID string
	// UserID is the numeric ID of the other person in a one-to-one chat.
	UserID string
}

func (c DMChat) addTo(f fields) error {
	return addDMChat(f, c.ConversationID, c.UserID)
}

func addDMChat(f fields, conversationID, userID string) error {
	if id := strings.TrimSpace(conversationID); id != "" {
		if !conversationIDPattern.MatchString(id) {
			return invalidParams("ConversationID may hold letters, digits, colon, hyphen and underscore, up to 128 characters")
		}
		f["conversation_id"] = id
		return nil
	}
	if id := strings.TrimSpace(userID); id != "" {
		if !dmUserIDPattern.MatchString(id) {
			return invalidParams("UserID must be a numeric ID of up to 20 digits, got %q", userID)
		}
		f["user_id"] = id
		return nil
	}
	return invalidParams("provide a ConversationID or UserID")
}

// DMInboxParams configures [AccountService.DMInbox].
type DMInboxParams struct {
	// Count is the number of conversations, from 1 to 50. 0 uses the API default (20).
	Count int
}

// DMConversationParams selects a conversation and a page for
// [AccountService.DMConversation].
type DMConversationParams struct {
	// ConversationID or UserID selects the conversation; see [DMChat].
	ConversationID string
	// UserID is the other person's numeric ID in a one-to-one chat.
	UserID string
	// Cursor is the cursor field of the previous answer, to get older messages. Leave empty
	// for the newest page.
	Cursor string
}

// SendDMParams describes a message for [AccountService.SendDM].
type SendDMParams struct {
	// ConversationID or UserID selects the conversation; see [DMChat].
	ConversationID string
	// UserID is the other person's numeric ID in a one-to-one chat.
	UserID string
	// Text is the message, up to 10,000 characters. Required.
	Text string
}

// DMDisappearingParams configures [AccountService.SetDMDisappearing].
type DMDisappearingParams struct {
	// ConversationID or UserID selects the conversation; see [DMChat].
	ConversationID string
	// UserID is the other person's numeric ID in a one-to-one chat.
	UserID string
	// TTL is how long messages stay, from 1 minute to 365 days, in whole seconds. 0 turns
	// disappearing messages off.
	TTL time.Duration
}

// DMSettingsUpdate describes changes for [AccountService.UpdateDMSettings]. Nil fields stay
// as they are; set at least one. [Ptr] helps fill them.
type DMSettingsUpdate struct {
	// AllowDMsFrom sets who can message you, X's value such as "all" or "verified".
	// [AccountService.DMSettings] shows the current one.
	AllowDMsFrom *string
	// DMReceiptSettings sets read receipts, X's value as shown by [AccountService.DMSettings].
	DMReceiptSettings *string
	// AlwaysAllowDMsFromSubscribers lets your subscribers always message you.
	AlwaysAllowDMsFromSubscribers *bool
	// AVCallsEnabled turns audio and video calls on or off.
	AVCallsEnabled *bool
	// AcceptCallsFromEveryone accepts calls from everyone.
	AcceptCallsFromEveryone *bool
	// AcceptCallsFromFollowing accepts calls from accounts you follow.
	AcceptCallsFromFollowing *bool
	// AcceptCallsFromVerified accepts calls from verified accounts.
	AcceptCallsFromVerified *bool
	// AcceptCallsFromAddressbook accepts calls from your address book.
	AcceptCallsFromAddressbook *bool
}

// passcode adds the passcode to f. With required set, a missing passcode is an error.
func (a *AccountService) passcode(f fields, required bool) error {
	if a.dmPasscode == "" {
		if required {
			return invalidParams("this call needs the X Chat passcode: use WithDMPasscode")
		}
		return nil
	}
	if len(a.dmPasscode) > dmPasscodeMax {
		return invalidParams("the DM passcode is longer than %d characters", dmPasscodeMax)
	}
	f["passcode"] = a.dmPasscode
	return nil
}

// DMInbox lists the account's conversations. With a passcode (see
// [AccountService.WithDMPasscode]) the latest messages are decrypted and included; without
// it the conversations are listed without messages. The cookie needs twid.
// POST /client/dm/inbox.
func (a *AccountService) DMInbox(ctx context.Context, p DMInboxParams) (json.RawMessage, error) {
	if p.Count != 0 && (p.Count < 1 || p.Count > dmInboxMaxCount) {
		return nil, invalidParams("Count must be from 1 to %d, got %d", dmInboxMaxCount, p.Count)
	}
	f := fields{}
	if err := a.passcode(f, false); err != nil {
		return nil, err
	}
	if p.Count != 0 {
		f["count"] = p.Count
	}
	return a.send(ctx, "/client/dm/inbox", f, true)
}

// DMConversation returns the decrypted messages of a conversation, newest first, with a
// cursor for older messages. Needs the passcode ([AccountService.WithDMPasscode]).
// POST /client/dm/conversation.
func (a *AccountService) DMConversation(ctx context.Context, p DMConversationParams) (json.RawMessage, error) {
	f := fields{}
	if err := addDMChat(f, p.ConversationID, p.UserID); err != nil {
		return nil, err
	}
	if c := strings.TrimSpace(p.Cursor); c != "" {
		if !dmUserIDPattern.MatchString(c) {
			return nil, invalidParams("Cursor must be digits only (up to 20), got %q", p.Cursor)
		}
		f["cursor"] = c
	}
	if err := a.passcode(f, true); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/dm/conversation", f, true)
}

// DMConversationInfo returns a conversation's settings and participants. It needs no
// passcode. POST /client/dm/conversation/info.
func (a *AccountService) DMConversationInfo(ctx context.Context, chat DMChat) (json.RawMessage, error) {
	f := fields{}
	if err := chat.addTo(f); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/dm/conversation/info", f, true)
}

// SetDMDisappearing turns disappearing messages on (TTL) or off (TTL 0) for a
// conversation. Needs the passcode ([AccountService.WithDMPasscode]).
// POST /client/dm/conversation/disappearing.
func (a *AccountService) SetDMDisappearing(ctx context.Context, p DMDisappearingParams) (json.RawMessage, error) {
	f := fields{}
	if err := addDMChat(f, p.ConversationID, p.UserID); err != nil {
		return nil, err
	}
	if p.TTL != 0 && (p.TTL < dmTTLMin || p.TTL > dmTTLMax || p.TTL%time.Second != 0) {
		return nil, invalidParams("TTL must be 0 (off) or whole seconds from %s to %s, got %s", dmTTLMin, dmTTLMax, p.TTL)
	}
	f["ttl_seconds"] = int64(p.TTL / time.Second)
	if err := a.passcode(f, true); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/dm/conversation/disappearing", f, true)
}

// SendDM sends an encrypted direct message. Needs the passcode
// ([AccountService.WithDMPasscode]). POST /client/dm/send. Not retried after a timeout or
// 5xx.
func (a *AccountService) SendDM(ctx context.Context, p SendDMParams) (json.RawMessage, error) {
	f := fields{}
	if err := addDMChat(f, p.ConversationID, p.UserID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Text) == "" {
		return nil, invalidParams("Text must be a non-empty string")
	}
	if n := utf8.RuneCountInString(p.Text); n > dmTextMax {
		return nil, invalidParams("Text is longer than %d characters (%d)", dmTextMax, n)
	}
	f["text"] = p.Text
	if err := a.passcode(f, true); err != nil {
		return nil, err
	}
	return a.send(ctx, "/client/dm/send", f, false)
}

// DMSettings returns the account's direct message settings. It needs no passcode.
// POST /client/dm/settings.
func (a *AccountService) DMSettings(ctx context.Context) (json.RawMessage, error) {
	return a.send(ctx, "/client/dm/settings", fields{}, true)
}

// UpdateDMSettings changes the account's direct message settings and returns the new
// settings. It needs no passcode. POST /client/dm/settings/update.
func (a *AccountService) UpdateDMSettings(ctx context.Context, p DMSettingsUpdate) (json.RawMessage, error) {
	f := fields{}
	for key, v := range map[string]*string{"allow_dms_from": p.AllowDMsFrom, "dm_receipt_settings": p.DMReceiptSettings} {
		if v != nil {
			if strings.TrimSpace(*v) == "" {
				return nil, invalidParams("%s must be a non-empty string", key)
			}
			f[key] = *v
		}
	}
	for key, v := range map[string]*bool{
		"always_allow_dms_from_subscribers": p.AlwaysAllowDMsFromSubscribers,
		"av_calls_enabled":                  p.AVCallsEnabled,
		"accept_calls_from_everyone":        p.AcceptCallsFromEveryone,
		"accept_calls_from_following":       p.AcceptCallsFromFollowing,
		"accept_calls_from_verified":        p.AcceptCallsFromVerified,
		"accept_calls_from_addressbook":     p.AcceptCallsFromAddressbook,
	} {
		if v != nil {
			f[key] = *v
		}
	}
	if len(f) == 0 {
		return nil, invalidParams("set at least one DM setting")
	}
	return a.send(ctx, "/client/dm/settings/update", f, true)
}
