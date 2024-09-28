package main

import (
	"fmt"
	"log"
)

func main() {
	store, err := NewPostGresStore()
	if err != nil {
		fmt.Print("Error connecting to database")
	}

	if err := store.init(); err != nil {
		log.Fatal("Error initializing database")
	}

	server := newApiServer(":8080", store)
	server.run()
}
