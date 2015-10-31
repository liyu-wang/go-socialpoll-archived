package main

import (
	"net/http"
	"sync"
)

// the map of per-request data
var vars map[*http.Request]map[string]interface{}
var varsLock sync.RWMutex

// OpenVars prepare the vars map to hold
// variables for a particular request
func OpenVars(r *http.Request) {
	// locks the mutex so that we can safely modify the map
	varsLock.Lock()
	if vars == nil {
		vars = map[*http.Request]map[string]interface{}{}
	}
	vars[r] = map[string]interface{}{}
	varsLock.Unlock()
}

// CloseVars safely deletes the entry in the vars map
// for the request onece finish handling the request.
func CloseVars(r *http.Request) {
	varsLock.Lock()
	delete(vars, r)
	varsLock.Unlock()
}

// GetVar function makes it easy for us to get a
// variable from the map for the request
func GetVar(r *http.Request, key string) interface{} {
	// Since we are using sync.RWMutex, it is safe
	// for many read to occur at the same time, as
	// long as a write isn't happening.
	varsLock.RLock()
	value := vars[r][key]
	varsLock.RUnlock()
	return value
}

// SetVar allows us to set a variable for the request
func SetVar(r *http.Request, key string, value interface{}) {
	varsLock.Lock()
	vars[r][key] = value
	varsLock.Unlock()
}
