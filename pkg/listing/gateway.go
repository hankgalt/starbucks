package listing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hankgalt/starbucks/pkg/constants"
	"github.com/hankgalt/starbucks/pkg/loader"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

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
	log     *log.Logger
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

func NewJasonGateway() *JsonGateway {
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

	jg := &JsonGateway{
		log:     log,
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
		jg.log.Debug("\033[34m ProcessFile() - finished processing stores \033[0m")
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
		jg.log.Debug("\033[34m ProcessFile() - gateway stats: \033[0m", stats)
	}()
}

func (jg *JsonGateway) GetStore(storeId uint32) (*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()

	s := jg.lookup(storeId)
	if s == nil {
		return nil, fmt.Errorf("store with storeID %d doesn't exist", storeId)
	}
	return s, nil
}

func (jg *JsonGateway) GetStoresForPostalCode(postalCode string, dist int) ([]*Store, error) {
	url := fmt.Sprintf("https://maps.google.com/maps/api/geocode/json?components=country:US|postal_code:%s&sensor=false&key=AIzaSyCcutJ7WAvncYD2vyyenPCF84Ycs5oDB9c", postalCode)

	r, err := http.Get(url)
	if err != nil {
		// fmt.Printf("GetStoresForPostalCode() - no stores found for postalCoode: %s, err: %v\n", postalCode, err)
		jg.log.Error("GetStoresForGeoPoint() - no stores found for postalCoode: %s, err: %v", postalCode, err)
		return nil, err
	}
	defer r.Body.Close()

	var results GeocoderResults
	err = json.NewDecoder(r.Body).Decode(&results)
	if err != nil {
		fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
		return nil, err
	}

	if strings.ToUpper(results.Status) != "OK" {
		// If the status is not "OK" check what status was returned
		switch strings.ToUpper(results.Status) {
		case "ZERO_RESULTS":
			err = errors.New("no results found")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "OVER_QUERY_LIMIT":
			err = errors.New("over quota request")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "REQUEST_DENIED":
			err = errors.New("request was denied")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "INVALID_REQUEST":
			err = errors.New("invalid request")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		case "UNKNOWN_ERROR":
			err = errors.New("server error, please, try again")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		default:
			err = errors.New("unknown error")
			fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s, err: %v\n", postalCode, err)
			return nil, err
		}
	}
	// fmt.Printf("GetStoresForPostalCode() - geocoding response: %v", results)
	lat, long := results.Results[0].Geometry.Location.Lat, results.Results[0].Geometry.Location.Lng
	fmt.Printf("GetStoresForPostalCode() - lat: %f, long: %f\n", lat, long)

	return jg.GetStoresForGeoPoint(lat, long, dist)

	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	fmt.Printf("GetStoresForPostalCode() - error reading response for postalCoode: %s\n", postalCode)
	// 	// jg.log.Error("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f", lat, long)
	// 	return nil, err
	// }
	// fmt.Printf("GetStoresForPostalCode() - geocoding response body : %s", body)
	// return nil, nil
}

func (jg *JsonGateway) GetStoresForGeoPoint(lat, long float64, dist int) ([]*Store, error) {
	jg.mu.RLock()
	defer jg.mu.RUnlock()
	fmt.Printf("GetStoresForGeoPoint() - lat: %f, long: %f, dist: %d\n", lat, long, dist)
	latKey := buildMapKey(lat)
	latStoreIDs, ok := jg.LatMap[latKey]
	if !ok {
		// fmt.Printf("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f\n", lat, long)
		jg.log.Error("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f", lat, long)
		return nil, fmt.Errorf("no stores found for lat: %f, long: %f", lat, long)
	}

	longKey := buildMapKey(long)
	longStoreIDs, ok := jg.LongMap[longKey]
	if !ok {
		// fmt.Printf("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f\n", lat, long)
		jg.log.Error("GetStoresForGeoPoint() - no stores found for lat: %f, long: %f", lat, long)
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
	fmt.Printf("GetStoresForGeoPoint() - found %d stores\n", len(ids))
	jg.log.Debug("GetStoresForGeoPoint() - found %d stores", len(ids))
	stores := []*Store{}
	// origin := haversine.Point{Lat: lat, Lon: long}
	origin := vincenty.LatLng{Latitude: lat, Longitude: long}
	for _, v := range ids {
		store, err := jg.GetStore(v)
		if err != nil {
			fmt.Printf("GetStoresForGeoPoint() - no stores found for id: %d\n", v)
			jg.log.Debug("GetStoresForGeoPoint() - no stores found for id: %d", v)
		}
		// jg.log.Debug("GetStoresForGeoPoint() - store id: %d, lat: %f, long: %f", store.ID, store.Latitude, store.Longitude)
		// pos := haversine.Point{Lat: store.Latitude, Lon: store.Longitude}
		pos := vincenty.LatLng{Latitude: store.Latitude, Longitude: store.Longitude}
		// d := haversine.Distance(origin, pos)
		d := vincenty.Inverse(origin, pos)
		// jg.log.Debug("GetStoresForGeoPoint() - distance: %f", d/1000)
		// fmt.Printf("GetStoresForGeoPoint() - distance: %f, dist: %d\n", d.Kilometers(), dist)
		// jg.log.Debug("GetStoresForGeoPoint() - distance: %f, dist: %f", d.Kilometers(), dist)
		// if float64(d) <= dist*1000 {
		if d.Kilometers() <= float64(dist) {
			stores = append(stores, store)
		}
	}
	fmt.Printf("GetStoresForGeoPoint() - returning %d stores\n", len(stores))
	jg.log.Debug("GetStoresForGeoPoint() - returning %d stores", len(stores))
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
	jg.log.Debug("\033[32m readFile() - start processing stores \033[0m")

	fileName := ctx.Value(constants.FileNameContextKey).(string)
	resultStream, err := loader.ReadFileArray(ctx, cancel, fileName)
	if err != nil {
		jg.log.Debug("\033[32m readFile() - error: %v# \033[0m", err)
		cancel()
	}
	count := 0

	for {
		select {
		case <-ctx.Done():
			jg.log.Debug("\033[32m readFile() - context done \033[0m")
			return
		case r, ok := <-resultStream:
			if !ok {
				jg.log.Debug("\033[32m readFile() - result stream closed \033[0m")
				wgp.Done()
				close(out)
				return
			}
			wgs.Add(1)
			count++
			if count%1000 == 0 {
				jg.log.Debug("\033[32m readFile() - publishing %d %v \033[0m", count, r)
			}
			jg.publishStore(r, out)
		}
	}
}

func (jg *JsonGateway) publishStore(r map[string]interface{}, out chan *Store) {
	store, err := mapResultToStore(r)
	if err != nil {
		jg.log.WithError(err).Error("\033[36m publishStore() - error publidhing store \033[0m")
	} else {
		// log.Println("\033[36m publishStore() - publishing \033[0m", store)
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
	jg.log.Debug("\033[33m processStore() - start reading stores  \033[0m")
	count := 0

	for {
		select {
		case <-ctx.Done():
			jg.log.Debug("\033[33m processStore() - context done \033[0m")
			return
		case store, ok := <-out:
			if !ok {
				jg.log.Debug("\033[33m processStore() - store notification channel closed \033[0m")
				wgp.Done()
				return
			}

			if count%1000 == 0 {
				jg.log.Debug("\033[33m  processStore() - received store notification %d %v \033[0m", count, store)
			}
			success := jg.updateDataStores(store)
			if !success {
				jg.log.Debug("\033[33m processStore() - error adding store %d %v \033[0m", count, store)
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

func roundTo(n float64, decimals uint32) float64 {
	return math.Round(n*math.Pow(10, math.Pow(10, float64(decimals)))) / math.Pow(10, math.Pow(10, float64(decimals)))
}
