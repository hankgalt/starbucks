package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/siruspen/logrus"
)

type App struct {
	log    *logrus.Logger
	Server *http.Server
	mux    *sync.Mutex
}

func New(logger *logrus.Logger) (*App, error) {
	logger.Trace("creating new store locator app instance")
	return &App{
		log:    logger,
		Server: nil,
		mux:    &sync.Mutex{},
	}, nil
}

func (a *App) CreateServer() *http.Server {
	handler := http.NewServeMux()
	server := http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf(":%d", constants.PORT),
	}
	a.log.Println("creating store locator server to serve http requests")

	handler.HandleFunc(constants.HEALTH_CHECK_URL, healthCheckHandler)

	return &server
}

func (a *App) Start() error {
	a.log.Println("starting store locator server")
	err := a.Server.ListenAndServe()
	if err != nil {
		a.log.WithError(err).Error("failed to start store locator server")
		return fmt.Errorf("failed to start store locator server")
	}
	return nil
}

func (a *App) Stop() {
	a.log.Println("stopping store locator server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.Server.Shutdown(ctx)
}

func SetupLogger() *logrus.Logger {
	log := logrus.StandardLogger()
	lvl, err := logrus.ParseLevel(os.Getenv("DEBUG_LVL"))
	if err != nil {
		log.SetLevel(lvl)
	} else {
		log.SetLevel(logrus.DebugLevel)
	}
	log.SetOutput(os.Stdout)

	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		PrettyPrint:     true,
	})
	return log
}

func healthCheckHandler(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Success"))
}
