package store

import (
	"context"
	"encoding/json"
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
	ss := &StoreService{
		logger:  logger,
		stores:  map[uint32]*Store{},
		HashMap: map[string][]uint32{},
		count:   0,
		ready:   false,
	}

	return ss
}

func (ss *StoreService) SetReady(ctx context.Context, ready bool) {
	ss.ready = ready
}

func (ss *StoreService) AddStore(ctx context.Context, s *Store) (bool, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.lookup(s.Id) == nil {
		hashKey, err := geohash.Encode(s.Latitude, s.Longitude, 8)
		if err != nil {
			ss.logger.Error(errors.ERROR_ENCODING_LAT_LONG, zap.Float64("latitude", s.Latitude), zap.Float64("longitude", s.Longitude))
			return false, errors.WrapError(err, errors.ERROR_ENCODING_LAT_LONG)
		}
		hashStoreIDs, ok := ss.HashMap[hashKey]
		if !ok {
			hashStoreIDs = []uint32{}
		}
		hashStoreIDs = append(hashStoreIDs, s.Id)
		ss.HashMap[hashKey] = hashStoreIDs

		ss.stores[s.Id] = s
		ss.count++

		return true, nil
	}
	ss.logger.Error(errors.ERROR_STORE_ID_ALREADY_EXISTS, zap.Float64("storeId", float64(s.Id)))
	return false, errors.NewAppError(errors.ERROR_STORE_ID_ALREADY_EXISTS)
}

func (ss *StoreService) GetStore(ctx context.Context, storeId uint32) (*Store, error) {
	s := ss.lookup(storeId)
	if s == nil {
		ss.logger.Error(errors.ERROR_NO_STORE_FOUND_FOR_ID, zap.Int("storeId", int(storeId)))
		return nil, errors.WrapError(errors.ErrNotFound, errors.ERROR_NO_STORE_FOUND_FOR_ID)
	}
	return s, nil
}

func (ss *StoreService) GetStoresForGeoPoint(ctx context.Context, lat, long float64, dist int) ([]*Store, error) {
	ss.logger.Debug("getting stores for geopoint", zap.Float64("latitude", lat), zap.Float64("longitude", long), zap.Int("distance", dist))
	ids, err := ss.getStoreIdsForLatLong(ctx, lat, long)
	if err != nil {
		return nil, err
	}
	ss.logger.Debug("found stores", zap.Int("numOfStores", len(ids)), zap.Float64("latitude", lat), zap.Float64("longitude", long))

	stores := []*Store{}
	// origin := haversine.Point{Lat: lat, Lon: long}
	origin := vincenty.LatLng{Latitude: lat, Longitude: long}
	for _, v := range ids {
		store, err := ss.GetStore(ctx, v)
		if err != nil {
			ss.logger.Error(errors.ERROR_NO_STORE_FOUND_FOR_ID, zap.Int("storeId", int(v)))
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

func (ss *StoreService) GetStoreStats() StoreStats {
	return StoreStats{
		Ready:     ss.ready,
		Count:     ss.count,
		HashCount: len(ss.HashMap),
	}
}

func (ss *StoreService) Clear() {
	ss.count = 0
	ss.stores = map[uint32]*Store{}
	ss.HashMap = map[string][]uint32{}
	ss.ready = false
}

func MapResultToStore(r map[string]interface{}) (*Store, error) {
	storeJson, err := json.Marshal(r)
	if err != nil {
		return nil, errors.WrapError(err, errors.ERROR_MARSHALLING_RESULT)
	}

	var s Store
	err = json.Unmarshal(storeJson, &s)
	if err != nil {
		return nil, errors.WrapError(err, errors.ERROR_UNMARSHALLING_STORE_JSON)
	}
	return &s, nil
}

func (ss *StoreService) getStoreIdsForLatLong(ctx context.Context, lat, long float64) ([]uint32, error) {
	hashKey, err := geohash.Encode(lat, long, 8)
	if err != nil {
		ss.logger.Error(errors.ERROR_ENCODING_LAT_LONG, zap.Float64("latitude", lat), zap.Float64("longitude", long))
		errors.WrapError(err, errors.ERROR_ENCODING_LAT_LONG)
		return nil, err
	}
	ss.logger.Debug("created hash key", zap.String("hashKey", hashKey), zap.Float64("latitude", lat), zap.Float64("longitude", long))
	ids, ok := ss.HashMap[hashKey]
	if !ok || len(ids) < 1 {
		ss.logger.Error(errors.ERROR_NO_STORE_FOUND, zap.Float64("latitude", lat), zap.Float64("longitude", long))
		return nil, errors.WrapError(err, errors.ERROR_NO_STORE_FOUND)
	}
	return ids, nil
}

func (ss *StoreService) lookup(k uint32) *Store {
	v, ok := ss.stores[k]
	if !ok {
		return nil
	}
	return v
}
