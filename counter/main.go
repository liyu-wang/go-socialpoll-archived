package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var fatalErr error

func fatal(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	fatalErr = e
}

const updateDuration = 1 * time.Second

func main() {
	defer func() {
		if fatalErr != nil {
			os.Exit(1)
		}
	}()

	// -----------------------------------
	// Connectiong to the database
	// -----------------------------------

	log.Println("Connecting to database...")
	db, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		fatal(err)
		return
	}

	defer func() {
		log.Println("Closing database connection...")
		db.Close()
	}()
	pollData := db.DB("ballots").C("polls")

	// -----------------------------------
	// Consuming messages in NSQ
	// -----------------------------------

	// store the counters of votes
	var counts map[string]int
	var countsLock sync.Mutex

	log.Println("Connecting to nsq...")
	// consumer for topic: votes channel: counter
	q, err := nsq.NewConsumer("votes", "counter", nsq.NewConfig())
	if err != nil {
		fatal(err)
		return
	}

	q.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		countsLock.Lock()
		defer countsLock.Unlock()
		if counts == nil {
			counts = make(map[string]int)
		}
		vote := string(m.Body)
		counts[vote]++
		return nil
	}))

	// connecting to the http port of the nsqlookupd instance,
	// rather than NSQ instances.
	// this abstraction means that our program doesn't need to know
	// where the messages are coming from in order to consume them.
	if err := q.ConnectToNSQLookupd("localhost:4161"); err != nil {
		fatal(err)
		return
	}

	// -----------------------------------
	// Keeping the database updated
	// -----------------------------------

	log.Println("Waiting for votes on nsq...")
	var updater *time.Timer
	updater = time.AfterFunc(updateDuration, func() {
		countsLock.Lock()
		defer countsLock.Unlock()
		if len(counts) == 0 {
			log.Println("No new votes, skipping database update")
		} else {
			log.Println("Updating database...")
			log.Println(counts)
			ok := true
			for option, count := range counts {
				sel := bson.M{"options": bson.M{"$in": []string{option}}}
				up := bson.M{"$inc": bson.M{"results." + option: count}}
				if _, err := pollData.UpdateAll(sel, up); err != nil {
					log.Println("failed to update:", err)
					ok = false
				}
			}
			if ok {
				log.Println("Finished updating database...")
				counts = nil // reset counts
			}
		}
		updater.Reset(updateDuration)
	})

	// -----------------------------------
	// Responding to Ctrl + C
	// -----------------------------------

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		// blocked until one of the channel is available
		select {
		case <-termChan:
			updater.Stop()
			q.Stop()
		case <-q.StopChan:
			// finished
			return
		}
	}
}
