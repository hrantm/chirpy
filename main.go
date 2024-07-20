package main

import (
	"encoding/json"
	"fmt"
	"hrantm/chirpy/db"
	"log"
	"net/http"
	"strings"
	"text/template"
)

type apiConfig struct {
	fileserverHits int
}

type App struct {
	DB *db.DB
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func main() {
	const pathRoot = "."
	const port = "8080"
	const dbPath = "/Users/hrant/chirpy/database.json"

	db, err := db.NewDB(dbPath)

	if err != nil {
		fmt.Println("FAILED TO INITIALIZE")
	}
	app := &App{DB: db}

	mux := http.NewServeMux()

	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(pathRoot)))
	apiCfg := &apiConfig{}
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(appHandler))

	mux.HandleFunc("GET /api/healthz", handleHealthz)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("GET /api/reset", apiCfg.handleReset)

	mux.HandleFunc("POST /api/chirps", app.handlePostChirp)
	mux.HandleFunc("GET /api/chirps", app.handleGetChirps)

	server := &http.Server{Handler: mux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (a *App) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := a.DB.GetChirps()
	if err != nil {
		log.Printf("Error getting Chirps %s:", err)
		w.WriteHeader(500)
		return
	}

	// type returnVals struct {
	// 	Id   int    `json:"id"`
	// 	Body string `json:"body"`
	// }

	data, err := json.Marshal(chirps)

	if err != nil {
		log.Printf("Error marshalling data %s:", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)

}

func (a *App) handlePostChirp(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		log.Printf("Error decoding parameters %s:", err)
		w.WriteHeader(500)
		return
	}
	type returnVals struct {
		Id          int    `json:"id"`
		Valid       bool   `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
		Body        string `json:"body"`
	}
	respBody := returnVals{
		Valid: true,
	}
	statusCode := 201
	if len(params.Body) > 140 {
		respBody.Valid = false
		statusCode = 400
	}
	newBody := []string{}
	profanes := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}
	for _, val := range strings.Split(params.Body, " ") {
		_, ok := profanes[strings.ToLower(val)]
		if ok {
			newBody = append(newBody, "****")
		} else {
			newBody = append(newBody, val)
		}
	}
	respBody.Body = params.Body
	respBody.CleanedBody = strings.Join(newBody, " ")

	chirp, err := a.DB.CreateChirp(respBody.CleanedBody)
	if err != nil {
		log.Printf("Error creating chirp %s:", err)
		w.WriteHeader(500)
		return
	}
	respBody.Id = chirp.Id

	data, err := json.Marshal(respBody)

	if err != nil {
		log.Printf("Error marshalling json %s:", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)

}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	const htmlTemplate = `
	<!DOCTYPE html>
	<html>

	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited {{.Hits}} times!</p>
	</body>
	
	</html>`

	tmpl := template.Must(template.New("metrics").Parse(htmlTemplate))
	data := struct {
		Hits int
	}{
		Hits: cfg.fileserverHits,
	}
	tmpl.Execute(w, data)

}

func (cfg *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Counter reset"))
}
