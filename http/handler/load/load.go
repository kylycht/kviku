package load

import (
	"net/http"

	"github.com/kylycht/kviku/cache"
)

func New(cache cache.Cache) http.Handler {
	return &retriver{cache: cache}
}

type retriver struct {
	cache cache.Cache
}

// ServeHTTP implements http.Handler.
func (r *retriver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// empty key is a valid key
	key := req.URL.Query().Get("key")
	// empty value is a valid value
	value, isPresent := r.cache.Get(key)
	if !isPresent {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Write([]byte(value))
}
