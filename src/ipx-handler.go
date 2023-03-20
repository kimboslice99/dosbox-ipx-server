package main

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/gorilla/websocket"
)

type IpxHandler struct {
	clients       sync.Map
	serverAddress string
}

func (handler *IpxHandler) OnConnect(c *websocket.Conn) {
	address := c.RemoteAddr().String()
	client, ok := handler.clients.Load(address)
	if ok {
		client.(*websocket.Conn).Close()
	}
	handler.clients.Store(address, c)
}

func (handler *IpxHandler) OnMessage(c *websocket.Conn, data []byte) {
	header := IPXHeader{}
	header.fromBytes(data)

	if header.Dest.Socket == 0x2 && header.Dest.Host == 0x0 {
		// registration
		header.CheckSum = 0xffff
		header.Length = 30
		header.TransControl = 0
		header.PType = 0

		header.Dest.Network = 0
		header.Dest.setAddress(c.RemoteAddr().String())
		header.Dest.Socket = 0x2

		header.Src.Network = 1
		header.Src.setAddress(handler.serverAddress)
		header.Src.Socket = 0x2

		c.WriteMessage(websocket.BinaryMessage, header.toBytes())
	} else if header.Dest.Host == 0xffffffff {
		// broadcast
		handler.clients.Range(func(address, dest interface{}) bool {
			if c != dest {
				dest.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, data)
			}
			return true
		})
	} else {
		dest, ok := handler.clients.Load(header.Dest.Address())
		if ok {
			dest.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, data)
		}
	}
}

func (handler *IpxHandler) OnClose(c *websocket.Conn) {
	address := c.RemoteAddr().String()
	handler.clients.Delete(address)
}

type IPXTransport struct {
	Network uint32
	Host    uint32
	Port    uint16
	Socket  uint16
}

type IPXHeader struct {
	CheckSum     uint16
	Length       uint16
	TransControl uint8
	PType        uint8

	Dest IPXTransport
	Src  IPXTransport
}

func (header *IPXHeader) toBytes() []byte {
	var payload bytes.Buffer
	binary.Write(&payload, binary.BigEndian, header.CheckSum)
	binary.Write(&payload, binary.BigEndian, header.Length)
	binary.Write(&payload, binary.BigEndian, header.TransControl)
	binary.Write(&payload, binary.BigEndian, header.PType)

	binary.Write(&payload, binary.BigEndian, header.Dest.Network)
	binary.Write(&payload, binary.BigEndian, header.Dest.Host)
	binary.Write(&payload, binary.BigEndian, header.Dest.Port)
	binary.Write(&payload, binary.BigEndian, header.Dest.Socket)

	binary.Write(&payload, binary.BigEndian, header.Src.Network)
	binary.Write(&payload, binary.BigEndian, header.Src.Host)
	binary.Write(&payload, binary.BigEndian, header.Src.Port)
	binary.Write(&payload, binary.BigEndian, header.Src.Socket)
	return payload.Bytes()
}

func (header *IPXHeader) fromBytes(data []byte) {
	payload := bytes.NewReader(data)
	binary.Read(payload, binary.BigEndian, &header.CheckSum)
	binary.Read(payload, binary.BigEndian, &header.Length)
	binary.Read(payload, binary.BigEndian, &header.TransControl)
	binary.Read(payload, binary.BigEndian, &header.PType)

	binary.Read(payload, binary.BigEndian, &header.Dest.Network)
	binary.Read(payload, binary.BigEndian, &header.Dest.Host)
	binary.Read(payload, binary.BigEndian, &header.Dest.Port)
	binary.Read(payload, binary.BigEndian, &header.Dest.Socket)

	binary.Read(payload, binary.BigEndian, &header.Src.Network)
	binary.Read(payload, binary.BigEndian, &header.Src.Host)
	binary.Read(payload, binary.BigEndian, &header.Src.Port)
	binary.Read(payload, binary.BigEndian, &header.Src.Socket)
}
