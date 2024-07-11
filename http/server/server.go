package server

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/kylycht/kviku/cache"
	"github.com/kylycht/kviku/cache/inmem"
	"github.com/kylycht/kviku/http/handler/load"
	"github.com/kylycht/kviku/http/handler/store"
	"github.com/sirupsen/logrus"
)

type server string

const (
	Master server = "master"
	Slave  server = "slave"
)

func New(serverType server, addr string, slaveAddr string) *Server {
	srv := &Server{
		mux:         http.NewServeMux(),
		serverAddr:  addr,
		replicaAddr: slaveAddr,
		shutdownC:   make(chan struct{}),
		serverType:  serverType,
	}

	return srv
}

type Server struct {
	serverType  server
	serverAddr  string
	replicaAddr string
	cache       cache.Cache
	mux         *http.ServeMux

	replicaData struct {
		replicaInC        chan cache.Item
		replicaOutC       chan cache.Item
		replicaRingBuffer *ringBuffer
		replicaClient     *http.Client
	}

	stopC     chan os.Signal
	shutdownC chan struct{}
}

func (s *Server) Start() error {
	logrus.Debug("starting server initialization")
	return s.init()
}

func (s *Server) init() error {
	s.stopC = make(chan os.Signal)
	s.cache = inmem.New(s.shutdownC)

	if s.serverType == Master {
		logrus.Debug("starting replication job")
		s.initReplica()
		go s.replicate()
	}

	s.routes()

	signal.Notify(s.stopC, os.Interrupt)

	go s.stop()

	logrus.Debug("starting http server ", s.serverAddr)

	if err := http.ListenAndServe(s.serverAddr, s.mux); err != nil {
		return err
	}

	return nil
}

func (s *Server) routes() {
	s.mux.Handle("/store", store.New(s.cache, s.replicaData.replicaInC))
	s.mux.Handle("/load", load.New(s.cache))
}

// initReplica will initialize connection to replica node
// over http2 if possible and POST any new data
func (s *Server) initReplica() {
	logrus.Debug("initializing replica node connection")
	s.replicaData.replicaInC = make(chan cache.Item)
	s.replicaData.replicaOutC = make(chan cache.Item, 1024)
	s.replicaData.replicaRingBuffer = newRingBuffer(s.replicaData.replicaInC, s.replicaData.replicaOutC)

	go s.replicaData.replicaRingBuffer.run()

	s.replicaData.replicaClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 1 * time.Minute,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     60 * time.Second,
		},
		Timeout: 1 * time.Second,
	}
	logrus.Debug("transport ready")
}

func (s *Server) replicate() {
	for item := range s.replicaData.replicaOutC {
		logrus.WithFields(logrus.Fields{
			"key":   item.Key(),
			"value": item.Value(),
			"ttl":   item.TTL().UTC(),
		}).Debug("replicating item into slave node")

		queryParams := url.Values{}
		queryParams.Add("key", item.Key())
		queryParams.Add("value", item.Value())
		queryParams.Add("expires_at", item.TTL().UTC().Format(time.RFC3339))

		fullURL := fmt.Sprintf("http://%s/store?%s", s.replicaAddr, queryParams.Encode())
		req, err := http.NewRequest("POST", fullURL, nil)
		if err != nil {
			logrus.Errorf("error creating request: %v\n", err)
			continue
		}

		resp, err := s.replicaData.replicaClient.Do(req)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"key":   item.Key(),
				"value": item.Value(),
				"ttl":   item.TTL().UTC(),
			}).WithError(err).Error("unable to store into replica")
			continue
		}

		resp.Body.Close()
		if resp.StatusCode != 200 {
			logrus.WithFields(logrus.Fields{
				"key":        item.Key(),
				"value":      item.Value(),
				"ttl":        item.TTL().UTC(),
				"statusCode": resp.StatusCode,
				"fullURL":    fullURL,
			}).WithError(err).Error("received non-200 status code")
			continue
		}

		logrus.Debug("successfully replicated data into slave node")
	}
}

func (s *Server) stop() {
	<-s.stopC
	s.shutdownC <- struct{}{}
	logrus.Debug("shutting down service")
	if s.replicaData.replicaClient != nil {
		close(s.replicaData.replicaInC)
		s.replicaData.replicaClient.CloseIdleConnections()
	}
	os.Exit(0)
}

func newRingBuffer(inCh, outCh chan cache.Item) *ringBuffer {
	return &ringBuffer{
		inCh:  inCh,
		outCh: outCh,
	}
}

type ringBuffer struct {
	inCh  chan cache.Item
	outCh chan cache.Item
}

func (r *ringBuffer) run() {
	for v := range r.inCh {
		select {
		case r.outCh <- v:
		default:
			<-r.outCh
			r.outCh <- v
		}
	}
	close(r.outCh)
}
