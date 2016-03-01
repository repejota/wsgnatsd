package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats"
)

// Version is the version number
var Version string

var addr = flag.String("addr", "localhost:4223", "http service address")
var n *nats.Conn

//var upgrader = websocket.Upgrader{} // use default options
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func post(subject string, r *http.Request) {
	// Publish messages
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		println("ERROR" + err.Error())
	}

	n.Publish(subject, body)
}

func get(subject string, w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	mt, _, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}
	// Subscribe to nats messages
	log.Println("Subscribed to " + subject)
	n.Subscribe(subject, func(m *nats.Msg) {
		err = c.WriteMessage(mt, m.Data)
		if err != nil {
			log.Println("write:", err)
		}
	})
	runtime.Goexit()
}

// Handle ...
func Handle(w http.ResponseWriter, r *http.Request) {
	n, _ = nats.Connect(nats.DefaultURL)
	subject := strings.Replace(r.URL.Path, "/", "", 1)

	if r.Method == "GET" {
		get(subject, w, r)
	} else if r.Method == "POST" {
		post(subject, r)
	} else {
		http.Error(w, "Method not allowed", 405)
		return
	}
}

func main() {
	flag.Parse()

	log.Println("Starting wsgnatsd version", Version)
	http.HandleFunc("/", Handle)

	log.Println("Listening for client connections on", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("wsgnatsd can't listen on", addr, err)
	}
}
