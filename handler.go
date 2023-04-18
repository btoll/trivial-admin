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

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

//go:embed template/app/*.gohtml
var templateFilesApp embed.FS

//go:embed template/login/*.gohtml
var templateFilesLogin embed.FS

var errTpl = template.Must(template.New("403.gohtml").ParseFiles("template/error/403.gohtml"))

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

type Game struct {
	Name      string      `json:"name"`
	Filename  string      `json:"filename"`
	Questions []*Question `json:"questions"`
}

type Question struct {
	Filename string   `json:"filename"`
	Name     string   `json:"name"`
	Weight   int      `json:"weight"`
	Answer   []int    `json:"answer"`
	Choices  []string `json:"choices"`
}

type Error struct {
	Code    int
	Message string
}

var game = []*Question{}

func joinAnswers(answers []int) string {
	res := []string{}
	for _, v := range answers {
		res = append(res, strconv.Itoa(v))
	}
	return strings.Join(res, ",")
}

func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Origin", "https://127.0.0.1:3001")
	w.Header().Set("Access-Control-Request-Method", "GET,OPTIONS,POST")
}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	r.Header = http.Header{
		"Content-Type": {"text/html; charset=utf-8"},
	}
	setCorsHeaders(w)
	tpl := template.Must(template.ParseFS(templateFilesApp, "template/app/*.gohtml"))
	if err := tpl.Execute(w, "https://127.0.0.1:3001"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateLoginHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	createLogin := &Login{
		Username: r.Form.Get("usernameCreate"),
		Password: r.Form.Get("passwordCreate"),
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
			//			fmt.Println("err.Code", err.Code)
			if duplicateKey == err.Code {
				errTpl.Execute(w, &Error{
					Code:    400,
					Message: "Username already exists",
				})
			} else {
				// If the `users` table doesn't exist, the `err.Code` will be "42P01".
				errTpl.Execute(w, &Error{
					Code:    502,
					Message: "Bad Gateway",
				})
			}
			return
		}
		w.Write([]byte("User created"))
		return
	}
}

func CreateQuestionHandler(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	question := &Question{}
	err := json.NewDecoder(r.Body).Decode(question)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	file, err := os.OpenFile(fmt.Sprintf("games/%s", question.Filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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
	setCorsHeaders(w)
	question := &Question{}
	err := json.NewDecoder(r.Body).Decode(question)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", question.Filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	// TODO: this should pass a name from the session?
	http.ServeFile(w, r, fmt.Sprintf("games/%s", question.Filename))
}

func GetGamesHandler(w http.ResponseWriter, r *http.Request) {
	ctxVal := r.Context().Value("session")
	session := ctxVal.(*Session)
	fmt.Println("session.Username", session.Username)
	fmt.Println("session.Cookie", session.Cookie)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.Header = http.Header{
		"Content-Type": {"text/html; charset=utf-8"},
	}

	tpl := template.Must(template.ParseFS(templateFilesLogin, "template/login/*.gohtml"))
	if err := tpl.Execute(w, "https://127.0.0.1:3001"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func SigninHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	compareLogin := &Login{
		Username: r.Form.Get("usernameLogin"),
		Password: r.Form.Get("passwordLogin"),
	}
	result := db.QueryRow("SELECT username, password FROM users WHERE username=$1", compareLogin.Username)
	storedLogin := &Login{}
	err = result.Scan(&storedLogin.Username, &storedLogin.Password)
	if err != nil {
		err = errTpl.Execute(w, &Error{
			Code:    403,
			Message: "Unauthorized",
		})
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedLogin.Password), []byte(compareLogin.Password))
	if err != nil {
		err = errTpl.Execute(w, &Error{
			Code:    403,
			Message: "Unauthorized",
		})
		return
	}
	NewSession(storedLogin.Username).WriteCookie(w)
	http.Redirect(w, r, "https://127.0.0.1:3001/", 302)
}

func ViewGameHandler(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	json.NewEncoder(w).Encode(game)
}
