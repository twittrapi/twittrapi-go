package twittrapi

import (
	"context"
	"encoding/json"
	"strconv"
)

// Common WOEIDs (Where On Earth IDs) for [TrendsService.Get]. Use
// [TrendsService.Locations] for the full list.
const (
	// WOEIDWorldwide is the WOEID for worldwide trends.
	WOEIDWorldwide = 1
	// WOEIDUnitedStates is the WOEID of the United States.
	WOEIDUnitedStates = 23424977
	// WOEIDUnitedKingdom is the WOEID of the United Kingdom.
	WOEIDUnitedKingdom = 23424975
)

const maxWOEID = 999_999_999_999 // 12 digits, as in the other SDKs

// TrendsService groups trending topic endpoints. Use it through [Client.Trends].
type TrendsService struct{ client *Client }

// Get returns trending topics for a location by WOEID ([WOEIDWorldwide] is 1).
func (s *TrendsService) Get(ctx context.Context, woeid int64) (json.RawMessage, error) {
	if woeid < 0 || woeid > maxWOEID {
		return nil, invalidParams("woeid must be a numeric WOEID (0 to %d), got %d", int64(maxWOEID), woeid)
	}
	return s.client.Get(ctx, "/trends/"+strconv.FormatInt(woeid, 10), nil)
}

// Locations returns the locations (countries and cities) that have trends, with their
// WOEIDs.
func (s *TrendsService) Locations(ctx context.Context) (json.RawMessage, error) {
	return s.client.Get(ctx, "/misc/locations", nil)
}
