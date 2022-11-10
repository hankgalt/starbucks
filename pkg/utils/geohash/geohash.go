package geohash

import (
	"fmt"

	"github.com/hankgalt/starbucks/pkg/errors"
)

type Point struct {
	Latitude  float64
	Longitude float64
}
type Range struct {
	Min float64
	Max float64
}
type Bounds struct {
	Latitude  Range
	Longitude Range
}

func Encode(lat float64, lon float64, percision int) (string, error) {
	var hash string
	var latMin float64 = -90
	var latMax float64 = 90
	var lonMin float64 = -180
	var lonMax float64 = 180
	quadMap := map[string]string{
		"00": "a",
		"01": "b",
		"11": "c",
		"10": "d",
	}

	if percision > 12 {
		percision = 12
	}

	for len(hash) < percision {
		var quad string
		var latMid float64 = (latMin + latMax) / 2
		if lat >= latMid {
			quad = fmt.Sprintf("%s%d", quad, 1)
			latMin = latMid
		} else {
			quad = fmt.Sprintf("%s%d", quad, 0)
			latMax = latMid
		}

		var lonMid float64 = (lonMin + lonMax) / 2
		if lon >= lonMid {
			quad = fmt.Sprintf("%s%d", quad, 1)
			lonMin = lonMid
		} else {
			quad = fmt.Sprintf("%s%d", quad, 0)
			lonMax = lonMid
		}
		hash = fmt.Sprintf("%s%s", hash, quadMap[quad])
	}

	return hash, nil
}

func Decode(hash string) (*Point, error) {
	boundaries, err := bounds(hash)
	if err != nil {
		return nil, errors.NewAppError(errors.ERROR_DECODING_BOUNDS)
	}

	var point Point
	point.Latitude = (boundaries.Latitude.Min + boundaries.Latitude.Max) / 2
	point.Longitude = (boundaries.Longitude.Min + boundaries.Longitude.Max) / 2

	return &point, nil
}

func bounds(hash string) (*Bounds, error) {
	var latMin float64 = -90
	var latMax float64 = 90
	var lonMin float64 = -180
	var lonMax float64 = 180
	quadMap := map[string]string{
		"a": "00",
		"b": "01",
		"c": "11",
		"d": "10",
	}

	for i := 0; i < len(hash); i++ {
		char := string(hash[i])
		quad := quadMap[char]
		latHash := quad[0]
		lonHash := quad[1]

		latMid := (latMin + latMax) / 2
		if latHash == '0' {
			latMax = latMid
		} else {
			latMin = latMid
		}

		lonMid := (lonMin + lonMax) / 2
		if lonHash == '0' {
			lonMax = lonMid
		} else {
			lonMin = lonMid
		}
	}

	var latRange, lonRange Range
	latRange.Min = latMin
	latRange.Max = latMax
	lonRange.Min = lonMin
	lonRange.Max = lonMax

	var bounds Bounds
	bounds.Latitude = latRange
	bounds.Longitude = lonRange

	return &bounds, nil
}
