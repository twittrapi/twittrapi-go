# Changelog

All notable changes to this module are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/). Until 1.0.0, minor versions may contain breaking changes.

## [Unreleased]

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

[Unreleased]: https://github.com/twittrapi/twittrapi-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/twittrapi/twittrapi-go/releases/tag/v0.1.0
