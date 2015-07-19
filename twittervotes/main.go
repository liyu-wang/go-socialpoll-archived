package main

import (
	"log"

	"gopkg.in/mgo.v2"
)

type poll struct {
	Options []string
}

var db *mgo.Session

func dialdb() error {
	var err error
	log.Println("dialing mongodb: localhost")
	db, err = mgo.Dial("localhost")
	return err
}

func closedb() {
	db.Close()
	log.Println("closed database connection")
}

func loadOptions() ([]string, error) {
	var options []string
	iter := db.DB("ballots").C("polls").Find(nil).Iter()
	var p poll
	for iter.Next(&p) {
		// append []string(as slice) to the end of options slice
		options = append(options, p.Options...)
	}
	iter.Close()
	return options, iter.Err()
}

func main() {

}
