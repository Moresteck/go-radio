package session

import (
	"log"
	"net/http"
	"time"

	"radio/utils"
)

type Session struct {
	handler http.Handler
	debug   bool
}

func (s *Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.debug {
		realAddr := r.Header.Get("X-NGINX-REAL-IP")
		log.Println(r.Proto, r.Method, r.URL.Path, r.UserAgent(), r.RemoteAddr, realAddr)
	}

	session, err := r.Cookie("session")
	if err == nil && session.Value != "" {
		s.handler.ServeHTTP(w, r)
		return
	}

	sessionId := utils.GenerateToken()
	cookie := http.Cookie{
		Name:     "session",
		Value:    sessionId,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Hour * 10),
	}

	// OWASP-certified engineering lol
	r.AddCookie(&cookie)
	http.SetCookie(w, &cookie)

	s.handler.ServeHTTP(w, r)
}

func New(h http.Handler, b bool) *Session {
	return &Session{handler: h, debug: b}
}
