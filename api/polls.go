package main

import (
	"net/http"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type poll struct {
	ID      bson.ObjectId `bson:"_id" json:"id"`
	Title   string        `json:"title"`
	Options []string      `json:"options"`
	//Field appears in JSON as key "results" and the field
	//is omitted from the object if its value is empty
	Results map[string]int `json:"results, omitempty"`
}

func handlePolls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handlePollsGet(w, r)
		return
	case "POST":
		handlePollsPost(w, r)
		return
	case "DELETE":
		handlePollsDelete(w, r)
		return
	case "OPTIONS":
		// A CORS browser will send a pre-flight reqeust(with an HTTP
		// method of OPTIONS) asking for permission to make a DELETE
		// request (listed in the Access-Control-Request-Method request header)
		w.Header().Add("Access-Control-Allow-Methods", "DELETE")
		respond(w, r, http.StatusOK, nil)
		return
	}
	// not found
	respondHTTPErr(w, r, http.StatusNotFound)
}

func handlePollsGet(w http.ResponseWriter, r *http.Request) {
	db := GetVar(r, "db").(*mgo.Database)
	// create an object referring to the polls collection
	c := db.C("polls")
	var q *mgo.Query
	p := NewPath(r.URL.Path)
	if p.HasID() {
		// get specific poll
		// ObjectIdHex method convert the ID from a string to a bson.ObjecId type
		q = c.FindId(bson.ObjectIdHex(p.ID))
	} else {
		// get all polls
		q = c.Find(nil)
	}
	var result []*poll
	// query all the polls and populate the result object
	if err := q.All(&result); err != nil {
		respondErr(w, r, http.StatusInternalServerError, err)
		return
	}
	respond(w, r, http.StatusOK, &result)
}

func handlePollsPost(w http.ResponseWriter, r *http.Request) {
	db := GetVar(r, "db").(*mgo.Database)
	c := db.C("polls")
	var p poll
	// decode the body of the request
	if err := decodeBody(r, &p); err != nil {
		respondErr(w, r, http.StatusBadRequest, "failed to read poll from request", err)
		return
	}
	p.ID = bson.NewObjectId()
	if err := c.Insert(p); err != nil {
		respondErr(w, r, http.StatusInternalServerError, "failed to insert poll", err)
		return
	}
	// As per HTTP standards, set the location header of the response
	// and respond with a 201 status code, pointing to the URL from
	// which the newly created poll maybe accessed.
	w.Header().Set("Location", "polls/"+p.ID.Hex())
	respond(w, r, http.StatusCreated, nil)
}

func handlePollsDelete(w http.ResponseWriter, r *http.Request) {
	db := GetVar(r, "db").(*mgo.Database)
	c := db.C("polls")
	p := NewPath(r.URL.Path)
	if !p.HasID() {
		respondErr(w, r, http.StatusMethodNotAllowed, "Cannot delete all polls.")
		return
	}
	// passing in the ID in the path after converting it into a bson.ObjectId type
	if err := c.RemoveId(bson.ObjectIdHex(p.ID)); err != nil {
		respondErr(w, r, http.StatusInternalServerError, "failed to delete poll", err)
		return
	}
	respond(w, r, http.StatusOK, nil)
}
