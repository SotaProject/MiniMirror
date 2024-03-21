package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

var (
	TargetDomain = os.Getenv("TARGET_DOMAIN")
)

var (
	SecondaryDomains = strings.Split(os.Getenv("SECONDARY_DOMAINS"), ";")
)

func handleExternalRequest(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	externalURL := queryParams.Get("url")

	if externalURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	// Clone request header
	resp, err := http.Get(externalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Check if it is a redirection
	if resp.StatusCode >= 300 && resp.StatusCode <= 308 {
		loc, err := resp.Location()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// replace scheme and host
		loc.Scheme = r.URL.Scheme
		loc.Host = r.URL.Host

		http.Redirect(w, r, loc.String(), resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(body)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Form new URL
	newURL := TargetDomain + r.URL.Path
	// Clone request header
	resp, err := http.Get(newURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	for _, secDomain := range SecondaryDomains {
		body = []byte(strings.ReplaceAll(string(body), secDomain, "/_EXTERNAL_?url=https://pub-e8e8b7fd0569420da3c4f7690cb95bbd.r2.dev"))
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Write(body)
}

func handleCheckAlive(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/_EXTERNAL_", handleExternalRequest).Methods(http.MethodGet)
	r.HandleFunc("/check_alive", handleCheckAlive).Methods(http.MethodGet)
	r.HandleFunc("/{path:.*}", handleRequest)
	fmt.Printf("Starting mirror of %s on port http://localhost:8080", TargetDomain)
	log.Fatal(http.ListenAndServe(":8080", r))
}
