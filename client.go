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

func NewTransport(serverUrl string, headers map[string]string, timeOut time.Duration, retries int) *Transport {
	return &Transport{
		ServerUrl: serverUrl,
		Headers:   headers,
		TimeOut:   timeOut,
		Retries:   retries,
	}
}

func (t *Transport) Client() *http.Client {
	return httpretry.NewCustomClient(&http.Client{
		Transport: t,
		Timeout:   t.TimeOut,
	},
		httpretry.WithMaxRetryCount(t.Retries),
	)
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	req := *r
	rawQuery := req.URL.RawQuery
	path := req.URL.Path
	u, err := url.Parse(fmt.Sprint(t.ServerUrl, path, "?", rawQuery))
	if err != nil {
		return &http.Response{}, err
	}
	req.URL = u
	for k, v := range t.Headers {
		req.Header.Add(k, v)
	}
	return t.transport().RoundTrip(&req)
}
