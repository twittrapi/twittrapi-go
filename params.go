package twittrapi

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// MaxUsersPerLookup is the maximum number of user IDs [UsersService.GetMany] accepts in one
// call.
const MaxUsersPerLookup = 100

// MaxCount is the largest page size accepted by paginated methods.
const MaxCount = 100

var numericIDPattern = regexp.MustCompile(`^\d{1,30}$`)

func numericID(name, value string) (string, error) {
	v := strings.TrimSpace(value)
	if !numericIDPattern.MatchString(v) {
		return "", invalidParams("%s must be a numeric ID (digits only), got %q", name, value)
	}
	return v, nil
}

func cleanUsername(value string) (string, error) {
	v := strings.TrimSpace(value)
	v = strings.TrimSpace(strings.TrimPrefix(v, "@"))
	if v == "" {
		return "", invalidParams("username must not be empty")
	}
	return v, nil
}

func nonEmpty(name, value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", invalidParams("%s must be a non-empty string", name)
	}
	return v, nil
}

func oneOf(name, value string, allowed []string) (string, error) {
	if value == "" {
		return "", nil
	}
	for _, a := range allowed {
		if value == a {
			return value, nil
		}
	}
	return "", invalidParams("%s must be one of %s, got %q", name, strings.Join(allowed, ", "), value)
}

// validCount checks a page size: 0 (the API default) or 1 to MaxCount.
func validCount(count int) error {
	if count != 0 && (count < 1 || count > MaxCount) {
		return invalidParams("count must be an integer from 1 to %d, got %d", MaxCount, count)
	}
	return nil
}

// pageQuery validates count and returns the count and cursor query parameters.
func pageQuery(count int, cursor string) (url.Values, error) {
	if err := validCount(count); err != nil {
		return nil, err
	}
	q := url.Values{}
	if count != 0 {
		q.Set("count", strconv.Itoa(count))
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	return q, nil
}

// Pageable is implemented by the params types of paginated methods. [Paginate] uses it to
// read the starting cursor and to request the next page.
type Pageable[P any] interface {
	// PageCursor returns the cursor the params currently request, or "".
	PageCursor() string
	// WithCursor returns a copy of the params that requests the page at cursor.
	WithCursor(cursor string) P
}
