package discogs

type (
	RateLimitError struct{}

	DiscogsAuthError struct{}
)

func (e *RateLimitError) Error() string {
	return "discogs rate limit exceeded"
}

func (e *DiscogsAuthError) Error() string {
	return "discogs token revoked or unauthorised"
}
