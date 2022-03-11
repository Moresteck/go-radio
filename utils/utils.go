package utils

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"

	"html/template"

	"bytes"
	"log"
	"net/http"
	"time"
)

func GenerateToken() string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).
		EncodeToString(GenerateSalt())
}

// Generates 32 random bytes
func GenerateSalt() []byte {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("COULDN'T GENERATE CSRF TOKEN: %v\n", err)
	}

	return randomBytes
}

// BaseView is a struct used by layout/base.html template. Homepage uses it as
// is, and other pages should embed BaseView in their respective ViewData
// UserString is an empty string if not logged in. A non-empty StatusMessage
// requires a fitting partial to be parsed too.
type BaseView struct {
	CurrentYear   int
	StatusMessage string
	Title         string
	UserString    string
	ViewData      interface{}
}

type JsonResponse struct {
	Err     bool   `json:"error"`
	Message string `json:"message"`
}

var tmplError = template.Must(template.ParseFiles(
	"html/base.layout.html",
))

func SendHTTP(w http.ResponseWriter, r *http.Request, msg string, title string) {
	w.Header().Set("Content-Type", "text/html")

	view := BaseView{
		StatusMessage: msg,
		Title:         title,
		CurrentYear:   time.Now().Year(),
	}

	tmplError.Execute(w, view)
}

func SendJSON(w http.ResponseWriter, r *http.Request, content []byte) {
	w.Header().Set("Content-Type", "application/json")

	w.Write(content)
}

func SendErrorJSON(w http.ResponseWriter, r *http.Request, msg string) {
	jsonerr := JsonResponse{
		Err:     true,
		Message: msg,
	}

	j, _ := JSONMarshal(jsonerr)

	SendJSON(w, r, j)
}

func SendResponseJSON(w http.ResponseWriter, r *http.Request, msg string) {
	jsonres := JsonResponse{
		Err:     false,
		Message: msg,
	}

	j, _ := JSONMarshal(jsonres)

	SendJSON(w, r, j)
}

func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(t)
	return buffer.Bytes(), err
}
