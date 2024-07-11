package main

import (
	"flag"

	"github.com/kylycht/kviku/http/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("starting caching service")

	slaveAddr := flag.String("slaveAddr", "", "replica address host:port")
	listenAddr := flag.String("listenAddr", "", "address to listen to")
	flag.Parse()

	logrus.WithField("listenAddr", *listenAddr).WithField("slaveAddr", *slaveAddr).Debug("starting master service")

	srv := server.New(server.Master, *listenAddr, *slaveAddr)

	if err := srv.Start(); err != nil {
		logrus.WithError(err).Error("unable to start master server")
	}
}
