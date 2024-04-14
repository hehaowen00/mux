package main

import (
	"log"
	"net"
	"time"

	"github.com/hehaowen00/mux"
)

func main() {
	l, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	log.Println("server started")

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			log.Println("new client")

			m := mux.NewMux(c, handleStream)
			go m.Receive()
		}()
	}
}

func handleStream(s *mux.Stream) {
	log.Println("new stream", s.ID())

	_ = s.Send([]byte("ping"))

	for msg := range s.Read() {
		log.Println(s.ID(), string(msg))

		if string(msg) == "ping" {
			_ = s.Send([]byte("pong"))
		}

		if string(msg) == "pong" {
			_ = s.Send([]byte("ping"))
		}

		time.Sleep(time.Second)
	}
}
