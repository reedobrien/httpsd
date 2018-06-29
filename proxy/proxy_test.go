package proxy_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/reedobrien/checkers"
	"github.com/reedobrien/httpsd/proxy"
)

func TestReverseProxy(t *testing.T) {
	const (
		cgiResponse   = "cgis"
		exResponse    = "nginx"
		plackResponse = "plack"
	)

	type test struct {
		query   string
		want    string
		err     bool
		errWant string
	}

	type config struct {
		targetPath string
		tests      []test
	}

	table := []config{
		config{targetPath: "",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
		config{targetPath: "/",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
		config{targetPath: "/try/this",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
		config{targetPath: "/trailing/slash/",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
		config{targetPath: "/with?query=string",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
		config{targetPath: "?only=query",
			tests: []test{
				{"", "https-" + exResponse, false, ""},
				{"/", "https-" + exResponse, false, ""},
				{"/do/foo", "https-" + plackResponse, false, ""},
				{"/cgi/bar", "https-" + cgiResponse, false, ""},
				{"/anthing/else", "https-" + exResponse, false, ""},
				{"/do/foo?q=a&q=a&b=d", "https-" + plackResponse, false, ""},
				{"/cgi/?bar=baz&spam=eggs", "https-" + cgiResponse, false, ""},
				{"/anthing/else?still=works", "https-" + exResponse, false, ""},
			}},
	}

	for _, cfg := range table {
		cgis := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "%s-%s", r.Header.Get("X-Forwarded-Proto"), cgiResponse)
			}))
		defer cgis.Close()
		cgisURL, err := url.Parse(cgis.URL + cfg.targetPath)
		checkers.OK(t, err)

		nginx := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "%s-%s", r.Header.Get("X-Forwarded-Proto"), exResponse)
			}))
		defer nginx.Close()
		nginxURL, err := url.Parse(nginx.URL + cfg.targetPath)
		checkers.OK(t, err)

		plack := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "%s-%s", r.Header.Get("X-Forwarded-Proto"), plackResponse)
			}))
		defer plack.Close()
		plackURL, err := url.Parse(plack.URL + cfg.targetPath)
		checkers.OK(t, err)

		tut := proxy.NewReverseProxy(cgisURL, nginxURL, plackURL)

		ts := httptest.NewTLSServer(tut)
		defer ts.Close()

		client := ts.Client()

		for _, unit := range cfg.tests {
			res, err := client.Get(ts.URL + unit.query)
			if unit.err {
				checkers.Assert(t, strings.Contains(err.Error(), unit.errWant), fmt.Sprintf("Error doesn't contain expected string\n\tgot: %s\n\twant:%s", err.Error(), unit.errWant))
				continue
			}
			checkers.OK(t, err)

			b, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			checkers.OK(t, err)
			checkers.Equals(t, string(b), unit.want)
		}
	}
}
