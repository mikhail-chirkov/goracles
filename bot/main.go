package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("beep")
	})

	// TODO: config
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 7777), nil))
}
