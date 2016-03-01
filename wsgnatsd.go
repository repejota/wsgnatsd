package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats"
)

// Version is the version number
var Version string

var n *nats.Conn

//var upgrader = websocket.Upgrader{} // use default options
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Publish a message to a subject
func post(subject string, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error publishing a message", err)
		return
	}
	n.Publish(subject, body)
	defer n.Close()
}

func get(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to a websocket connection", err)
		return
	}
	messageType, payload, err := conn.ReadMessage()
	subject := string(payload)
	if err != nil {
		log.Println("Error reading a message", err)
		return
	}
	// Subscribe to nats messages
	log.Println("Subscribed to", subject)
	n.Subscribe(subject, func(m *nats.Msg) {
		err = conn.WriteMessage(messageType, m.Data)
		if err != nil {
			log.Println("Error writing a message", err)
		}
	})
	runtime.Goexit()
}

// Handle handles the initial HTTP connection.
//
func Handle(w http.ResponseWriter, r *http.Request) {
	n, _ = nats.Connect(nats.DefaultURL)
	if r.Method == "GET" {
		get(w, r)
	} else if r.Method == "POST" {
		subject := strings.Replace(r.URL.Path, "/", "", 1)
		post(subject, r)
	} else {
		http.Error(w, "Method not allowed", 405)
		return
	}
}

func main() {
	var showVersion bool
	var addrHost string
	var addrPort int

	flag.BoolVar(&showVersion, "version", false, "Print version information.")
	flag.StringVar(&addrHost, "addr", "localhost", "Network host to listen on.")
	flag.IntVar(&addrPort, "port", 4223, "Port to listen on.")

	flag.Parse()

	// Show version and exit
	if showVersion {
		fmt.Println("wsgnatsd version", Version)
		os.Exit(0)
	}

	// Start server
	log.Println("Starting wsgnatsd version", Version)
	http.HandleFunc("/", Handle)
	log.Printf("Listening for client connections on %s:%d", addrHost, addrPort)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", addrHost, addrPort), nil)
	if err != nil {
		log.Fatal("wsgnatsd can't listen on", fmt.Sprintf("%s:%d", addrHost, addrPort), err)
	}
}
