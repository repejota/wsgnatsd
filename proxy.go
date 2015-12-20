// Copyright 2013-2015 Apcera Inc. All rights reserved.

package main

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/nats-io/gnatsd/server"
	"golang.org/x/net/websocket"
)

// StartWebsocketServer will enable the Websocket port.
func StartWebsocketServer(opts server.Options, port int) {
	hp := fmt.Sprintf("%s:%d", opts.Host, port)
	server.Noticef("Listening for websocket client connections on %s", hp)

	wsHandler := func(ws *websocket.Conn) {
		sp := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
		conn, _ := net.Dial("tcp", sp)
		go io.Copy(conn, ws)
		io.Copy(ws, conn)
	}

	http.Handle("/", websocket.Handler(wsHandler))

	server.Noticef("wsgnatsd is ready")

	go func() {
		e := http.ListenAndServe(hp, nil)
		if e != nil {
			server.Fatalf("Error listening on port: %s, %q", hp, e)
			return
		}
	}()
}
