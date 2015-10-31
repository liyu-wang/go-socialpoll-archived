package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/stretchr/graceful"

	"gopkg.in/mgo.v2"
)

func main() {
	var (
		// defines string flags with specified name, default value, and usage string.
		addr  = flag.String("addr", ":8080", "endpoint address")
		mongo = flag.String("mongo", "127.0.0.1", "mongodb address")
	)
	flag.Parse()
	log.Println("Dialing mongo", *mongo)
	db, err := mgo.Dial(*mongo)
	if err != nil {
		log.Fatalln("failed to connect to mongo:", err)
	}
	defer db.Close()
	mux := http.NewServeMux()
	// register a signle handler for all requests begin with the path /polls/
	mux.HandleFunc("/polls/", withCORS(withVars(withData(db, withAPIKey(handlePolls)))))
	log.Println("Starting web server on", *addr)
	// specify time.Duration when running any http.Handler(ServeMux handler), which
	// allow in-flight requests some time to complete before the function exits.
	// wait 1 sec before killing active requests and stopping the server.
	graceful.Run(*addr, 1*time.Second, mux)
	log.Println("Stopping...")
}

// ---------------------------
// handler wrappers
// ---------------------------

// API key
func withAPIKey(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isValidAPIKey(r.URL.Query().Get("key")) {
			respondErr(w, r, http.StatusUnauthorized, "invalid API key")
			return
		}
		fn(w, r)
	}
}

// Database session
func withData(d *mgo.Session, f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		thisDb := d.Copy()
		defer thisDb.Close()
		SetVar(r, "db", thisDb.DB("ballots"))
		f(w, r)
	}
}

// Per request variables
func withVars(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		OpenVars(r)
		defer CloseVars(r)
		fn(w, r)
	}
}

// CORS header
func withCORS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Location")
		fn(w, r)
	}
}

func isValidAPIKey(key string) bool {
	return key == "abc123"
}
