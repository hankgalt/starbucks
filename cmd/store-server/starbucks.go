package main

import (
	"fmt"
	"log"

	"github.com/hankgalt/starbucks/pkg/app"
	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/jobs"
	"github.com/hankgalt/starbucks/pkg/logging"
	"github.com/hankgalt/starbucks/pkg/services/geocode"
	"github.com/hankgalt/starbucks/pkg/services/store"
	"go.uber.org/zap"
)

func main() {
	// initialize logger instance
	logging.InitializeLogger()

	// create app config,
	// throws app startup error if required config is missing
	config, err := config.GetConfig()
	if err != nil {
		logging.Logger.Fatal("unable to setup config", zap.Error(err))
		return
	}

	// create geo coding service instance,
	// requires config and logger instance
	// throws app startup error
	geoService, err := geocode.New(config, logging.Logger)
	if err != nil {
		logging.Logger.Fatal("error initializing maps client", zap.Error(err))
		return
	}

	// create store service instance
	storeService := store.New(logging.Logger)

	// start background job to build store data from persisted json array
	// requires storeService and logger instance
	go jobs.ProcessFile(storeService, logging.Logger)

	// create app instance
	// requires geoService, storeService and logger instance
	servicePort := fmt.Sprintf(":%d", constants.SERVICE_PORT)
	app := app.NewApp(servicePort, geoService, storeService, logging.Logger)

	// start listening for requests and serving responses
	logging.Logger.Info("listening for store requests", zap.Int("port", constants.SERVICE_PORT))
	log.Fatal(app.ListenAndServe())
}
