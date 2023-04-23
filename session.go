package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

var sessions = make(map[string]*Session)

type SessionManager struct {
	Sessions map[string]*Session
	Store    Store
}

type Session struct {
	Username string
	Value    string
	Expiry   time.Time
	Cookie   *SessionCookie
}

type SessionCookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie#samesitesamesite-value
	SameSite http.SameSite
}

func setExpiry(seconds time.Duration) time.Time {
	return time.Now().Add(seconds * time.Second)
}

func NewSessionManager() SessionManager {
	return SessionManager{
		Sessions: make(map[string]*Session),
	}
}

func NewSession(token string) *Session {
	session := &Session{
		Username: username,
		Value:    token,
		Expiry:   setExpiry(259200), // 3 days.
		Cookie: &SessionCookie{
			Name:  "trivial-admin-session",
			Value: token,
			//			Domain:  "127.0.0.1",
			Path:     "/",
			Expires:  setExpiry(3600),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	}
	fmt.Printf("[INFO] Session `%s` added.\n", token)
	sessionManager.Sessions[token] = session
	return session
}

func (s *Session) WriteCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     s.Cookie.Name,
		Value:    s.Cookie.Value,
		Domain:   s.Cookie.Domain,
		Path:     s.Cookie.Path,
		Expires:  s.Cookie.Expires,
		Secure:   s.Cookie.Secure,
		HttpOnly: s.Cookie.HttpOnly,
		SameSite: s.Cookie.SameSite,
	}
	http.SetCookie(w, cookie)
	w.Header().Add("Cache-Control", `no-cache="Set-Cookie"`)
}

func (s *Session) IsExpired() bool {
	return s.Expiry.Before(time.Now())
}

func (s *SessionManager) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/favicon.ico" && r.URL.Path != "/login" && r.URL.Path != "/signin" && r.URL.Path != "/create" {
			setCorsHeaders(w)
			c, err := r.Cookie("trivial-admin-session")
			if err != nil {
				if err == http.ErrNoCookie {
					fmt.Println("[WARNING] http: no Cookie")
					http.Redirect(w, r, "https://127.0.0.1:3001/login", 302)
				}
			} else {
				session, exists := s.Sessions[c.Value]
				if !exists {
					fmt.Println("[INFO] Session doesn't exist, creating now...")
					session = NewSession(c.Value)
				}
				fmt.Println("[INFO] Adding the user session to the request context...")
				newCtx := context.WithValue(r.Context(), "session", session)
				r = r.WithContext(newCtx)
			}
		}
		next.ServeHTTP(w, r)
	})
}
