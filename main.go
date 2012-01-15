package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

const max_retries = 5
const max_time = 24 * time.Hour

type notifier struct {
	Name     string
	Driver   string
	Disabled bool
	Config   map[string]string
}

type notification struct {
	Url   string `json:"url"`
	Event string `json:"event"`
	Msg   string `json:"msg"`
}

func checker(u string, ch chan<- notification) {
	note := notification{Url: u}
	start := time.Now()
	for {
		log.Printf("Getting %v", u)
		r, err := http.Get(u)
		if err == nil {
			defer r.Body.Close()
			log.Printf("Status of %s:  %v", u, r.Status)
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				note.Msg = fmt.Sprintf("Connected to %s, status=%s",
					u, r.Status)
				note.Event = "connected"
				break
			} else {
				log.Printf("HTTP Error:  %v", r.Status)
			}
		} else {
			log.Printf("Error:  %v", err)
		}

		if time.Now().Sub(start) > max_time {
			note.Msg = fmt.Sprintf("Giving up on %s", u)
			note.Event = "timeout"
			break
		}
		time.Sleep(5 * time.Second)
	}
	ch <- note
}

func main() {
	flag.Parse()
	notifiers, err := loadNotifiers()
	if err != nil {
		log.Printf("Problem loading notifiers: %v", err)
	}

	ch := make(chan notification)
	resq := make(chan bool)
	todo := 0
	pending := 0

	if flag.NArg() == 0 {
		log.Fatalf("You didn't give me any URLs to watch.")
	}

	for _, u := range flag.Args() {
		go checker(u, ch)
		todo++
	}

	for todo > 0 || pending > 0 {
		select {
		case note := <-ch:
			todo--
			for _, n := range notifiers {
				if !n.Disabled {
					go n.notify(note, resq)
					pending++
				}

			}
		case _ = <-resq:
			pending--
		}
	}
}
