package main

import (
	"bytes"
	"encoding/binary"

	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/ringbuffer"
	"github.com/gobwas/pool/pbytes"
)

const headerLength = 4

type IPXProtocol struct{}

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

func (d *IPXProtocol) UnPacket(c *connection.Connection, buffer *ringbuffer.RingBuffer) (interface{}, []byte) {
	if buffer.VirtualLength() > headerLength {
		buf := pbytes.GetLen(headerLength)
		defer pbytes.Put(buf)
		_, _ = buffer.VirtualRead(buf)
		packetLen := binary.BigEndian.Uint16(buf[2:4])
		buffer.VirtualRevert()

		if buffer.VirtualLength() >= int(packetLen) {
			ret := make([]byte, packetLen)
			_, _ = buffer.VirtualRead(ret)
			buffer.VirtualFlush()

			return nil, ret
		}
	}

	return nil, nil
}

func (d *IPXProtocol) Packet(c *connection.Connection, data interface{}) []byte {
	return data.([]byte)
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
