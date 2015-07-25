package main

import (
	"net/http"
	"sync"
)

// the map of per-request data
var vars map[*http.Request]map[string]interface{}
var varsLock sync.RWMutex

// OpenVars allow us safely modify the map
func OpenVars(r *http.Request) {
	varsLock.Lock()
	if vars == nil {
		vars = map[*http.Request]map[string]interface{}{}
	}
	vars[r] = map[string]interface{}{}
	varsLock.Unlock()
}

// CloseVars safely deletes the entry
// in the vars map for the request
func CloseVars(r *http.Request) {
	varsLock.Lock()
	delete(vars, r)
	varsLock.Unlock()
}

// GetVar function makes it easy for us to get a
// variable from the map for the specified request
func GetVar(r *http.Request, key string) interface{} {
	varsLock.RLock()
	value := vars[r][key]
	varsLock.RUnlock()
	return value
}

// SetVar allows us to set one
func SetVar(r *http.Request, key string, value interface{}) {
	varsLock.Lock()
	vars[r][key] = value
	varsLock.Unlock()
}
