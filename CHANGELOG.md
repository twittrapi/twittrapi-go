# Changelog

All notable changes to this module are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [1.0.0] - 2026-10-15

First stable release: the API is stable and follows semantic versioning.

### Added

- `WithRapidAPIHost` and the `RAPIDAPI_HOST` environment variable set the RapidAPI host for
  keys from another TwittrAPI listing on RapidAPI. The host is sent as `x-rapidapi-host` and
  requests go to `https://<host>` unless `WithBaseURL` is set. `https://` and a trailing slash
  are accepted; an invalid host makes `NewClient` return an error. The default stays
  `twitter-aio.p.rapidapi.com`. `Client.RapidAPIHost` returns the host in use.

## [0.3.0] - 2026-10-14

### Changed

- `Users.GetMany` accepts up to 500 IDs per call (was 100), matching the API.

- **Breaking:** `Tweets.Likers` now takes `LikersParams` (`TweetID`, `Cookie`, `Proxy`, `Count`,
  `Cursor`) and sends a POST to `/tweet/{id}/favorites` with cookie, proxy, count and cursor in
  a JSON body. The API no longer accepts GET for this endpoint. It works only for your own
  tweets: X shows who liked a tweet only to its author, so `Cookie` must be the Cookie header
  (with `auth_token` and `ct0`) of the account that posted it. `Proxy` is
  `http://user:pass@host:port`. An empty `Cookie` or `Proxy` returns `ErrInvalidParams`
  without sending a request.

### Added

- `Client.Post` sends an authenticated POST with a JSON body, with the same retries, error
  handling and simple-format support as `Client.Get`.

## [0.2.0] - 2026-10-11

### Added

- Opt-in simple response format (`format=simple`). `WithFormat(FormatSimple)` sets it for the
  client, `ContextWithFormat` for one call. The default stays `FormatRaw`, which sends no
  format parameter, so existing code gets the same requests and responses as before.
- Typed simple-format models: `SimpleTweet`, `SimpleUser`, `SimpleUserAbout`, `SimpleList`,
  `SimpleCommunity`, `SimpleTrend`, `SimpleTrendLocation` and the `SimplePage[T]` envelope
  (`Data`, `NextCursor`, `PreviousCursor`, `HasNextPage`).
- `FetchSimple` and `FetchSimplePage` call any service method in the simple format and decode
  the result. `DecodeSimple` and `DecodeSimplePage` decode bodies you fetched yourself;
  `ErrNotSimpleFormat` is returned for bodies that are not in the simple format.
- `PaginateSimple` iterates typed simple pages, following `nextCursor` while `hasNextPage` is
  true. `Paginate` also recognizes simple-format pages and stops when `hasNextPage` is false.
- Tests against the API's simple-format fixtures in `testdata/simple`.
- Table tests against trimmed real API responses in `testdata/live` for every paginated shape.

### Fixed

- `Paginate` stops when a slice-shaped page (community search, community members, job search:
  `items_results` + `slice_info.next_cursor`) has no items, instead of following its cursor.
- Examples: `extract.Tweets` uses the original full text for retweets (new `RetweetOf` field)
  and `extract.Users` also reads `{"result": User}` items (users by IDs, community members).

## [0.1.0] - 2026-10-10

### Added

- `NewClient` with direct API (`X-API-KEY`) and RapidAPI auth, keys from options or
  `TWITTRAPI_API_KEY` / `RAPIDAPI_KEY`. Options: `WithAPIKey`, `WithRapidAPIKey`, `WithBaseURL`,
  `WithHTTPClient`, `WithTimeout`, `WithMaxRetries`, `WithRetryDelay`, `WithUserAgent`.
- Service methods for all 23 read endpoints: `Users`, `Tweets`, `Search`, `Lists`,
  `Communities`, `Trends`. Every method takes a `context.Context` and returns `json.RawMessage`.
- Typed errors for use with `errors.As`: `APIError`, `AuthenticationError`,
  `InsufficientBalanceError`, `NotFoundError`, `ValidationError`, `RateLimitError` (with
  `RetryAfter`), `ServerError`, `ConnectionError`, `TimeoutError`. `ErrInvalidParams` for
  arguments rejected before a request is sent.
- Retries with exponential backoff for 429, 5xx, timeouts and network errors.
- `Paginate` iterator that follows X's bottom cursor (`Next`/`Page`/`Err`, `ForEach`, and
  `All` for range-over-func on Go 1.23+), plus `ExtractCursors` and `ExtractBottomCursor`.
- Standard library only.

[1.0.0]: https://github.com/twittrapi/twittrapi-go/compare/v0.3.0...v1.0.0
[0.3.0]: https://github.com/twittrapi/twittrapi-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/twittrapi/twittrapi-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/twittrapi/twittrapi-go/releases/tag/v0.1.0
