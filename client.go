package deepl

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ybbus/httpretry"
)

type Transport struct {
	ServerUrl string
	Headers   map[string]string
	TimeOut   time.Duration
	Retries   int
	Transport http.RoundTripper
}

// NewTransport returns a new Transport with the given server url, headers, timeout and retries.
func NewTransport(serverUrl string, headers map[string]string, timeOut time.Duration, retries int) *Transport {
	if retries <= 0 {
		retries = 5
	}
	if timeOut <= 0 {
		timeOut = time.Duration(5) * time.Second
	}
	return &Transport{
		ServerUrl: serverUrl,
		Headers:   headers,
		TimeOut:   timeOut,
		Retries:   retries,
	}
}

// Client returns a new http.Client with the Transport as the underlying transport.
func (t *Transport) Client() *http.Client {
	return httpretry.NewCustomClient(&http.Client{
		Transport: t,
		Timeout:   t.TimeOut,
	},
		httpretry.WithMaxRetryCount(t.Retries),
		httpretry.WithRetryPolicy(func(statusCode int, err error) bool {
			return err != nil || statusCode >= 500 || statusCode == 429 || statusCode == 0
		}),
	)
}

// transport returns the underlying RoundTripper, defaulting to http.DefaultTransport.
func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
// Sets prior defined headers and adds the host url to the request url.
func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	req := r.Clone(r.Context())
	fullURL := fmt.Sprintf("%s%s?%s", t.ServerUrl, req.URL.Path, req.URL.RawQuery)
	u, err := url.Parse(fullURL)
	if err != nil {
		return &http.Response{}, err
	}
	req.URL = u
	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}
	return t.transport().RoundTrip(req)
}
