package main

import "github.com/hankgalt/starbucks/pkg/services/store"

var DefaultStores = []store.Store{
	{
		City:      "Hong Kong",
		Name:      "Plaza Hollywood",
		Country:   "CN",
		Longitude: 114.20169067382812,
		Latitude:  22.340700149536133,
		Id:        1,
	},
	{
		City:      "Hong Kong",
		Name:      "Exchange Square",
		Country:   "CN",
		Longitude: 114.15818786621094,
		Latitude:  22.283939361572266,
		Id:        6,
	},
	{
		City:      "Kowloon",
		Name:      "Telford Plaza",
		Country:   "CN",
		Longitude: 114.21343994140625,
		Latitude:  22.3228702545166,
		Id:        8,
	},
	{
		City:      "Hong Kong",
		Name:      "Hong Kong Station",
		Country:   "CN",
		Longitude: 114.15846252441406,
		Latitude:  22.28445053100586,
		Id:        13,
	},
	{
		City:      "Hong Kong",
		Name:      "Pacific Place, Central",
		Country:   "CN",
		Longitude: 114.16461944580078,
		Latitude:  22.27765655517578,
		Id:        17,
	},
}
