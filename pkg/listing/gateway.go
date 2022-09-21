package listing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/loader"
	"go.uber.org/zap"

	// "github.com/paultag/go-haversine"
	"gitlab.com/xerra/common/vincenty"
)

type Gateway interface {
	ProcessFile()
	GetStore(storeId uint32) (*Store, error)
	GetStoresForGeoPoint(lat, long, dist float64) ([]*Store, error)
	GetStoreStats() GatewayStats
}

type JsonGateway struct {
	mu      sync.RWMutex
	logger  *zap.Logger
	config  *config.Configuration
	stores  map[uint32]*Store
	LatMap  map[string][]uint32
	LongMap map[string][]uint32
	count   int
	ready   bool
}

type GatewayStats struct {
	Count     int
	LatCount  int
	LongCount int
	Ready     bool
}

func NewJasonGateway(config *config.Configuration, logger *zap.Logger) *JsonGateway {
	jg := &JsonGateway{
		config:  config,
		logger:  logger,
		stores:  map[uint32]*Store{},
		LatMap:  map[string][]uint32{},
		LongMap: map[string][]uint32{},
		count:   0,
		ready:   false,
	}

	return jg
}

func (jg *JsonGateway) ProcessFile() {
	defer func() {
		jg.logger.Info("finished setting up store data")
	}()

	ctx := context.WithValue(context.Background(), constants.FileNameContextKey, "locations.json")
	ctx = context.WithValue(ctx, constants.ReadRateContextKey, 2)
	ctx, cancel := context.WithCancel(ctx)

	cout := make(chan *Store)
	var wgp sync.WaitGroup
	var wgs sync.WaitGroup

	wgp.Add(1)
	go jg.readFile(ctx, cancel, &wgp, &wgs, cout)
	wgp.Add(1)
	go jg.processStore(ctx, cancel, &wgp, &wgs, cout)

	func() {
		wgp.Wait()
		jg.ready = true
		stats := jg.GetStoreStats()
		jg.logger.Info("gateway status", zap.Any("stats", stats))
	}()
}

func (jg *JsonGateway) GetStore(storeId uint32) (*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()

	s := jg.lookup(storeId)
	if s == nil {
		jg.logger.Error("store doesn't exist", zap.Int("storeId", int(storeId)))
		return nil, fmt.Errorf("store with storeId %d doesn't exist", storeId)
	}
	return s, nil
}

func (jg *JsonGateway) GetStoresForPostalCode(postalCode string, dist int) ([]*Store, error) {
	url := fmt.Sprintf("https://maps.google.com/maps/api/geocode/json?components=country:US|postal_code:%s&sensor=false&key=%s", postalCode, jg.config.GEOCODER_API_KEY)

	r, err := http.Get(url)
	if err != nil {
		jg.logger.Error("geocoder request error", zap.Error(err), zap.String("postalCode", postalCode))
		return nil, err
	}
	defer r.Body.Close()

	var results GeocoderResults
	err = json.NewDecoder(r.Body).Decode(&results)
	if err != nil {
		jg.logger.Error("error decoding geocode response", zap.Error(err), zap.String("postalCode", postalCode))
		return nil, err
	}

	if strings.ToUpper(results.Status) != "OK" {
		// If the status is not "OK" check what status was returned
		switch strings.ToUpper(results.Status) {
		case "ZERO_RESULTS":
			err = errors.New("no results found")
		case "OVER_QUERY_LIMIT":
			err = errors.New("over quota request")
		case "REQUEST_DENIED":
			err = errors.New("request was denied")
		case "INVALID_REQUEST":
			err = errors.New("invalid request")
		case "UNKNOWN_ERROR":
			err = errors.New("server error, please, try again")
		default:
			err = errors.New("unknown error")
		}
		jg.logger.Error("geocode response error", zap.Error(err), zap.String("postalCode", postalCode))
		return nil, err
	}
	lat, long := results.Results[0].Geometry.Location.Lat, results.Results[0].Geometry.Location.Lng
	jg.logger.Debug("geocoder geopoint response", zap.Float64("latitude", lat), zap.Float64("longitude", long))

	return jg.GetStoresForGeoPoint(lat, long, dist)
}

func (jg *JsonGateway) GetStoresForGeoPoint(lat, long float64, dist int) ([]*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()

	jg.logger.Debug("getting stores for geopoint", zap.Float64("latitude", lat), zap.Float64("longitude", long), zap.Int("distance", dist))
	latKey := buildMapKey(lat)
	latStoreIDs, ok := jg.LatMap[latKey]
	if !ok {
		jg.logger.Error("no stores found for latitude", zap.Float64("latitude", lat))
		return nil, fmt.Errorf("no stores found for lat: %f", lat)
	}

	longKey := buildMapKey(long)
	longStoreIDs, ok := jg.LongMap[longKey]
	if !ok {
		jg.logger.Error("no stores found for longitude", zap.Float64("longitude", long))
		return nil, fmt.Errorf("no stores found for long: %f", long)
	}

	m := make(map[uint32]bool)
	ids := []uint32{}
	for _, item := range latStoreIDs {
		m[item] = true
		ids = append(ids, item)
	}

	for _, item := range longStoreIDs {
		if _, ok := m[item]; !ok {
			ids = append(ids, item)
		}
	}
	jg.logger.Debug("found stores", zap.Int("numOfStores", len(ids)), zap.Float64("latitude", lat), zap.Float64("longitude", long))
	stores := []*Store{}
	// origin := haversine.Point{Lat: lat, Lon: long}
	origin := vincenty.LatLng{Latitude: lat, Longitude: long}
	for _, v := range ids {
		store, err := jg.GetStore(v)
		if err != nil {
			jg.logger.Error("no stores found for id", zap.Int("storeId", int(v)))
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
	jg.logger.Debug("returning stores", zap.Int("numOfStores", len(stores)), zap.Float64("latitude", lat), zap.Float64("longitude", long), zap.Int("distance", dist))
	return stores, nil
}

func (jg *JsonGateway) GetStoreStats() GatewayStats {
	jg.mu.RLock()
	defer jg.mu.RUnlock()

	return GatewayStats{
		Ready:     jg.ready,
		Count:     jg.count,
		LatCount:  len(jg.LatMap),
		LongCount: len(jg.LongMap),
	}
}

func (jg *JsonGateway) readFile(
	ctx context.Context,
	cancel func(),
	wgp *sync.WaitGroup,
	wgs *sync.WaitGroup,
	out chan *Store,
) {
	jg.logger.Info("start reading store data file")
	fileName := ctx.Value(constants.FileNameContextKey).(string)
	resultStream, err := loader.ReadFileArray(ctx, cancel, fileName)
	if err != nil {
		jg.logger.Error("error reading store data file", zap.Error(err))
		cancel()
	}
	count := 0

	for {
		select {
		case <-ctx.Done():
			jg.logger.Info("store data file read context done")
			return
		case r, ok := <-resultStream:
			if !ok {
				jg.logger.Info("store data file result stream closed")
				wgp.Done()
				close(out)
				return
			}
			wgs.Add(1)
			count++
			// if count%1000 == 0 {
			// 	jg.logger.Debug("publishing store", zap.Any("storeJson", r), zap.Int("storeCount", count))
			// }
			jg.publishStore(r, out)
		}
	}
}

func (jg *JsonGateway) publishStore(r map[string]interface{}, out chan *Store) {
	store, err := mapResultToStore(r)
	if err != nil {
		jg.logger.Error("error processing store data", zap.Error(err), zap.Any("storeJson", r))
	} else {
		// jg.logger.Debug("publishing store", zap.Any("store", store))
		out <- store
	}
}

func (jg *JsonGateway) processStore(
	ctx context.Context,
	cancel func(),
	wgp *sync.WaitGroup,
	wgs *sync.WaitGroup,
	out chan *Store,
) {
	jg.logger.Info("start updating store data")
	count := 0

	for {
		select {
		case <-ctx.Done():
			jg.logger.Info("store data update context done")
			return
		case store, ok := <-out:
			if !ok {
				jg.logger.Info("store notification channel closed")
				wgp.Done()
				return
			}
			// if count%1000 == 0 {
			// 	 jg.logger.Debug("processing store", zap.Any("store", store), zap.Int("storeCount", count))
			// }
			success := jg.updateDataStores(store)
			if !success {
				jg.logger.Error("error processing store data", zap.Any("store", store), zap.Int("storeCount", count))
			}
			count++
			wgs.Done()
		}
	}
}

func (jg *JsonGateway) updateDataStores(s *Store) bool {
	jg.mu.Lock()
	defer jg.mu.Unlock()

	if jg.lookup(s.Id) == nil {
		latKey := buildMapKey(s.Latitude)
		latStoreIDs, ok := jg.LatMap[latKey]
		if !ok {
			latStoreIDs = []uint32{}
		}
		latStoreIDs = append(latStoreIDs, s.Id)

		longKey := buildMapKey(s.Longitude)
		longStoreIDs, ok := jg.LongMap[longKey]
		if !ok {
			longStoreIDs = []uint32{}
		}
		longStoreIDs = append(longStoreIDs, s.Id)

		jg.LatMap[latKey] = latStoreIDs
		jg.LongMap[longKey] = longStoreIDs
		jg.stores[s.Id] = s
		jg.count++

		return true
	}
	return false
}

func (jg *JsonGateway) lookup(k uint32) *Store {
	v, ok := jg.stores[k]
	if !ok {
		return nil
	}
	return v
}

func buildMapKey(v float64) string {
	le := math.Floor(v*10) / 10
	te := le + 0.1
	k := fmt.Sprintf("%.1f-%.1f", le, te)
	// fmt.Println("buildMapKey() - map key: ", k)
	return k
}
