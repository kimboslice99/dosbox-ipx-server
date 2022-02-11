package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Allenxuxu/gev"
	"github.com/Allenxuxu/gev/connection"
	"github.com/Allenxuxu/gev/log"
)

var clients sync.Map
var port int = 1900
var serverAddress = ""
var idleTimeMin int64 = 15
var sessionTestIntervalMin int64 = 5
var testSession = ""
var lastTimeUsedMs = time.Now().UnixMilli()

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
	atomic.StoreInt64(&lastTimeUsedMs, time.Now().UnixMilli())

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
	flag.Int64Var(&idleTimeMin, "timeout", idleTimeMin, "terminate after (min) ")
	flag.StringVar(&testSession, "test", "", "use this url to test session")
	flag.Parse()

	idleTicker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	go func() {
		var sessionTestedAtMs int64 = 0
		sessionTestIntervalMs := sessionTestIntervalMin * 60 * 1000
		idleTimeMs := idleTimeMin * 60 * 1000
		for {
			select {
			case <-idleTicker.C:
				now := time.Now().UnixMilli()
				usedMs := atomic.LoadInt64(&lastTimeUsedMs)
				if now-usedMs > idleTimeMs {
					log.Info("Idle... Exiting...")
					syscall.Kill(syscall.Getpid(), syscall.SIGINT)
				}
				if len(testSession) > 0 && (now-sessionTestedAtMs) > sessionTestIntervalMs {
					sessionTestedAtMs = now
					log.Info("Test session with url: ", testSession)
					resp, err := http.Get(testSession)
					if err != nil {
						log.Info(err)
						syscall.Kill(syscall.Getpid(), syscall.SIGINT)
					} else {
						body, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							log.Info(err)
							syscall.Kill(syscall.Getpid(), syscall.SIGINT)
						} else if string(body) != "{\"code\":200,\"alive\":true}" {
							log.Info("Not live, exiting... status: ", string(body))
							syscall.Kill(syscall.Getpid(), syscall.SIGINT)
						} else {
							log.Info("Session is live...")
						}
					}
				}
			case <-quit:
				idleTicker.Stop()
				return
			}
		}
	}()

	server, err := gev.NewServer(handler,
		gev.Network("tcp"),
		gev.Address(":"+strconv.Itoa(port)),
		gev.NumLoops(loops),
		gev.Protocol(&IPXProtocol{}),
	)

	if err != nil {
		panic(err)
	}

	log.Info("Server started on port: ", port, ", timeout: ", idleTimeMin, " min",
		", test-session: ", testSession)
	serverAddress = "127.0.0.1:" + strconv.Itoa((port))
	server.Start()
	close(quit)
}
