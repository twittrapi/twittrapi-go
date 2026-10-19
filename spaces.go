package twittrapi

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
)

var spaceIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{1,32}$`)

// SpacesService groups X Spaces endpoints. Use it through [Client.Spaces].
type SpacesService struct{ client *Client }

// Get returns a Space: title, state, schedule, host and speakers, listener counts and
// whether a replay is available. spaceID is the ID from x.com/i/spaces/<id>, such as
// "1YqKDqWqdPLsV" (letters and digits). There is no simple format: the response is X's raw
// JSON either way.
func (s *SpacesService) Get(ctx context.Context, spaceID string) (json.RawMessage, error) {
	id := strings.TrimSpace(spaceID)
	if !spaceIDPattern.MatchString(id) {
		return nil, invalidParams("spaceID must be 1 to 32 letters and digits, got %q", spaceID)
	}
	return s.client.Get(ctx, "/spaces/"+id, nil)
}
