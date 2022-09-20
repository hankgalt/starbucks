package listing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/loader"

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

func NewJasonGateway(config *config.Configuration) *JsonGateway {
	jg := &JsonGateway{
		config:  config,
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
		log.Println("ProcessFile() - finished processing stores")
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
		log.Println("ProcessFile() - gateway stats: ", stats)
	}()
}

func (jg *JsonGateway) GetStore(storeId uint32) (*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()

	s := jg.lookup(storeId)
	if s == nil {
		log.Printf("GetStore() - store with storeId %d doesn't exist", storeId)
		return nil, fmt.Errorf("store with storeId %d doesn't exist", storeId)
	}
	return s, nil
}

func (jg *JsonGateway) GetStoresForPostalCode(postalCode string, dist int) ([]*Store, error) {
	url := fmt.Sprintf("https://maps.google.com/maps/api/geocode/json?components=country:US|postal_code:%s&sensor=false&key=%s", postalCode, jg.config.GEOCODER_API_KEY)

	r, err := http.Get(url)
	if err != nil {
		log.Printf("GetStoresForGeoPoint() - no stores found for postalCoode: %s, err: %v", postalCode, err)
		return nil, err
	}
	defer r.Body.Close()

	var results GeocoderResults
	err = json.NewDecoder(r.Body).Decode(&results)
	if err != nil {
		log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
		return nil, err
	}

	if strings.ToUpper(results.Status) != "OK" {
		// If the status is not "OK" check what status was returned
		switch strings.ToUpper(results.Status) {
		case "ZERO_RESULTS":
			err = errors.New("no results found")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "OVER_QUERY_LIMIT":
			err = errors.New("over quota request")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "REQUEST_DENIED":
			err = errors.New("request was denied")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "INVALID_REQUEST":
			err = errors.New("invalid request")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "UNKNOWN_ERROR":
			err = errors.New("server error, please, try again")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		default:
			err = errors.New("unknown error")
			log.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		}
	}
	// log.Printf("GetStoresForPostalCode() - geocoding response: %v\n", results)
	lat, long := results.Results[0].Geometry.Location.Lat, results.Results[0].Geometry.Location.Lng
	log.Printf("GetStoresForPostalCode() - lat: %f, long: %f\n", lat, long)

	return jg.GetStoresForGeoPoint(lat, long, dist)
}

func (jg *JsonGateway) GetStoresForGeoPoint(lat, long float64, dist int) ([]*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()
	log.Printf("GetStoresForGeoPoint() - lat: %f, long: %f, dist: %d\n", lat, long, dist)
	latKey := buildMapKey(lat)
	latStoreIDs, ok := jg.LatMap[latKey]
	if !ok {
		log.Printf("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f", lat, long)
		return nil, fmt.Errorf("no stores found for lat: %f, long: %f", lat, long)
	}

	longKey := buildMapKey(long)
	longStoreIDs, ok := jg.LongMap[longKey]
	if !ok {
		log.Printf("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f", lat, long)
		return nil, fmt.Errorf("no stores found for lat: %f, long: %f", lat, long)
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
	log.Printf("GetStoresForGeoPoint() - found %d stores\n", len(ids))
	stores := []*Store{}
	// origin := haversine.Point{Lat: lat, Lon: long}
	origin := vincenty.LatLng{Latitude: lat, Longitude: long}
	for _, v := range ids {
		store, err := jg.GetStore(v)
		if err != nil {
			log.Printf("GetStoresForGeoPoint() - no store found for id: %d\n", v)
		}
		// pos := haversine.Point{Lat: store.Latitude, Lon: store.Longitude}
		pos := vincenty.LatLng{Latitude: store.Latitude, Longitude: store.Longitude}
		// d := haversine.Distance(origin, pos)
		d := vincenty.Inverse(origin, pos)
		// log.Printf("GetStoresForGeoPoint() - distance: %f, dist: %d\n", d.Kilometers(), dist)
		// if float64(d) <= dist*1000 {
		if d.Kilometers() <= float64(dist) {
			stores = append(stores, store)
		}
	}
	log.Printf("GetStoresForGeoPoint() - returning %d stores\n", len(stores))
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
	log.Println("readFile() - start processing stores")
	fileName := ctx.Value(constants.FileNameContextKey).(string)
	resultStream, err := loader.ReadFileArray(ctx, cancel, fileName)
	if err != nil {
		log.Printf("readFile() - error: %v# \n", err)
		cancel()
	}
	count := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("readFile() - context done")
			return
		case r, ok := <-resultStream:
			if !ok {
				log.Println("readFile() - result stream closed")
				wgp.Done()
				close(out)
				return
			}
			wgs.Add(1)
			count++
			// if count%1000 == 0 {
			// 	log.Printf("readFile() - publishing %d %v\n", count, r)
			// }
			jg.publishStore(r, out)
		}
	}
}

func (jg *JsonGateway) publishStore(r map[string]interface{}, out chan *Store) {
	store, err := mapResultToStore(r)
	if err != nil {
		log.Printf("publishStore() - error publishing store, error: %v\n", err)
	} else {
		// log.Println("publishStore() - publishing: ", store)
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
	log.Println("processStore() - start reading stores")
	count := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("processStore() - context done")
			return
		case store, ok := <-out:
			if !ok {
				log.Println("processStore() - store notification channel closed")
				wgp.Done()
				return
			}
			// if count%1000 == 0 {
			// 	log.Printf("processStore() - received store notification %d %v\n", count, store)
			// }
			success := jg.updateDataStores(store)
			if !success {
				log.Printf("processStore() - error adding store %d %v\n", count, store)
			}
			count++
			wgs.Done()
		}
	}
}

func (jg *JsonGateway) updateDataStores(s *Store) bool {
	// log.Println("\033[33m updateDataStores() - updating data stores for new store:  \033[0m", s)
	// log.Println()
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
	// log.Println("buildMapKey() - map key: ", k)
	return k
}
