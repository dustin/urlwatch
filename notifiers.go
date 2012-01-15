package main

import (
	"encoding/json"
	"errors"
	"github.com/devcamcar/notifo.go"
	"github.com/rem7/goprowl"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type notifyFun func(n notifier, note notification) error

var notifyFuns = map[string]notifyFun{
	"prowl":   notifyProwl,
	"notifo":  notifyNotifo,
	"webhook": notifyWebhook,
}

func notifyProwl(n notifier, note notification) (err error) {
	p := goprowl.Goprowl{}
	p.RegisterKey(n.Config["apikey"])

	msg := goprowl.Notification{
		Application: n.Config["application"],
		Description: note.Msg,
		Event:       note.Event,
		Priority:    n.Config["priority"],
		Url:         note.Url,
	}

	return p.Push(&msg)
}

func notifyNotifo(n notifier, note notification) (err error) {
	nfo := notifo.New(n.Config["apiuser"], n.Config["apisecret"])
	_, err = nfo.SendNotification(n.Config["to"], note.Msg,
		n.Config["label"], n.Config["title"], note.Url)
	return
}

func notifyWebhook(n notifier, note notification) (err error) {
	data, err := json.Marshal(note)
	if err != nil {
		return
	}

	r, err := http.Post(n.Config["url"], "application/json",
		strings.NewReader(string(data)))
	if err == nil {
		defer r.Body.Close()
		if r.StatusCode < 200 || r.StatusCode >= 300 {
			err = errors.New(r.Status)
		}
	}
	return
}

func (n notifier) notify(note notification, resq chan<- bool) {
	defer func() { resq <- true }()

	for i := 0; i < max_retries; i++ {
		if err := notifyFuns[n.Driver](n, note); err == nil {
			break
		} else {
			time.Sleep(1 * time.Second)
			log.Printf("Retrying notification %s due to %v", n.Name, err)
		}
	}
}

func loadNotifiers() ([]notifier, error) {
	notifiers := []notifier{}

	f, err := os.Open("notify.json")
	if err != nil {
		return notifiers, err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	if err = d.Decode(&notifiers); err != nil {
		return notifiers, err
	}

	for _, v := range notifiers {
		if _, ok := notifyFuns[v.Driver]; !ok {
			log.Fatalf("Unknown driver '%s' in '%s'", v.Driver, v.Name)
		}
	}

	return notifiers, nil
}
