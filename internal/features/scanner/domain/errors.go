package domain

import "errors"

var (
	ErrCollectionCheckFailed = errors.New("collection check failed")
	ErrAddToCollectionFailed = errors.New("add to collection failed")
	ErrDiscogsNotConnected   = errors.New("discogs account not connected")
	ErrSearchFailed          = errors.New("discogs search failed")
)
