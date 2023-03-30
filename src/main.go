package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const port = "1900"

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"binary"},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var ipxHandler = &IpxHandler{
	serverAddress: "127.0.0.1:" + port,
}

func getRoom(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		return ""
	}
	if parts[1] != "ipx" {
		return ""
	}
	return parts[2]
}

func ipxWebSocket(w http.ResponseWriter, r *http.Request) {
	room := getRoom(r)
	if len(room) == 0 {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ipxHandler.OnConnect(conn, room)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		ipxHandler.OnMessage(conn, room, data)
	}

	ipxHandler.OnClose(conn, room)
	conn.Close()
}

var cert string
var key string

func main() {
	flag.StringVar(&cert, "c", "", ".cert file")
	flag.StringVar(&key, "k", "", ".key file")
	flag.Parse()
	http.HandleFunc("/ipx/", ipxWebSocket)
	if len(cert) == 0 || len(key) == 0 {
		log.Println(".cert or .key file is not provided, disabling TLS")
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	} else if err := http.ListenAndServeTLS(":"+port, cert, key, nil); err != nil {
		log.Fatal(err)
	}
}
