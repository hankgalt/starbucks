package main

import (
	"log"

	"github.com/hankgalt/starbucks/pkg/config"
	"github.com/hankgalt/starbucks/pkg/listing"
	"github.com/hankgalt/starbucks/pkg/server"
)

func main() {
	config := config.GetConfig()
	gateway := listing.NewJasonGateway(config)
	gateway.ProcessFile()

	srv := server.NewHTTPServer(":8080", gateway)
	log.Println("main() - listening for store requests")
	log.Fatal(srv.ListenAndServe())
}
