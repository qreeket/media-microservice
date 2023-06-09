package main

import (
	"github.com/joho/godotenv"
	"github.com/qcodelabsllc/qreeket/media/network"
	"log"
)

func main() {
	// This line loads the environment variables from the ".env" file.
	if err := godotenv.Load(); err != nil {
		log.Fatalf("unable to load environment variables: %+v\n", err)
	}

	// This line initializes the grpc server
	network.InitServer()
}
