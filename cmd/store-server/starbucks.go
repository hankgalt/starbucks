package main

import (
	"fmt"
	"log"

	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/listing"
	"github.com/hankgalt/starbucks/pkg/logging"
	"github.com/hankgalt/starbucks/pkg/server"
	"go.uber.org/zap"
)

func main() {
	logging.InitializeLogger()

	config, err := config.GetConfig()
	if err != nil {
		logging.Logger.Error("unable to setup config", zap.Error(err))
		return
	}
	gateway := listing.NewJasonGateway(config, logging.Logger)
	gateway.ProcessFile()

	srv := server.NewHTTPServer(fmt.Sprintf(":%d", constants.SERVICE_PORT), gateway, logging.Logger)
	logging.Logger.Info("listening for store requests", zap.Int("port", constants.SERVICE_PORT))
	log.Fatal(srv.ListenAndServe())
}
