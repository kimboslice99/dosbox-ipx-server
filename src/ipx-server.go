package main

import (
	"flag"
	"strconv"
	"sync"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
)

var clients sync.Map
var port int = 1900
var serverAddress = ""

type ipxServer struct {
}

func (s *ipxServer) OnConnect(c *connection.Connection) {
	address := c.PeerAddr()
	client, ok := clients.Load(address)
	if ok {
		client.(*connection.Connection).Close()
	}
	clients.Store(address, c)
	log.Info("Connected: ", address)
}

func (s *ipxServer) OnMessage(c *connection.Connection, ctx interface{}, data []byte) (out interface{}) {
	header := IPXHeader{}
	header.fromBytes(data)

	if header.Dest.Socket == 0x2 && header.Dest.Host == 0x0 {
		// registration
		header.CheckSum = 0xffff
		header.Length = 30
		header.TransControl = 0
		header.PType = 0

		header.Dest.Network = 0
		header.Dest.setAddress(c.PeerAddr())
		header.Dest.Socket = 0x2

		header.Src.Network = 1
		header.Src.setAddress(serverAddress)
		header.Src.Socket = 0x2

		return header.toBytes()
	} else if header.Dest.Host == 0xffffffff {
		// broadcast
		clients.Range(func(address, dest interface{}) bool {
			if c != dest {
				dest.(*connection.Connection).Send(data)
			}
			return true
		})
	} else {
		dest, ok := clients.Load(header.Dest.Address())
		if ok {
			dest.(*connection.Connection).Send(data)
		}
	}

	return nil
}

func (s *ipxServer) OnClose(c *connection.Connection) {
	address := c.PeerAddr()
	clients.Delete(address)
	log.Info("Disconnected: ", address)
}

func main() {
	var loops int

	handler := new(ipxServer)

	flag.IntVar(&port, "port", port, "server port")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	server, err := gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops),
		gev.Protocol(&IPXProtocol{}),
	)

	if err != nil {
		panic(err)
	}

	log.Info("Server started on port ", port)
	serverAddress = "127.0.0.1:" + strconv.Itoa((port))
	server.Start()
}
