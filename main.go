package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

var db *sql.DB

func initDB() {
	var err error
	connStr := "host=localhost port=5432 user=web_server password=apikeypass dbname=api_db sslmode=disable"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			first_name VARCHAR(50),
			last_name VARCHAR(50),
			phone VARCHAR(20)
		);
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := db.QueryRow(
		"INSERT INTO users (first_name, last_name, phone) VALUES ($1, $2, $3) RETURNING id",
		user.FirstName, user.LastName, user.Phone,
	).Scan(&user.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	query := "SELECT id, first_name, last_name, phone FROM users WHERE 1=1"
	args := []interface{}{}

	if id, ok := params["id"]; ok {
		query += " AND id = $1"
		args = append(args, id)
	} else if fname := r.URL.Query().Get("first_name"); fname != "" {
		query += " AND first_name = $1"
		args = append(args, fname)
	} else if lname := r.URL.Query().Get("last_name"); lname != "" {
		query += " AND last_name = $1"
		args = append(args, lname)
	} else {
		http.Error(w, "Missing search parameter", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Phone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func main() {
	initDB()
	r := mux.NewRouter()
	r.HandleFunc("/users", createUser).Methods("POST")
	r.HandleFunc("/users/{id}", getUser).Methods("GET")
	r.HandleFunc("/users", getUser).Methods("GET")

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", r)
}
