package session

import (
	"context"
	"log"
	"net/http"
	"sync"

	"radio/utils"
)

// Doesn't mark requests /assets/
type Protect struct {
	handler  http.Handler
	database map[string]string
	mutex    sync.RWMutex
}

func (p *Protect) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// prevent token leakage etc."no-referrer"
	w.Header().Set("Referrer-Policy", "no-referrer")

	if r.Method == http.MethodPost {
		// Cookie should be set by session middleware
		// But it doesn't hurt to double-check.
		session, err := r.Cookie("session")
		if err != nil || session.Value == "" {
			log.Printf("Missing session cookie")
			return
		}

		p.mutex.RLock()
		expected := p.database[session.Value]
		p.mutex.RUnlock()

		token := r.FormValue("csrf")

		if expected == "" || token == "" || token != expected {
			log.Printf(
				"Potential CSRF attack detected. %s sent a request with cookie '%s' and CSRF '%s', we were expecting token '%s'",
				r.RemoteAddr, session.Value, token, expected,
			)

			return
		}
	}

	newToken := utils.GenerateToken()
	ctx := context.WithValue(r.Context(), "csrf", newToken)

	if session, err := r.Cookie("session"); err == nil && session.Value != "" {
		p.mutex.Lock()
		p.database[session.Value] = newToken
		p.mutex.Unlock()
	}

	p.handler.ServeHTTP(w, r.Clone(ctx))
	return
}

func NewProtect(h http.Handler) *Protect {
	return &Protect{
		handler:  h,
		database: make(map[string]string),
		mutex:    sync.RWMutex{},
	}
}
