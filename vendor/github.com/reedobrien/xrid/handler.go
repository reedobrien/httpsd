// Package xrid provides a middleware handler for setting or passing on the
// correlation ID from the `X-Request-ID header. The handler sets the value if
// it doesn't exist.
package xrid

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid"
	"github.com/rs/xid"
)

type key int

const (
	requestIDKey key = 0
	headerKey        = "X-Request-ID"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
var entropy = ulid.Monotonic(rng, 0)

func newContextWithID(r *http.Request) context.Context {
	id := r.Header.Get(headerKey)
	if id == "" {
		return context.WithValue(r.Context(), requestIDKey, newXID())

	}
	return context.WithValue(r.Context(), requestIDKey, id)
}

func newID() string {
	return ulid.MustNew(
		ulid.Timestamp(time.Now()),
		entropy,
	).String()
}

func newXID() string {
	return xid.New().String()
}

// FromContext returns the correlation ID stored in the given context.
func FromContext(ctx context.Context) string {
	return ctx.Value(requestIDKey).(string)
}

// Handler wraps the provided handler with one which passes the request id to
// the next handler and sets it on the response.
func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := newContextWithID(r)
			next.ServeHTTP(w, r.WithContext(ctx))
			w.Header().Set(headerKey, FromContext(ctx))
		})
}
