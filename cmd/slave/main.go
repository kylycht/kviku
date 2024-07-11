package main

import (
	"flag"

	"github.com/kylycht/kviku/http/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("starting caching service")

	listenAddr := flag.String("listenAddr", ":8888", "address to listen to")
	flag.Parse()

	srv := server.New(server.Slave, *listenAddr, "")

	if err := srv.Start(); err != nil {
		logrus.WithError(err).Error("unable to start slave server")
	}
}
