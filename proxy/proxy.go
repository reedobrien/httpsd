package proxy

import (
	"expvar"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/reedobrien/rbp"
)

var (
	backendCounter *expvar.Map
)

func init() {
	backendCounter = expvar.NewMap("backendCounter")
}

const (
	cgiRequests   = "cgiRequests"
	nginxRequests = "nginxRequests"
	plackRequests = "plackRequests"
)

// NewReverseProxy builds a reverse proxy for webs and nginx.
func NewReverseProxy(cgis, nginx, plack *url.URL) *httputil.ReverseProxy {
	const (
		cgiPrefix   = "/cgi/"
		plackPrefix = "/do/"
	)

	director := func(req *http.Request) {
		var counterSet = false
		target := nginx
		if strings.HasPrefix(req.URL.Path, cgiPrefix) {
			target = cgis
			backendCounter.Add(cgiRequests, 1)
			counterSet = true
		}
		if strings.HasPrefix(req.URL.Path, plackPrefix) {
			target = plack
			backendCounter.Add(plackRequests, 1)
			counterSet = true
		}
		if !counterSet {
			backendCounter.Add(nginxRequests, 1)
			counterSet = true
		}

		targetQuery := target.RawQuery
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// Explicitly disable "User-Agent" so it's not set to the default
			// client value.
			req.Header.Set("User-Agent", "")
		}

		// Backwards... because we always expect https.
		var proto = "https"
		if req.TLS == nil {
			proto = "http"
		}
		req.Header.Set("X-Forwarded-Proto", proto)
	}

	bp := rbp.NewBufferPool()

	return &httputil.ReverseProxy{Director: director, BufferPool: bp}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")

	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}

	return a + b
}
