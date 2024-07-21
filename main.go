package main

import (
	"encoding/json"
	"fmt"
	"hrantm/chirpy/db"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type apiConfig struct {
	fileserverHits int
	DB             *db.DB
	JWTSecret      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	const pathRoot = "."
	const port = "8080"
	const dbPath = "/Users/hrant/chirpy/database.json"

	db, err := db.NewDB(dbPath)

	if err != nil {
		fmt.Println("FAILED TO INITIALIZE")
	}

	mux := http.NewServeMux()

	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(pathRoot)))
	apiCfg := &apiConfig{
		DB:             db,
		fileserverHits: 0,
		JWTSecret:      os.Getenv("JWT_SECRET"),
	}
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(appHandler))

	mux.HandleFunc("GET /api/healthz", handleHealthz)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handleMetrics)
	mux.HandleFunc("GET /api/reset", apiCfg.handleReset)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlePostChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handleGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpid}", apiCfg.handleGetChirpById)

	mux.HandleFunc("POST /api/users", apiCfg.handlePostUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlePutUser)
	mux.HandleFunc("POST /api/login", apiCfg.handlePostLogin)

	server := &http.Server{Handler: mux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (a *apiConfig) handlePutUser(w http.ResponseWriter, r *http.Request) {
	type Claims struct {
		jwt.RegisteredClaims
	}
	tokenString := strings.Split(r.Header.Get("Authorization"), " ")[1]
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.JWTSecret), nil
	})
	if err != nil {
		log.Printf("Error parsing jwt %s:", err)
		w.WriteHeader(401)
		return
	}

	id, err := token.Claims.GetSubject()
	if err != nil {
		log.Printf("Error parsing id from jwt %s:", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
}

func (a *apiConfig) handlePostLogin(w http.ResponseWriter, r *http.Request) {

	type params struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	p := params{}
	err := decoder.Decode(&p)
	if err != nil {
		log.Printf("Error decoding parameters %s:", err)
		w.WriteHeader(500)
		return
	}

	user, err := a.DB.GetUserByEmail(p.Email)

	if err != nil {
		log.Printf("Error Fetching user %s:", err)
		w.WriteHeader(500)
		return
	}

	tAdd := 24 * time.Hour
	if p.ExpiresInSeconds > 0 && p.ExpiresInSeconds < 60*p.ExpiresInSeconds*24 {
		tAdd = time.Second * time.Duration(p.ExpiresInSeconds)
	}
	mySigningKey := []byte(a.JWTSecret)
	fmt.Println("SUBJECT", user.Id)
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tAdd)),
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   strconv.Itoa(user.Id),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	if err != nil {
		log.Printf("Error signing jwt %s:", err)
		w.WriteHeader(500)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(p.Password))
	if err != nil {
		log.Printf("Incorrect password %s:", err)
		w.WriteHeader(401)
		return
	}

	type returnVals struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}

	resp := returnVals{
		Id:    user.Id,
		Email: user.Email,
		Token: ss,
	}

	data, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error Marshalling Json %s:", err)
		w.WriteHeader(401)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func (a *apiConfig) handlePostUser(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	u := params{}
	err := decoder.Decode(&u)

	if err != nil {
		log.Printf("Error decoding parameters %s:", err)
		w.WriteHeader(500)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password %s:", err)
		w.WriteHeader(500)
		return
	}
	user, err := a.DB.CreateUser(u.Email, string(hashedPass))

	if err != nil {
		log.Printf("Error creating user %s:", err)
		w.WriteHeader(500)
		return
	}

	resp, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling data %s:", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(resp)

}

func (a *apiConfig) handleGetChirpById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpid")

	strId, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Error converting id to int, bad id %s:", err)
		w.WriteHeader(500)
		return
	}

	chirp, err := a.DB.GetChirpById(strId)
	if err != nil {
		log.Printf("Error getting Chirps %s:", err)
		w.WriteHeader(404)
		return
	}

	data, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("Error Marshalling json %s:", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func (a *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := a.DB.GetChirps()
	if err != nil {
		log.Printf("Error getting Chirps %s:", err)
		w.WriteHeader(500)
		return
	}

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

func (a *apiConfig) handlePostChirp(w http.ResponseWriter, r *http.Request) {

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
