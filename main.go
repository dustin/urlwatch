package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/devcamcar/notifo.go"
	"github.com/rem7/goprowl"
	"log"
	"net/http"
	"os"
	"time"
)

const max_retries = 5
const max_time = 24 * time.Hour

type notifier struct {
	Driver string
	Config map[string]string
}

type notification struct {
	url   string
	event string
	msg   string
}

func (n notifier) notifyProwl(note notification) (err error) {
	p := goprowl.Goprowl{}
	p.RegisterKey(n.Config["apikey"])

	msg := goprowl.Notification{
		Application: n.Config["application"],
		Description: note.msg,
		Event:       note.event,
		Priority:    n.Config["priority"],
		Url:         note.url,
	}

	if err = p.Push(&msg); err != nil {
		log.Printf("Error delivering prowl message: %v", err)
	}

	return
}

func (n notifier) notifyNotifo(note notification) (err error) {
	nfo := notifo.New(n.Config["apiuser"], n.Config["apisecret"])
	if _, err = nfo.SendNotification(n.Config["to"], note.msg,
		n.Config["label"], n.Config["title"], note.url); err != nil {

		log.Printf("Error delivering notifo message:  %v", err)
	}
	return
}

func (n notifier) notify(note notification, resq chan<- bool) {
	defer func() { resq <- true }()

	for i := 0; i < max_retries; i++ {
		switch n.Driver {
		default:
			log.Fatalf("Unknown driver:  %v", n.Driver)
		case "prowl":
			if n.notifyProwl(note) == nil {
				return
			}
		case "notifo":
			if n.notifyNotifo(note) == nil {
				return
			}
		}
		time.Sleep(1 * time.Second)
		log.Printf("Retrying a %s notification", n.Driver)
	}
}

func loadNotifiers() (map[string]notifier, error) {
	notifiers := map[string]notifier{}

	f, err := os.Open("notify.json")
	if err != nil {
		return notifiers, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err = d.Decode(&notifiers); err != nil {
		return notifiers, err
	}

	return notifiers, nil
}

func checker(u string, ch chan<- notification) {
	note := notification{url: u}
	start := time.Now()
	for {
		log.Printf("Getting %v", u)
		r, err := http.Get(u)
		if err == nil {
			defer r.Body.Close()
			log.Printf("Status of %s:  %v", u, r.Status)
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				note.msg = fmt.Sprintf("Connected to %s, status=%s",
					u, r.Status)
				note.event = "connected"
				break
			} else {
				log.Printf("HTTP Error:  %v", r.Status)
			}
		} else {
			log.Printf("Error:  %v", err)
		}

		if time.Now().Sub(start) > max_time {
			note.msg = fmt.Sprintf("Giving up on %s", u)
			note.event = "timeout"
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
				go n.notify(note, resq)
				pending++

			}
		case _ = <-resq:
			pending--
		}
	}
}
