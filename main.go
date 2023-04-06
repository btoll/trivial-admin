package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "btoll"
	password = "test"
	dbname   = "trivial"
)

var db *sql.DB

func initDB() {
	var err error
	conn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		dbname)
	db, err = sql.Open("postgres", conn)
	if err != nil {
		panic(err)
	}
	fmt.Println("[INFO] Database started.")
}

func main() {
	initDB()
	mux := http.NewServeMux()
	mux.HandleFunc("/", BaseHandler)
	mux.HandleFunc("/create", CreateHandler)
	mux.HandleFunc("/download", DownloadHandler)
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/print", PrintGameHandler)
	mux.HandleFunc("/question", CreateQuestionHandler)
	log.Fatal(http.ListenAndServe(":3001", mux))
}
