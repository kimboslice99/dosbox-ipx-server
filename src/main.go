package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const port = "1900"

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"binary"},
	CheckOrigin: func(r *http.Request) bool {
		// log connections to console
		line := fmt.Sprintf("Connection from %s using host %s", r.RemoteAddr, r.Host)
		fmt.Println(line)
		// if we havent been given a hosts param
		if hosts == "" {
			return true
		}
		hostsArray := strings.Split(hosts, `;`)
		fmt.Println("Hosts array:", hostsArray)
		// loop over our given allowed hosts, return true if we find one
		for i := 0; i < len(hostsArray); i++ {
			if hostsArray[i] == r.Host {
				return true
			}
		}
		denyline := fmt.Sprintf("Denied access to %s, %s was not in allowed hosts %s", r.RemoteAddr, r.Host, hostsArray)
		fmt.Println(denyline)
		return false
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
var hosts string

func main() {
	flag.StringVar(&cert, "c", "", ".cert file")
	flag.StringVar(&key, "k", "", ".key file")
	flag.StringVar(&hosts, "h", "", "Allowed hostnames")
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
