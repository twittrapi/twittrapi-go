package twittrapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidParams is wrapped by every error returned for invalid arguments that are caught
// before a request is sent, such as a non-numeric ID or more than 500 user IDs. Check for it
// with errors.Is.
var ErrInvalidParams = errors.New("invalid parameter")

func invalidParams(format string, args ...any) error {
	return fmt.Errorf("twittrapi: %w: %s", ErrInvalidParams, fmt.Sprintf(format, args...))
}

// APIError is an error response from the API (HTTP status 400 or higher). Status codes with
// a specific meaning are returned as one of the more specific types below, each of which
// also matches *APIError with errors.As.
type APIError struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Message is a human readable description, starting with the status code.
	Message string
	// Detail is the "error" (or "message" or "detail") string from the JSON body, if any.
	Detail string
	// Body is the raw response body.
	Body []byte
	// Header holds the response headers.
	Header http.Header
}

// Error implements the error interface.
func (e *APIError) Error() string { return "twittrapi: " + e.Message }

// AuthenticationError is returned for 401 and 403: the API key is missing, invalid or not
// allowed to make this call.
type AuthenticationError struct{ *APIError }

// Unwrap returns the underlying *APIError.
func (e *AuthenticationError) Unwrap() error { return e.APIError }

// InsufficientBalanceError is returned for 402: the account has no balance left. Top up at
// https://twittrapi.com. It is never retried.
type InsufficientBalanceError struct{ *APIError }

// Unwrap returns the underlying *APIError.
func (e *InsufficientBalanceError) Unwrap() error { return e.APIError }

// NotFoundError is returned for 404: the user, tweet, list or community does not exist, is
// private, or is suspended.
type NotFoundError struct{ *APIError }

// Unwrap returns the underlying *APIError.
func (e *NotFoundError) Unwrap() error { return e.APIError }

// ValidationError is returned for 400 and 422: the API rejected the parameters.
type ValidationError struct{ *APIError }

// Unwrap returns the underlying *APIError.
func (e *ValidationError) Unwrap() error { return e.APIError }

// RateLimitError is returned for 429: too many requests.
type RateLimitError struct {
	*APIError
	// RetryAfter is the wait the server asked for in the Retry-After header, or -1 when the
	// header was missing or could not be parsed.
	RetryAfter time.Duration
}

// Unwrap returns the underlying *APIError.
func (e *RateLimitError) Unwrap() error { return e.APIError }

// ServerError is returned for 5xx: a failure at TwittrAPI or upstream at X. Usually
// temporary.
type ServerError struct{ *APIError }

// Unwrap returns the underlying *APIError.
func (e *ServerError) Unwrap() error { return e.APIError }

// ConnectionError is returned when a request got no response: DNS failure, refused or
// dropped connection, TLS error and so on.
type ConnectionError struct {
	// Message describes what failed.
	Message string
	// Err is the underlying error from the HTTP client.
	Err error
}

// Error implements the error interface.
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("twittrapi: %s: %v", e.Message, e.Err)
	}
	return "twittrapi: " + e.Message
}

// Unwrap returns the underlying error.
func (e *ConnectionError) Unwrap() error { return e.Err }

// TimeoutError is returned when an attempt took longer than the client's timeout (see
// [WithTimeout]). It also matches *ConnectionError with errors.As. When the context passed
// to a method expires instead, the context's error is returned.
type TimeoutError struct{ *ConnectionError }

// Unwrap returns the underlying *ConnectionError.
func (e *TimeoutError) Unwrap() error { return e.ConnectionError }

// ParseRetryAfter parses a Retry-After header value given in seconds or as an HTTP date. It
// returns the wait and true, or 0 and false when the value is empty or invalid. Dates in the
// past give a wait of 0.
func ParseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseFloat(v, 64); err == nil {
		if secs < 0 || secs != secs {
			return 0, false
		}
		if secs > 1e9 {
			secs = 1e9
		}
		return time.Duration(secs * float64(time.Second)), true
	}
	t, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	d := t.Sub(now)
	if d < 0 {
		d = 0
	}
	return d, true
}

func extractDetail(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err == nil {
		for _, key := range []string{"error", "message", "detail"} {
			if s, ok := obj[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
		return ""
	}
	if json.Valid(body) {
		// Valid JSON that is not an object (a number, an array...): no usable message.
		var s string
		if json.Unmarshal(body, &s) == nil && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
		return ""
	}
	if len(text) > 500 {
		return text[:500] + "..."
	}
	return text
}

var defaultMessages = map[int]string{
	400: "Bad request",
	401: "Authentication failed: check your API key",
	402: "No balance left on this TwittrAPI account: top up to continue",
	403: "Access denied: the key may lack access to this endpoint",
	404: "Not found: the resource may not exist, be private, or be suspended",
	422: "Invalid parameters",
	429: "Rate limit exceeded",
}

// errorFromResponse maps an error status to the matching error type.
func errorFromResponse(status int, body []byte, header http.Header) error {
	detail := extractDetail(body)
	text := detail
	if text == "" {
		if m, ok := defaultMessages[status]; ok {
			text = m
		} else if status >= 500 {
			text = "Server error"
		} else {
			text = "Request failed"
		}
	}
	base := &APIError{
		StatusCode: status,
		Message:    fmt.Sprintf("%d %s", status, text),
		Detail:     detail,
		Body:       body,
		Header:     header,
	}
	switch {
	case status == 401 || status == 403:
		return &AuthenticationError{base}
	case status == 402:
		return &InsufficientBalanceError{base}
	case status == 404:
		return &NotFoundError{base}
	case status == 400 || status == 422:
		return &ValidationError{base}
	case status == 429:
		retryAfter := time.Duration(-1)
		if d, ok := ParseRetryAfter(header.Get("Retry-After"), time.Now()); ok {
			retryAfter = d
		}
		return &RateLimitError{APIError: base, RetryAfter: retryAfter}
	case status >= 500:
		return &ServerError{base}
	default:
		return base
	}
}
