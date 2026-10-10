//go:build go1.23

package twittrapi

import (
	"context"
	"errors"
	"testing"
)

func TestAllRangeOverFunc(t *testing.T) {
	c, cursors := pagedServer(t, map[string]string{
		"":   timelinePage(1, "c1"),
		"c1": timelinePage(1, "c2"),
		"c2": timelinePage(1, ""),
	})
	n := 0
	for page, err := range Paginate(context.Background(), c.Users.Media, UserTimelineParams{UserID: "12"}, nil).All() {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if page.Number != n {
			t.Fatalf("page number %d, want %d", page.Number, n)
		}
	}
	if n != 3 {
		t.Fatalf("pages = %d", n)
	}

	// Breaking early stops further requests.
	*cursors = nil
	for range Paginate(context.Background(), c.Users.Media, UserTimelineParams{UserID: "12"}, nil).All() {
		break
	}
	if len(*cursors) != 1 {
		t.Fatalf("requests after break = %d", len(*cursors))
	}

	// Errors are yielded once.
	c, _ = pagedServer(t, map[string]string{"": timelinePage(1, "missing")})
	var errs, pages int
	for _, err := range Paginate(context.Background(), c.Users.Media, UserTimelineParams{UserID: "12"}, nil).All() {
		if err != nil {
			var nf *NotFoundError
			if !errors.As(err, &nf) {
				t.Fatalf("unexpected error %v", err)
			}
			errs++
			continue
		}
		pages++
	}
	if pages != 1 || errs != 1 {
		t.Fatalf("pages=%d errs=%d", pages, errs)
	}
}
