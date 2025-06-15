package main

import (
	"sync"
	"sync/atomic"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// sync.Pool for header maps
var headerPool = sync.Pool{
	New: func() interface{} {
		return make(map[string][]string, 8)
	},
}

// Pre-processed request data for better performance
type processedRequest struct {
	method      string
	url         string
	body        []byte
	headers     map[string][]string
	headerCount int
}

// Creates a targeter that rotates through requests
func createRotatingTargeter(requests []RequestConfig) vegeta.Targeter {
	var counter uint64

	// Pre-process requests to create header maps
	processed := make([]processedRequest, len(requests))
	for i, req := range requests {
		// Calculate expected header count
		headerCount := 1 // User-Agent is always present
		if req.ContentType != "" {
			headerCount++
		}
		headerCount += len(req.Headers)

		pr := processedRequest{
			method:      req.Method,
			url:         req.URL,
			headers:     make(map[string][]string, headerCount),
			headerCount: headerCount,
		}

		// Pre-convert body
		if req.Body != "" {
			pr.body = []byte(req.Body)
		}

		// Pre-build headers
		pr.headers["User-Agent"] = []string{"otto-load-test"}
		if req.ContentType != "" {
			pr.headers["Content-Type"] = []string{req.ContentType}
		}
		for k, v := range req.Headers {
			pr.headers[k] = []string{v}
		}

		processed[i] = pr
	}

	return func(tgt *vegeta.Target) error {
		// Rotate through requests using atomic counter
		idx := int(atomic.AddUint64(&counter, 1)-1) % len(processed)
		req := processed[idx]

		// Set fields from pre-processed data
		tgt.Method = req.method
		tgt.URL = req.url
		tgt.Body = req.body

		// Get a header map from the pool
		headerMap := headerPool.Get().(map[string][]string)

		// Clear the map for reuse
		for k := range headerMap {
			delete(headerMap, k)
		}

		// Copy headers into the pooled map
		for k, v := range req.headers {
			headerMap[k] = v
		}

		tgt.Header = headerMap

		// Note: Vegeta will handle returning the map to the pool
		// after the request is completed

		return nil
	}
}
