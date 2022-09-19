<p align="center"><img src="https://i.imgur.com/gODVANP.png" border="0" /></p>

# starbucks

All [Starbucks](https://www.starbucks.com/) locations in the world.

## Attribution

Dataset is a lightly processed version of a [Socrata
dataset](https://opendata.socrata.com/Business/All-Starbucks-Locations-in-the-World/xy4y-c4mk)
scraped by Chris Meller. Underlying data is owned by
[Starbucks](https://www.starbucks.com/).

## development
- cd cmd/store-server
- go run starbucks.go
- curl -X POST localhost:8080/search -d '{"postalCode": "92612", "distance": 5}'
- cntrl + C to stop the server
