package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hankgalt/starbucks/pkg/errors"
	"github.com/hankgalt/starbucks/pkg/utils/geohash"
	"go.uber.org/zap"

	"gitlab.com/xerra/common/vincenty"
)

type Store struct {
	Id        uint32    `json:"store_id"`
	Name      string    `json:"name"`
	Longitude float64   `json:"longitude"`
	Latitude  float64   `json:"latitude"`
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Created   time.Time `json:"created"`
}

func MapResultToStore(r map[string]interface{}) (*Store, error) {
	storeJson, err := json.Marshal(r)
	if err != nil {
		log.Println("\033[31m Error marshalling result to store json \033[0m")
		return nil, errors.WrapError(err, "Error marshalling result to store json")
	}

	var s Store
	err = json.Unmarshal(storeJson, &s)
	if err != nil {
		log.Println("\033[31m Error unmarshalling store json to store \033[0m", err)
		return nil, errors.WrapError(err, "Error unmarshalling store json to store")
	}
	return &s, nil
}

type StoreStats struct {
	Count     int
	HashCount int
	Ready     bool
}

type StoreService struct {
	mu      sync.RWMutex
	logger  *zap.Logger
	stores  map[uint32]*Store
	HashMap map[string][]uint32
	count   int
	ready   bool
}

func New(logger *zap.Logger) *StoreService {
	jg := &StoreService{
		logger:  logger,
		stores:  map[uint32]*Store{},
		HashMap: map[string][]uint32{},
		count:   0,
		ready:   false,
	}

	return jg
}

func (ss *StoreService) SetReady(ctx context.Context, ready bool) {
	ss.ready = ready
}

func (ss *StoreService) AddStore(ctx context.Context, s *Store) bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.lookup(ctx, s.Id) == nil {
		hashKey, _ := geohash.Encode(s.Latitude, s.Longitude, 8)
		hashStoreIDs, ok := ss.HashMap[hashKey]
		if !ok {
			hashStoreIDs = []uint32{}
		}
		hashStoreIDs = append(hashStoreIDs, s.Id)
		ss.HashMap[hashKey] = hashStoreIDs

		ss.stores[s.Id] = s
		ss.count++

		return true
	}
	return false
}

func (ss *StoreService) GetStore(ctx context.Context, storeId uint32) (*Store, error) {
	s := ss.lookup(ctx, storeId)
	if s == nil {
		ss.logger.Error("store doesn't exist", zap.Int("storeId", int(storeId)))
		return nil, fmt.Errorf("store with storeId %d doesn't exist", storeId)
	}
	return s, nil
}

func (ss *StoreService) GetStoresForGeoPoint(ctx context.Context, lat, long float64, dist int) ([]*Store, error) {
	ss.logger.Debug("getting stores for geopoint", zap.Float64("latitude", lat), zap.Float64("longitude", long), zap.Int("distance", dist))
	ids, err := ss.getHashStoreIds(ctx, lat, long)
	if err != nil || len(ids) < 1 {
		ss.logger.Error("no stores found", zap.Error(err))
		return nil, err
	}
	ss.logger.Debug("found stores", zap.Int("numOfStores", len(ids)), zap.Float64("latitude", lat), zap.Float64("longitude", long))

	stores := []*Store{}
	// origin := haversine.Point{Lat: lat, Lon: long}
	origin := vincenty.LatLng{Latitude: lat, Longitude: long}
	for _, v := range ids {
		store, err := ss.GetStore(ctx, v)
		if err != nil {
			ss.logger.Error("no stores found for id", zap.Int("storeId", int(v)))
		}
		// pos := haversine.Point{Lat: store.Latitude, Lon: store.Longitude}
		pos := vincenty.LatLng{Latitude: store.Latitude, Longitude: store.Longitude}
		// d := haversine.Distance(origin, pos)
		d := vincenty.Inverse(origin, pos)
		// if float64(d) <= dist*1000 {
		if d.Kilometers() <= float64(dist) {
			stores = append(stores, store)
		}
	}
	ss.logger.Debug("returning stores", zap.Int("numOfStores", len(stores)), zap.Float64("latitude", lat), zap.Float64("longitude", long), zap.Int("distance", dist))
	return stores, nil
}

func (ss *StoreService) GetStoreStats(ctx context.Context) StoreStats {
	// for k, v := range ss.HashMap {
	// 	if len(v) < 20 {
	// 		fmt.Printf("%s - %v\n", k, v)
	// 	}
	// }
	return StoreStats{
		Ready:     ss.ready,
		Count:     ss.count,
		HashCount: len(ss.HashMap),
	}
}

func (ss *StoreService) getHashStoreIds(ctx context.Context, lat, long float64) ([]uint32, error) {
	hashKey, err := geohash.Encode(lat, long, 8)
	if err != nil {
		ss.logger.Error("error creating hash key", zap.Float64("latitude", lat), zap.Float64("longitude", long))
		return nil, err
	}
	ss.logger.Debug("created hash key", zap.String("hashKey", hashKey), zap.Float64("latitude", lat), zap.Float64("longitude", long))
	ids, ok := ss.HashMap[hashKey]
	if !ok || len(ids) < 1 {
		ss.logger.Error("no stores found", zap.Float64("latitude", lat), zap.Float64("longitude", long))
		return nil, fmt.Errorf("no stores found for lat: %f, long: %f", lat, long)
	}
	return ids, nil
}

func (ss *StoreService) lookup(ctx context.Context, k uint32) *Store {
	v, ok := ss.stores[k]
	if !ok {
		return nil
	}
	return v
}
