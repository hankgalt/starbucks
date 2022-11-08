package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/services/geocode"
	"github.com/hankgalt/starbucks/pkg/services/store"
	"go.uber.org/zap"
)

func NewApp(a string, gcs *geocode.GeoCodeService, s *store.StoreService, l *zap.Logger) *http.Server {
	httpsrv := newApp(gcs, s, l)
	r := mux.NewRouter()

	r.HandleFunc(constants.SEARCH_URL, httpsrv.handleSearch).Methods("POST")
	r.HandleFunc(constants.STATS_CHECK_URL, httpsrv.handleStatsCheck)
	r.HandleFunc(constants.HEALTH_CHECK_URL, httpsrv.handleHealthCheck)

	return &http.Server{
		Addr:    a,
		Handler: r,
	}
}

type app struct {
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
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Success"))
}

func (s *app) handleStatsCheck(rw http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	stats := s.store.GetStoreStats(ctx)
	res := StatsResponse{Stats: stats}

	err := json.NewEncoder(rw).Encode(res)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *app) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.logger.Error("error decoding searchRequest", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := context.Background()
	s.logger.Info("searchRequest", zap.Any("request", req))
	if (req.Latitude == 0 || req.Longitude == 0) && req.PostalCode != "" {
		point, err := s.geocode.Geocode(ctx, req.PostalCode, "US")
		if err != nil {
			s.logger.Error("error geocoding postalcode", zap.Error(err), zap.String("postalcode", req.PostalCode))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Latitude = point.Latitude
		req.Longitude = point.Longitude
	}

	var stores []*store.Store
	stores, err = s.store.GetStoresForGeoPoint(ctx, req.Latitude, req.Longitude, req.Distance)
	if err != nil {
		s.logger.Error("error getting stores", zap.Error(err), zap.Float64("latitude", req.Latitude), zap.Float64("longitude", req.Longitude))
		http.Error(w, err.Error(), http.StatusNoContent)
		return
	}

	res := SearchResponse{Stores: stores, Count: len(stores)}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
