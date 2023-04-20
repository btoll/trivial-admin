package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
)

//go:embed template/app/*.gohtml
var templateFilesApp embed.FS

//go:embed template/app/page/*.gohtml
var templateFilesPage embed.FS

//go:embed template/app/css/*.gohtml
var templateFilesCSS embed.FS

//go:embed template/app/js/*.gohtml
var templateFilesJS embed.FS

//go:embed template/login/*.gohtml
var templateFilesLogin embed.FS

var errTpl = template.Must(template.New("403.gohtml").ParseFiles("template/error/403.gohtml"))

type Response struct {
	Success     bool   `json:"success"`
	Code        string `json:"code,omitempty"`
	CodeMessage string `json:"message,omitempty"`
	UIMessage   string `json:"ui_message,omitempty"`
}

type Game struct {
	Name     string `json:"name"`
	Filename string `json:"filename"`
	//	Questions []*Question `json:"questions"`
	Questions []string `json:"questions"`
}

type Question struct {
	Filename string   `json:"filename"`
	Name     string   `json:"name"`
	Weight   int      `json:"weight"`
	Answer   []int    `json:"answer"`
	Choices  []string `json:"choices"`
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

func AddGameHandler(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	ctxVal := r.Context().Value("session")
	session := ctxVal.(*Session)
	game := &Game{}
	err := json.NewDecoder(r.Body).Decode(game)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	err = os.Mkdir(fmt.Sprintf("games/%s", "ben"), os.FileMode(0700))
	if err != nil {
		fmt.Printf("Directory `games/%s` already exists, not creating...\n", "ben")
	}
	_, err = db.Query("INSERT INTO games (owner_id, name, filename) VALUES ((SELECT user_id FROM users WHERE username = $1), $2, $3)", session.Username, game.Name, game.Filename)
	if err != nil {
		// err.Code 23505
		// err pq: duplicate key value violates unique constraint "games_name_key"
		// or
		// err.Code 23505
		// err pq: duplicate key value violates unique constraint "games_filename_key"
		fmt.Println(err)
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	fmt.Printf("Added game `%s` in filename %s\n", game.Name, game.Filename)
	json.NewEncoder(w).Encode(&Response{
		Success:   true,
		UIMessage: fmt.Sprintf("Added game `%s` in filename %s\n", game.Name, game.Filename),
	})
}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	r.Header = http.Header{
		"Content-Type": {"text/html; charset=utf-8"},
	}
	tpl := template.Must(template.ParseFS(templateFilesApp, "template/app/*.gohtml"))
	template.Must(tpl.ParseFS(templateFilesPage, "template/app/page/*.gohtml"))
	template.Must(tpl.ParseFS(templateFilesCSS, "template/app/css/*.gohtml"))
	template.Must(tpl.ParseFS(templateFilesJS, "template/app/js/*.gohtml"))
	if err := tpl.Execute(w, "https://127.0.0.1:3001"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateLoginHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	l := &Login{
		Username: r.Form.Get("usernameCreate"),
		Password: r.Form.Get("passwordCreate"),
	}
	err = l.Create()
	if err != nil {
		errTpl.Execute(w, err)
		return
	}
	w.Write([]byte("User created"))
}

func CreateQuestionHandler(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	ctxVal := r.Context().Value("session")
	session := ctxVal.(*Session)
	question := &Question{}
	err := json.NewDecoder(r.Body).Decode(question)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusInternalServerError)
		return
	}
	file, err := os.OpenFile(fmt.Sprintf("games/%s/%s", session.Username, question.Filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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
	http.ServeFile(w, r, fmt.Sprintf("games/%s", question.Filename))
}

func GetGamesHandler(w http.ResponseWriter, r *http.Request) {
	//	ctxVal := r.Context().Value("session")
	//	session := ctxVal.(*Session)
	rows, err := db.Query("SELECT name,filename FROM games WHERE owner_id=1")
	if err != nil {
		fmt.Println(err)
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	defer rows.Close()
	games := []*Game{}
	for rows.Next() {
		var game Game
		if err := rows.Scan(&game.Name, &game.Filename); err != nil {
			json.NewEncoder(w).Encode(&Response{
				Success:   false,
				UIMessage: fmt.Sprintf("%s", err),
			})
		}
		games = append(games, &game)
	}
	if err = rows.Err(); err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
	}
	b, err := json.Marshal(games)
	if err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
	}
	json.NewEncoder(w).Encode(&Response{
		Success:   true,
		UIMessage: string(b),
	})
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

func ReadfileHandler(w http.ResponseWriter, r *http.Request) {
	ctxVal := r.Context().Value("session")
	session := ctxVal.(*Session)
	question := &Question{}
	_ = json.NewDecoder(r.Body).Decode(question)
	// The files are small enough to read in one go.
	// Note that `os.ReadFile` closes the file automatically.
	//	b, err := os.ReadFile(fmt.Sprintf("games/%s/%s", session.Username, question.Filename))
	file, err := os.Open(fmt.Sprintf("games/%s/%s", session.Username, question.Filename))
	if err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	q := []string{}
	for scanner.Scan() {
		q = append(q, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		json.NewEncoder(w).Encode(&Response{
			Success:   false,
			UIMessage: fmt.Sprintf("%s", err),
		})
		return
	}
	json.NewEncoder(w).Encode(&Game{
		Name:      question.Name,
		Filename:  question.Filename,
		Questions: q,
	})
}

func SigninHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	l := &Login{
		Username: r.Form.Get("usernameLogin"),
		Password: r.Form.Get("passwordLogin"),
	}
	err = l.Read()
	if err != nil {
		errTpl.Execute(w, err)
		return
	}
	NewSession(l.Username).WriteCookie(w)
	http.Redirect(w, r, "https://127.0.0.1:3001/", 302)
}

func ViewGameHandler(w http.ResponseWriter, r *http.Request) {
	setCorsHeaders(w)
	json.NewEncoder(w).Encode(game)
}
