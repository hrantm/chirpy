package db

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
)

type DB struct {
	path string
	mu   *sync.RWMutex
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

func NewDB(path string) (*DB, error) {
	_, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	return &DB{path: path, mu: &sync.RWMutex{}}, nil
}

func (db *DB) loadDB() (DBStructure, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	f, err := os.Open(db.path)

	var data DBStructure
	if err != nil {
		return data, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)

	err = decoder.Decode(&data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	chirps, err := db.GetChirps()
	var chirp Chirp
	if err != nil {
		return chirp, err
	}
	largestId := 0
	data := DBStructure{
		Chirps: make(map[int]Chirp),
	}
	for _, c := range chirps {
		if c.Id > largestId {
			largestId = c.Id
		}
		data.Chirps[c.Id] = c
	}
	data.Chirps[largestId+1] = Chirp{
		Body: body,
		Id:   largestId + 1,
	}
	err = db.writeDB(data)
	if err != nil {
		return chirp, nil
	}
	return data.Chirps[largestId+1], nil
}

func (db *DB) writeDB(data DBStructure) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	jsonData, err := json.Marshal(data)

	if err != nil {
		return err
	}

	f, err := os.Create(db.path)

	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(jsonData)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) GetChirps() ([]Chirp, error) {

	var chirps []Chirp
	data, err := db.loadDB()
	if err != nil {
		return chirps, nil
	}

	for _, v := range data.Chirps {
		chirps = append(chirps, v)
	}
	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].Id < chirps[j].Id
	})
	return chirps, nil
}

func (db *DB) GetChirpById(id int) (Chirp, error) {
	var chirp Chirp
	data, err := db.loadDB()
	if err != nil {
		return chirp, nil
	}
	for k, v := range data.Chirps {
		if k == id {
			chirp = v
			break
		}
	}
	if chirp.Id == 0 {
		return chirp, errors.New("Chirp not found in db")
	}
	return chirp, nil
}
