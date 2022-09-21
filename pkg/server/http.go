package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/listing"
	"go.uber.org/zap"
)

func NewHTTPServer(addr string, gateway *listing.JsonGateway, logger *zap.Logger) *http.Server {
	httpsrv := newHTTPServer(gateway, logger)
	r := mux.NewRouter()

	r.HandleFunc(constants.SEARCH_URL, httpsrv.handleSearch).Methods("POST")
	r.HandleFunc(constants.HEALTH_CHECK_URL, httpsrv.handleHealthCheck)

	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

type httpServer struct {
	gateway *listing.JsonGateway
	logger  *zap.Logger
}

type SearchRequest struct {
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	PostalCode string  `json:"postalCode"`
	Distance   int     `json:"distance"`
}

type SearchResponse struct {
	Stores []*listing.Store `json:"stores"`
	Count  int              `json:"count"`
}

func newHTTPServer(gateway *listing.JsonGateway, logger *zap.Logger) *httpServer {
	return &httpServer{
		gateway: gateway,
		logger:  logger,
	}
}

func (s *httpServer) handleHealthCheck(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Success"))
}

func (s *httpServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.logger.Error("error decoding searchRequest", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.logger.Info("searchRequest", zap.Any("request", req))
	var stores []*listing.Store
	if req.PostalCode != "" {
		stores, err = s.gateway.GetStoresForPostalCode(req.PostalCode, req.Distance)
		if err != nil {
			s.logger.Error("error getting stores", zap.Error(err), zap.String("postalCode", req.PostalCode))
			http.Error(w, err.Error(), http.StatusNoContent)
			return
		}
	} else {
		stores, err = s.gateway.GetStoresForGeoPoint(req.Latitude, req.Longitude, req.Distance)
		if err != nil {
			s.logger.Error("error getting stores", zap.Error(err), zap.Float64("latitude", req.Latitude), zap.Float64("longitude", req.Longitude))
			http.Error(w, err.Error(), http.StatusNoContent)
			return
		}
	}

	res := SearchResponse{Stores: stores, Count: len(stores)}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
