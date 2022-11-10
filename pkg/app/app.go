package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/services/geocode"
	"github.com/hankgalt/starbucks/pkg/services/store"
	"go.uber.org/zap"
)

func NewApp(a string, gcs *geocode.GeoCodeService, s *store.StoreService, l *zap.Logger) *app {
	app := newApp(gcs, s, l)
	r := mux.NewRouter()

	r.HandleFunc(constants.SEARCH_URL, app.handleSearch).Methods("POST")
	r.HandleFunc(constants.STATS_CHECK_URL, app.handleStatsCheck)
	r.HandleFunc(constants.HEALTH_CHECK_URL, app.handleHealthCheck)

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	httpSrv := &http.Server{
		Addr:      a,
		Handler:   r,
		TLSConfig: cfg,
	}

	app.Server = httpSrv
	return app
}

type app struct {
	Server  *http.Server
	geocode *geocode.GeoCodeService
	store   *store.StoreService
	logger  *zap.Logger
}

type SearchRequest struct {
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	PostalCode string  `json:"postalCode"`
	Distance   int     `json:"distance"`
}

type SearchResponse struct {
	Stores []*store.Store `json:"stores"`
	Count  int            `json:"count"`
}

type StatsResponse struct {
	Stats store.StoreStats `json:"stats"`
}

func newApp(gcs *geocode.GeoCodeService, s *store.StoreService, l *zap.Logger) *app {
	return &app{
		geocode: gcs,
		store:   s,
		logger:  l,
	}
}

func (s *app) handleHealthCheck(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Success"))
}

func (s *app) handleStatsCheck(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	stats := s.store.GetStoreStats()
	res := StatsResponse{Stats: stats}

	err := json.NewEncoder(rw).Encode(res)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *app) handleSearch(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	var req SearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.logger.Error("error decoding searchRequest", zap.Error(err))
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	s.logger.Info("searchRequest", zap.Any("request", req))
	if (req.Latitude == 0 || req.Longitude == 0) && req.PostalCode != "" {
		point, err := s.geocode.Geocode(ctx, req.PostalCode, "US")
		if err != nil {
			s.logger.Error("error geocoding postalcode", zap.Error(err), zap.String("postalcode", req.PostalCode))
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		req.Latitude = point.Latitude
		req.Longitude = point.Longitude
	}

	var stores []*store.Store
	stores, err = s.store.GetStoresForGeoPoint(ctx, req.Latitude, req.Longitude, req.Distance)
	if err != nil {
		s.logger.Error("error getting stores", zap.Error(err), zap.Float64("latitude", req.Latitude), zap.Float64("longitude", req.Longitude))
		http.Error(rw, err.Error(), http.StatusNoContent)
		return
	}

	res := SearchResponse{Stores: stores, Count: len(stores)}
	err = json.NewEncoder(rw).Encode(res)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}
