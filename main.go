package main

import (
	"log"
	"net/http"
)

func main() {
	const pathRoot = "."
	const port = "8080"
	mux := http.NewServeMux()
	mux.Handle("/app/*", http.StripPrefix("/app", http.FileServer(http.Dir(pathRoot))))
	mux.HandleFunc("/healthz", handleHealthEndpoint)
	server := &http.Server{Handler: mux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func handleHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
