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
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats"
)

// Version is the version number
var Version string

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 2000
)

var n *nats.Conn

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
}

// Publish a message to a subject over http POST
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

	//Start ticker for sending ping
	ticker := time.NewTicker(pingPeriod)
	go func() {
		for {
			select {
			case <-ticker.C:
				err = conn.WriteMessage(websocket.PingMessage, []byte{})
				if err != nil {
					log.Println("Error writing ping", err)
					if n != nil {
						log.Printf("Close NATS ticker")
						n.Close()
					}
					if conn != nil {
						log.Printf("Close Conn ticker")
						conn.Close()
					}
					if ticker != nil {
						log.Printf("Stop ticker ticker")
						ticker.Stop()
					}
					break
				}
			}
		}
	}()

	messageType, payload, err := conn.ReadMessage()
	subject := string(payload)
	if err != nil {
		log.Println("Error reading a message", err)
		if n != nil {
			log.Printf("Close NATS error reading first mesage")
			n.Close()
		}
		if conn != nil {
			log.Printf("Close Conn error reading first mesage")
			conn.Close()
		}
		if ticker != nil {
			log.Printf("Stop ticker error reading first mesage")
			ticker.Stop()
		}
		return
	}

	// Subscribe to nats messages
	log.Println("Subscribed to", subject)
	n.Subscribe(subject, func(m *nats.Msg) {
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		err = conn.WriteMessage(messageType, m.Data)
		if err != nil {
			log.Println("Error writing a message", err)
			if n != nil {
				log.Printf("Close NATS error writing mesage")
				n.Close()
			}
			if conn != nil {
				log.Printf("Close Conn error writing mesage")
				conn.Close()
			}
			if ticker != nil {
				log.Printf("Stop ticker error writing mesage")
				ticker.Stop()
			}
		}
	})

	// Start read loop so we can publish a message to a subject over websocket
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			if n != nil {
				log.Printf("Close NATS read")
				n.Close()
			}
			if conn != nil {
				log.Printf("Close Conn read")
				conn.Close()
			}
			if ticker != nil {
				log.Printf("Stop ticker read")
				ticker.Stop()
			}
			break
		}

		mes := string(message)
		index := strings.Index(mes, " ")

		if index > -1 {
			subject := mes[0:index]
			data := mes[index:len(mes)]

			n.Publish(subject, []byte(data))
		}
	}

	runtime.Goexit()
}

// Handle handles the initial HTTP connection.
//
func Handle(w http.ResponseWriter, r *http.Request) {
	n, _ = nats.Connect("nats://10.240.0.22:4222")
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
