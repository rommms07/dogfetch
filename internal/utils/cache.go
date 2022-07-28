package utils

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type CacheResponse struct {
	e_at       time.Time
	cache_path string
	http.Response
}

/**
 * NewCacheResponse
 *
 * Creates a new cache response by fetching the resUrl parameter value.
 */
func NewCacheResponse(resUrl string) (key string) {
	key = getSha512Sum(resUrl)

	if cache := fetch(resUrl); cache != nil {
		mkCacheFrom(cache)
		return
	}

	return
}

var fetch = func(resUrl string) *http.Response {
	if _, err := atmptLoadStoredCache(resUrl); err == nil {
		return nil
	}

	tout := time.Second * 8
	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(tout))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resUrl, nil)

	defer cancel()

	if err != nil {
		log.Fatalf("Something went wrong while creating a new request. (err: %v)", err)
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("Something went wrong while fetch the request. (err: %v)", err)
	}

	return res
}

var atmptLoadStoredCache = func(resUrl string) (*CacheResponse, error) {
	return nil, errors.New("not implemented")
}

var mkCacheFrom = func(res *http.Response) *CacheResponse {
	return &CacheResponse{}
}

var getCache = func(resUrl string) *CacheResponse {
	sum := getSha512Sum(resUrl)

	log.Println(sum)

	return &CacheResponse{}
}

func (res *CacheResponse) Delete() {}
