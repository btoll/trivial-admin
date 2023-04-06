package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.gohtml
var templateFiles embed.FS

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Success     bool   `json:"success"`
	Code        string `json:"code,omitempty"`
	CodeMessage string `json:"message,omitempty"`
	UIMessage   string `json:"ui_message,omitempty"`
}

type Question struct {
	Name    string   `json:"name"`
	Weight  int      `json:"weight"`
	Answer  []int    `json:"answer"`
	Choices []string `json:"choices"`
}

var game = []*Question{}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	r.Header = http.Header{
		"Content-Type": {"text/html; charset=utf-8"},
	}

	tpl := template.Must(template.ParseFS(templateFiles, "templates/*.gohtml"))
	if err := tpl.Execute(w, "http://127.0.0.1:3001"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateHandler(w http.ResponseWriter, r *http.Request) {
	createLogin := &Login{}
	err := json.NewDecoder(r.Body).Decode(createLogin)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(createLogin.Password), 8)
	// Go creates prepared statements for you under the covers.  A simple db.Query(sql, param1, param2),
	// for example, works by preparing the sql, then executing it with the parameters and finally
	// closing the statement.
	// http://go-database-sql.org/prepared.html#avoiding-prepared-statements
	_, err = db.Query("INSERT INTO users VALUES (DEFAULT, $1, $2)", createLogin.Username, string(hashedPassword))
	if err != nil {
		// https://pkg.go.dev/github.com/lib/pq?utm_source=godoc#hdr-Errors
		// https://www.postgresql.org/docs/current/errcodes-appendix.html
		if err, ok := err.(*pq.Error); ok {
			var duplicateKey pq.ErrorCode = "23505"
			fmt.Println("err.Code", err.Code)
			if duplicateKey == err.Code {
				json.NewEncoder(w).Encode(&Response{
					Success:     false,
					Code:        fmt.Sprintf("%v", err.Code),
					CodeMessage: err.Code.Name(),
					UIMessage:   "Username already exists.",
				})
			} else {
				// todo
				panic(err)
			}
		}
		return
	}
	// todo
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "trivial-auth",
		Expires:  time.Now(),
		HttpOnly: true,
	})
	json.NewEncoder(w).Encode(&Response{
		Success:   true,
		UIMessage: fmt.Sprintf("User %s successfully created.", createLogin.Username),
	})
}

func joinAnswers(answers []int) string {
	res := []string{}
	for _, v := range answers {
		res = append(res, strconv.Itoa(v))
	}
	return strings.Join(res, ",")
}

func CreateQuestionHandler(w http.ResponseWriter, r *http.Request) {
	question := &Question{}
	err := json.NewDecoder(r.Body).Decode(question)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	file, err := os.OpenFile("derp.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	defer file.Close()
	_, err = fmt.Fprintf(file,
		"%s|%d|%s|%s\n",
		question.Name,
		question.Weight,
		joinAnswers(question.Answer),
		strings.Join(question.Choices, "|"))
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: "Question could not be written to disk, try again.",
		})
	} else {
		game = append(game, question)
		json.NewEncoder(w).Encode(&Response{
			Success:   true,
			UIMessage: "Question successfully received and written to disk.",
		})
	}
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=game.csv")
	w.Header().Set("Content-Type", "application/octet-stream")
	// TODO: this should pass a name from the session?
	http.ServeFile(w, r, "derp.csv")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	compareLogin := &Login{}
	err := json.NewDecoder(r.Body).Decode(compareLogin)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	result := db.QueryRow("SELECT password FROM users WHERE username=$1", compareLogin.Username)
	storedLogin := &Login{}
	err = result.Scan(&storedLogin.Password)
	if err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: err.Error(),
		})
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedLogin.Password), []byte(compareLogin.Password))
	if err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:     false,
			CodeMessage: err.Error(),
			UIMessage:   "Bad username or password.",
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "trivial-auth",
		Expires:  time.Now(),
		HttpOnly: true,
	})
	json.NewEncoder(w).Encode(&Response{
		Success:   true,
		UIMessage: "Login successful.",
	})
}

func PrintGameHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(game)
}
