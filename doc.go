// Package twittrapi is the official Go client for TwittrAPI (https://twittrapi.com), a
// Twitter/X data API. It reads profiles, tweets, followers, search results, lists,
// communities and trends.
//
// The package uses only the standard library.
//
// # Authentication
//
// Create a client with a direct TwittrAPI key or a RapidAPI key:
//
//	client, err := twittrapi.NewClient(twittrapi.WithAPIKey(os.Getenv("TWITTRAPI_API_KEY")))
//	// or
//	client, err := twittrapi.NewClient(twittrapi.WithRapidAPIKey(os.Getenv("RAPIDAPI_KEY")))
//
// With no key option, NewClient reads TWITTRAPI_API_KEY and then RAPIDAPI_KEY from the
// environment. Direct keys are sent to https://api.twittrapi.com in the X-API-KEY header.
// RapidAPI keys are sent to https://twitter-aio.p.rapidapi.com in the x-rapidapi-key and
// x-rapidapi-host headers.
//
// # Responses
//
// Every method returns X's raw GraphQL JSON unchanged, as a [json.RawMessage]. Decode it
// into your own structs or into a map. Typed models may be added in a later version.
//
// # Services
//
// Endpoints are grouped by service on the client: [Client.Users], [Client.Tweets],
// [Client.Search], [Client.Lists], [Client.Communities] and [Client.Trends]. Every method
// takes a [context.Context] as its first argument.
//
//	user, err := client.Users.GetByUsername(ctx, "jack")
//
// # Pagination
//
// Paginated methods take a params struct with Count and Cursor fields. [Paginate] calls such
// a method repeatedly and follows the bottom cursor in each response:
//
//	it := twittrapi.Paginate(ctx, client.Users.Followers,
//		twittrapi.UserTimelineParams{Username: "jack", Count: 100},
//		&twittrapi.PaginateOptions{MaxPages: 5})
//	for it.Next() {
//		page := it.Page()
//		fmt.Println(page.Number, len(page.Data))
//	}
//	if err := it.Err(); err != nil {
//		log.Fatal(err)
//	}
//
// # Errors
//
// HTTP errors are returned as typed values such as [*NotFoundError] or [*RateLimitError].
// Use [errors.As] to check for them. All of them also match [*APIError]. Network failures
// are returned as [*ConnectionError] or [*TimeoutError]. Invalid arguments caught before a
// request is sent wrap [ErrInvalidParams].
//
// Requests that fail with 429, a 5xx status, a timeout or a network error are retried
// (2 times by default) with exponential backoff. Other errors are returned immediately.
//
// This package is not affiliated with X Corp.
package twittrapi
