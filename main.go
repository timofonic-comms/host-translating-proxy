package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/handlers"
)

// Requests
// - Host header
// - Referer [sic] header
//
// Reponses
// - Location header (redirect)
// - Cookie domain scope
// - Body content

func main() {
	backendURL := os.Getenv("BACKEND_URL")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	p, err := newProxy(backendURL)
	if err != nil {
		panic(err)
	}
	h := handlers.CompressHandler(p.handler())

	fmt.Println("Proxy listening on", port)
	http.ListenAndServe(":"+port, h)
}

type proxy struct {
	backendURL *url.URL
}

func newProxy(backendURL string) (proxy, error) {
	parsedBackendURL, err := url.Parse(backendURL)
	if err != nil {
		return proxy{}, err
	}

	return proxy{backendURL: parsedBackendURL}, nil
}

func (p proxy) modifyBackendRequest(req *http.Request) {
	// Forward to backend
	req.URL.Scheme = p.backendURL.Scheme
	req.URL.Host = p.backendURL.Host

	// Explicitly set the host header, so the backend is unaware that the
	// request was originally for a different host. Golang treats the Host
	// header separate from most other headers and does not store it in the
	// Header map.
	req.Host = p.backendURL.Host

	// Explicitly disable User-Agent so it's not set to default value.
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "")
	}

	// Update Referer header
	mapHeaderVals(req.Header, "Referer", func(s string) string {
		return strings.Replace(
			s,
			stripPort(req.Host),
			stripPort(p.backendURL.Host),
			-1,
		)
	})
}

func (p proxy) modifyBackendResponse(resp *http.Response) error {
	backendHost := stripPort(p.backendURL.Host)
	reqHost := stripPort(resp.Request.Host)

	// Location header
	mapHeaderVals(resp.Header, "Location", func(s string) string {
		return strings.Replace(s, backendHost, reqHost, -1)
	})

	// Cookie domains: replace instances of backend host with request host.
	mapHeaderVals(resp.Header, "Set-Cookie", func(s string) string {
		return strings.Replace(s, backendHost, reqHost, -1)
	})

	// Body content: replace instances of backend host with request host.
	// For now, only perform replacement for web content mimetypes: HTML, CSS, JS.
	var contentType string
	if len(resp.Header["Content-Type"]) > 0 {
		contentType = resp.Header["Content-Type"][0]
	}

	mimePrefixes := []string{
		"text/html",
		"text/css",
		"application/javascript",
		"application/json",
	}

	if hasAnyPrefix(contentType, mimePrefixes) {
		// TODO: make this perform stream decoding and replacing.
		var contentReader io.Reader = resp.Body
		defer resp.Body.Close()

		if resp.Header.Get("Content-Encoding") == "gzip" {
			var err error
			contentReader, err = gzip.NewReader(contentReader)
			if err != nil {
				return err
			}
		}

		content, err := ioutil.ReadAll(contentReader)
		if err != nil {
			return err
		}

		resp.Body = newClosingBuffer(bytes.Replace(
			content,
			[]byte(backendHost),
			[]byte(reqHost),
			-1,
		))

		// Clear content-related headers
		resp.Header.Del("Content-Length")
		resp.Header.Del("Content-Encoding")
	}

	return nil
}

func (p proxy) handler() http.Handler {
	// Caveat: httputil.ReverseProxy is not 100% transparent, since it sets an
	// X-Forwarded-For header that is visible to the backend.
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			p.modifyBackendRequest(req)
		},
		ModifyResponse: func(resp *http.Response) error {
			return p.modifyBackendResponse(resp)
		},
	}
}

type closingBuffer struct {
	*bytes.Buffer
}

func (_ *closingBuffer) Close() error {
	return nil
}

func newClosingBuffer(buf []byte) *closingBuffer {
	return &closingBuffer{Buffer: bytes.NewBuffer(buf)}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func mapHeaderVals(header http.Header, field string, update func(string) string) {
	h := header[field]
	if len(h) > 0 {
		var newVals []string
		for _, v := range h {
			newVals = append(newVals, update(v))
		}
		header[field] = newVals
	}
}

func stripPort(s string) string {
	return strings.Split(s, ":")[0]
}
