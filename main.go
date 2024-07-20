package main

import (
	"log"
	"net/http"
)

func main() {
	const pathRoot = "."
	const port = "8080"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(pathRoot)))
	server := &http.Server{Handler: mux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
