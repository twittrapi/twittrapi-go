//go:build go1.23

package twittrapi

import "iter"

// All returns a Go 1.23 range-over-func iterator over the remaining pages. A failed request
// is yielded once as a non-nil error, after which iteration ends. Breaking out of the loop
// stops further requests.
//
//	for page, err := range twittrapi.Paginate(ctx, client.Users.Followers, params, nil).All() {
//		if err != nil {
//			return err
//		}
//		// use page.Data
//	}
//
// All is only available when building with Go 1.23 or later.
func (it *PageIterator) All() iter.Seq2[Page, error] {
	return func(yield func(Page, error) bool) {
		for it.Next() {
			if !yield(it.Page(), nil) {
				it.done = true
				return
			}
		}
		if err := it.Err(); err != nil {
			yield(Page{}, err)
		}
	}
}
