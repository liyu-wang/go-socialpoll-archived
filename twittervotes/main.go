package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"
	"gopkg.in/mgo.v2"
)

type poll struct {
	Options []string
}

var db *mgo.Session

func dialdb() error {
	var err error
	log.Println("dialing mongodb: localhost")
	db, err = mgo.Dial("127.0.0.1:27017")
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

func publishVotes(votes <-chan string) <-chan struct{} {
	stopchan := make(chan struct{}, 1)
	pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
	go func() {
		// continually pull values from the channel, whenever
		// the channel has no values, execution will be blocked.
		// the for loop exits only when the channel is closed
		for vote := range votes {
			pub.Publish("votes", []byte(vote)) // publish vote
		}
		log.Println("Publisher: Stopping")
		pub.Stop()
		log.Println("Publisher: Stopped")
		stopchan <- struct{}{}
	}()
	return stopchan
}

func main() {
	var stoplock sync.Mutex
	stop := false
	stopChan := make(chan struct{}, 1)
	signalChan := make(chan os.Signal, 1)
	go func() {
		// the goroutine blocks as it is waiting for the
		// signal by trying to read from signalChan
		<-signalChan
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		log.Println("Stopping...")
		stopChan <- struct{}{}
		closeConn()
	}()
	// ask GO to send the signal down the signalChan
	// when someone tries to halt the program
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	if err := dialdb(); err != nil {
		log.Fatalln("failed to dial MongoDB:", err)
	}
	defer closedb()

	// start things
	votes := make(chan string) // chan for votes
	// passing in the votes channel for it to receive from
	// capturing the returned pub stop signal channel
	publisherStoppedChan := publishVotes(votes)
	// passing in the stopChan to receive the stop signal
	// passing in the votes channel for it to send to
	// capturing the returned twitter stream stop signal channel
	twitterStoppedChan := startTwitterStream(stopChan, votes)
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			// there are two goroutines that might try to
			// access the stop variable at the same time
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				break
			}
			stoplock.Unlock()
		}
	}()
	// once the goroutine has started, we then block on the
	// twitterStoppedChan by attempting to read from it.
	// When signal is sent on stopChan, the signal will be sent
	// on twitterStoppedChan.(check the startTwitterStream func)
	// We close the votes channel which will cause the publisher's
	// for...range loop to exit, and the publisher itself to stop,
	// after which the signal will be sent on the publisherStoppedChan.
	<-twitterStoppedChan
	close(votes)
	<-publisherStoppedChan
}
