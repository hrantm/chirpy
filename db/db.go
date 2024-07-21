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

type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
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
	var chirp Chirp
	loadedData, err := db.loadDB()
	if err != nil {
		return chirp, err
	}
	largestId := 0
	data := DBStructure{
		Chirps: make(map[int]Chirp),
		Users:  make(map[int]User),
	}
	if loadedData.Users != nil {
		data.Users = loadedData.Users
	}
	if loadedData.Chirps != nil {
		data.Chirps = loadedData.Chirps
	}

	for _, c := range data.Chirps {
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

func (db *DB) CreateUser(email string, password string) (User, error) {
	var user User
	data := DBStructure{
		Chirps: make(map[int]Chirp),
		Users:  make(map[int]User),
	}
	loadedData, err := db.loadDB()
	if loadedData.Users != nil {
		data.Users = loadedData.Users
	}
	if loadedData.Chirps != nil {
		data.Chirps = loadedData.Chirps
	}
	if err != nil {
		return user, err
	}

	largestId := 0
	for _, u := range data.Users {
		if u.Email == email {
			return user, errors.New("User with this email already exists")
		}
		if u.Id > largestId {
			largestId = u.Id
		}
		data.Users[u.Id] = u
	}

	data.Users[largestId+1] = User{
		Email:    email,
		Id:       largestId + 1,
		Password: password,
	}
	err = db.writeDB(data)
	if err != nil {
		return user, nil
	}

	return data.Users[largestId+1], nil
}

func (db *DB) GetUsers() ([]User, error) {
	var users []User
	data, err := db.loadDB()
	if err != nil {
		return users, err
	}

	for _, user := range data.Users {
		users = append(users, user)
	}

	return users, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	var user User
	data, err := db.loadDB()
	if err != nil {
		return user, err
	}

	for _, u := range data.Users {
		if u.Email == email {
			return u, nil
		}
	}

	return user, nil
}
