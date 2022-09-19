package listing

import (
	"encoding/json"
	"log"
	"time"

	"github.com/hankgalt/starbucks/pkg/errors"
)

// Store defines the properties of a store to be listed
type Store struct {
	Id        uint32    `json:"store_id"`
	Name      string    `json:"name"`
	Longitude float64   `json:"longitude"`
	Latitude  float64   `json:"latitude"`
	City      string    `json:"city"`
	Country   string    `json:"country"`
	Created   time.Time `json:"created"`
}

func mapResultToStore(r map[string]interface{}) (*Store, error) {
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

type GeocoderResults struct {
	Results []Result `json:"results"`
	Status  string   `json:"status"`
}

type Result struct {
	AddressComponents []Address `json:"address_components"`
	FormattedAddress  string    `json:"formatted_address"`
	Geometry          Geometry  `json:"geometry"`
	PlaceId           string    `json:"place_id"`
	Types             []string  `json:"types"`
}

// Address store each address is identified by the 'types'
type Address struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// Geometry store each value in the geometry
type Geometry struct {
	Bounds       Bounds `json:"bounds"`
	Location     LatLng `json:"location"`
	LocationType string `json:"location_type"`
	Viewport     Bounds `json:"viewport"`
}

// Bounds Northeast and Southwest
type Bounds struct {
	Northeast LatLng `json:"northeast"`
	Southwest LatLng `json:"southwest"`
}

// LatLng store the latitude and longitude
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
