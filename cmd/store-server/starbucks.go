package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hankgalt/starbucks/internal/config"
	"github.com/hankgalt/starbucks/pkg/app"
	appConfig "github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/jobs"
	"github.com/hankgalt/starbucks/pkg/logging"
	"github.com/hankgalt/starbucks/pkg/services/geocode"
	"github.com/hankgalt/starbucks/pkg/services/store"
	"go.uber.org/zap"
)

func main() {
	// fx.New(
	// 	fx.Provide(providers.NewZapLogger),
	// 	fx.Provide(providers.NewConfig),
	// 	fx.Provide(providers.NewStoreService),
	// 	fx.Provide(providers.NewGeocodeService),
	// 	fx.Provide(providers.NewRouteHandler),
	// 	fx.Provide(providers.NewHTTPServer),
	// 	fx.Provide(providers.NewJSONDataLoader),
	// 	fx.Invoke(providers.ProcessFile),
	// 	fx.Invoke(func(*http.Server) {}),
	// ).Run()

	// initialize logger instance
	logger := logging.NewZapLogger()

	// create app config,
	// throws app startup error if required config is missing
	cfg, err := appConfig.GetAppConfig(logger)
	if err != nil {
		logger.Fatal("unable to setup config", zap.Error(err))
		return
	}

	// create geo coding service instance,
	// requires config and logger instance
	// throws app startup error
	geoService, err := geocode.New(cfg, logger)
	if err != nil {
		logger.Fatal("error initializing maps client", zap.Error(err))
		return
	}

	// create store service instance
	storeService := store.New(logger)

	// start background job to build store data from persisted json array
	// requires storeService and logger instance
	jsonDataLoader := jobs.NewJSONDataLoader(storeService, logger)
	go jsonDataLoader.ProcessFile()

	// create app instance
	// requires geoService, storeService and logger instance
	servicePort := fmt.Sprintf(":%d", constants.SERVICE_PORT)
	app := app.NewApp(servicePort, geoService, storeService, logger)

	// start listening for requests and serving responses
	logger.Info("listening for store requests", zap.Int("port", constants.SERVICE_PORT))
	go func() {
		// service connections
		if err := app.Server.ListenAndServeTLS(config.ServerCertFile, config.ServerKeyFile); err != nil && err != http.ErrServerClosed {
			logger.Fatal("error starting server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("starting server shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Server.Shutdown(ctx); err != nil {
		logger.Fatal("error shutting down server", zap.Error(err))
	}
	<-ctx.Done()
	logger.Info("Server exiting")
}
