package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/errors"
	"github.com/hankgalt/starbucks/pkg/utils/geohash"
	"go.uber.org/zap"
	"googlemaps.github.io/maps"
)

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

type GeoCodeService struct {
	host   string
	path   string
	client *maps.Client
	config *config.Configuration
	logger *zap.Logger
}

func New(config *config.Configuration, logger *zap.Logger) (*GeoCodeService, error) {
	c, err := maps.NewClient(maps.WithAPIKey(config.GEOCODER_API_KEY))
	if err != nil {
		logger.Error("error initializing maps client", zap.Error(err))
		return nil, err
	}

	return &GeoCodeService{
		host:   "https://maps.googleapis.com",
		path:   "/maps/api/geocode/json",
		client: c,
		config: config,
		logger: logger,
	}, nil
}

func (g *GeoCodeService) Geocode(ctx context.Context, postalCode, countryCode string) (*geohash.Point, error) {
	if ctx == nil {
		g.logger.Error("context is nil", zap.Error(errors.ErrNilContext))
		return nil, errors.ErrNilContext
	}
	if countryCode == "" {
		countryCode = "US"
	}

	url := g.postalCodeURL(countryCode, postalCode)
	r, err := http.Get(url)
	if err != nil {
		g.logger.Error("geocoder request error", zap.Error(err), zap.String("postalCode", postalCode))
		return nil, err
	}
	defer r.Body.Close()

	var results GeocoderResults
	err = json.NewDecoder(r.Body).Decode(&results)
	if err != nil || &results == (*GeocoderResults)(nil) || len(results.Results) < 1 {
		g.logger.Error(errors.ERROR_GEOCODING_POSTALCODE, zap.Error(err), zap.String("postalCode", postalCode))
		return nil, errors.NewAppError(errors.ERROR_GEOCODING_POSTALCODE)
	}
	lat, long := results.Results[0].Geometry.Location.Lat, results.Results[0].Geometry.Location.Lng

	// req := &maps.GeocodingRequest{
	// 	Components: map[maps.Component]string{
	// 		maps.ComponentCountry:    fmt.Sprintf("country:%s", countryCode),
	// 		maps.ComponentPostalCode: fmt.Sprintf("postal_code:%s", postalCode),
	// 	},
	// }

	// results, err := g.client.Geocode(ctx, req)
	// if err != nil || len(results) < 1 {
	// 	g.logger.Error(errors.ERROR_GEOCODING_POSTALCODE, zap.Error(err), zap.String("country", countryCode), zap.String("postalcode", postalCode))
	// 	return nil, errors.NewAppError(errors.ERROR_GEOCODING_POSTALCODE)
	// }
	// lat, long := results[0].Geometry.Location.Lat, results[0].Geometry.Location.Lng

	return &geohash.Point{
		Latitude:  lat,
		Longitude: long,
	}, nil
}

func (g *GeoCodeService) postalCodeURL(countryCode, postalCode string) string {
	return fmt.Sprintf("%s%s?components=country:%s|postal_code:%s&sensor=false&key=%s", g.host, g.path, countryCode, postalCode, g.config.GEOCODER_API_KEY)
}
