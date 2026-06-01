package main

import (
	"net/http"
	"time"
)

// newHTTPClient builds a client tuned for load generation:
//   - redirects are NOT followed (302 from /redirect is the success signal),
//   - a large idle-connection pool so workers reuse keep-alive connections,
//   - HTTP/2 disabled to keep a predictable HTTP/1.1 connection-per-worker model.
func newHTTPClient(cfg Config) *http.Client {
	conns := cfg.maxIdleConns()
	tr := &http.Transport{
		MaxIdleConns:          conns,
		MaxIdleConnsPerHost:   conns,
		MaxConnsPerHost:       0, // unbounded; concurrency is the real limiter
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     !cfg.KeepAlive,
		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   cfg.Timeout,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Transport: tr,
		Timeout:   cfg.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
