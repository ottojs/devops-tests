package main

import (
	"sync/atomic"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// processedRequest holds pre-processed request data for better performance
type processedRequest struct {
	method  string
	url     string
	body    []byte
	headers map[string][]string
}

// Creates a targeter that rotates through requests
func createRotatingTargeter(requests []RequestConfig) vegeta.Targeter {
	var counter uint64

	// Pre-process requests to create header maps
	processed := make([]processedRequest, len(requests))
	for i, req := range requests {
		pr := processedRequest{
			method:  req.Method,
			url:     req.URL,
			headers: make(map[string][]string),
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

		// Clone the pre-built headers map
		tgt.Header = make(map[string][]string, len(req.headers))
		for k, v := range req.headers {
			tgt.Header[k] = v
		}

		return nil
	}
}
