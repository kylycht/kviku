package store

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kylycht/kviku/cache"
	"github.com/kylycht/kviku/model"
	"github.com/sirupsen/logrus"
)

var (
	validDuration = `"ns", "us" (or "µs"), "ms", "s", "m", "h" e.g.: "300ms", "-1.5h" or "2h45m"`
)

func New(cache cache.Cache, replicaC chan cache.Item) http.Handler {
	return &saver{
		cache:    cache,
		replicaC: replicaC,
	}
}

type saver struct {
	cache    cache.Cache
	replicaC chan cache.Item
}

// ServeHTTP implements http.Handler.
func (s *saver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var (
		expiresAt time.Time
		err       error
	)

	var (
		key        = req.URL.Query().Get("key")
		value      = req.URL.Query().Get("value")
		ttl        = req.URL.Query().Get("ttl")
		expiresAtS = req.URL.Query().Get("expires_at")
	)
	// attempt to parse duration
	logrus.WithFields(logrus.Fields{
		"key":        key,
		"value":      value,
		"ttl":        ttl,
		"expires_at": expiresAtS,
	}).Debug("received data to store")

	expiresAt, err = s.parseTTL(expiresAtS, ttl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("invalid duration. valid format: ttl=%s or expires_at=%s", validDuration, time.RFC3339Nano)))
		logrus.WithField("duration", ttl).WithError(err).Error("unable to parse duration")
		return
	}

	i := model.NewItem(key, value, expiresAt)

	if err := s.cache.Save(i); err != nil {
		logrus.WithFields(logrus.Fields{
			"key":   key,
			"value": value,
			"ttl":   ttl,
		}).WithError(err).Error("unable to store into cache")
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Write([]byte("ok"))
	// replicate item to slave nodes
	if s.replicaC != nil {
		logrus.Debug("passing item for replica node")
		s.replicaC <- i
	}
}

// if ttl and expires_at query params are present, ttl has higher priority
// expiration will be equal to date()+ttl
func (s *saver) parseTTL(expiresAtS, ttl string) (time.Time, error) {
	if ttl != "" {
		duration, err := time.ParseDuration(ttl)
		if err != nil {
			return time.Time{}, err
		}

		return time.Now().UTC().Add(duration), nil
	}

	if expiresAtS != "" {
		expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtS)
		if err != nil {
			return time.Time{}, err
		}
		logrus.Debug("item expires in ", expiresAtS)
		return expiresAt, nil
	}

	return time.Time{}, nil
}
