package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server{Handler: mux, Addr: ":8080"}
	fmt.Println("Starting Server on port 8080")
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error Starting Server", err)
	}
}
