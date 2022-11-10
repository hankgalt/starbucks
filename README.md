<p align="center"><img src="https://i.imgur.com/gODVANP.png" border="0" /></p>

# starbucks

All [Starbucks](https://www.starbucks.com/) locations in the world.

## Attribution

Dataset is a lightly processed version of a [Socrata
dataset](https://opendata.socrata.com/Business/All-Starbucks-Locations-in-the-World/xy4y-c4mk)
scraped by Chris Meller. Underlying data is owned by
[Starbucks](https://www.starbucks.com/).

## development
- update `config.json` with valid google maps api key
- `cd cmd/store-server`
- `go run starbucks.go`
- In another shell, `curl -k -X POST localhost:8080/search -d '{"postalCode": "92612", "distance": 5}'`
- `ctrl + C` to stop the server

- update grpc api
- `protoc api/v1/*.proto --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative --proto_path=.`