package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/btoll/trivial-admin/postgres"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	username = "btoll"
	password = "test"
	dbname   = "trivial"
)

var db *sql.DB
var sessionManager SessionManager

func main() {
	var err error
	db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s", username, password, host, dbname))
	if err != nil {
		panic(err)
	}
	//		defer db.Close()
	fmt.Println("[INFO] Database started.")

	sessionManager = NewSessionManager()
	sessionManager.Store = postgres.New(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/", BaseHandler)
	mux.HandleFunc("/create", CreateLoginHandler)
	mux.HandleFunc("/download", DownloadHandler)
	mux.HandleFunc("/get_games", GetGamesHandler)
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/question", CreateQuestionHandler)
	mux.HandleFunc("/signin", SigninHandler)
	mux.HandleFunc("/view", ViewGameHandler)
	log.Fatal(http.ListenAndServeTLS(":3001", "cert.pem", "key.pem", sessionManager.Authenticate(mux)))
	// log.Fatal(http.ListenAndServeTLS(":3001", "cert.pem", "key.pem", mux))
}
