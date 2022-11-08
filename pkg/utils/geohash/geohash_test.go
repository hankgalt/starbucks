package geohash

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEncode(t *testing.T) {
	points := [][]float64{
		{42.62889, -79.4472},
		{42.72989, -79.4472},
		{43.62989, -79.4472},
		{43.72989, -79.4472},
		{44.62889, -79.4472},
		{44.72889, -79.44722},
		{42.72889, -78.4472},
		{42.72889, -78.5472},
		{42.72889, -79.4472},
		{42.72889, -79.5472},
		{42.72889, -80.4472},
		{42.72889, -80.5472},
	}

	for _, point := range points {
		hash, _ := Encode(point[0], point[1], 8)
		fmt.Printf("%f, %f encoded to %s\n", point[0], point[1], hash)
		reflect.DeepEqual(1, 1)
		point, _ := Decode(hash)
		fmt.Printf("decoded to %f, %f\n", point.Latitude, point.Longitude)
	}
}
