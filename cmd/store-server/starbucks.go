package main

import (
	"fmt"
	"log"

	"github.com/hankgalt/starbucks/pkg/listing"
	"github.com/hankgalt/starbucks/pkg/server"
)

func main() {
	// log := app.SetupLogger()

	gateway := listing.NewJasonGateway()
	gateway.ProcessFile()
	stats := gateway.GetStoreStats()
	fmt.Println("main() - gateway stats: ", stats)
	fmt.Println("main() - finished processing stores")
	// log.Println("main() - stores latMap : ", gateway.LatMap)
	// log.Println("main() - stores longMap : ", gateway.LongMap)
	// stores, _ := gateway.GetStoresForGeoPoint(42.835, -71.409, 10)
	// fmt.Printf("main() - found %d stores for 42.835/-71.409", len(stores))
	// fmt.Println("main() - stores: ", stores)

	srv := server.NewHTTPServer(":8080", gateway)
	log.Fatal(srv.ListenAndServe())

	// app, err := app.New(log)
	// if err != nil {
	// 	log.WithError(err).Error("error creating store locator app")
	// }

	// if err := app.Start(); err != nil {
	// 	log.WithError(err).Error("main() - error starting store locator app")
	// }
	// log.Debug("main() - started store locator app")
	// gracefulShutdown(app)
	// log.Debug("main() - started store locator app")
	// log.Exit(0)
}

// func gracefulShutdown(app *app.App) {
// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, os.Interrupt)
// 	<-stop
// 	app.Stop()
// }
