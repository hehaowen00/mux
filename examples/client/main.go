package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/hehaowen00/mux"
)

func main() {
	conn, err := net.Dial("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	mux := mux.NewMux(conn, nil)
	defer mux.Close()

	go mux.Receive()

	s1, err := mux.NewStream()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-s1.Context().Done():
				return
			case msg, ok := <-s1.Read():
				if !ok {
					return
				}

				log.Println(s1.ID(), string(msg))

				if string(msg) == "ping" {
					_ = s1.Send([]byte("pong"))
				}

				if string(msg) == "pong" {
					_ = s1.Send([]byte("ping"))
				}

				time.Sleep(time.Second)
			}
		}
	}()

	s2, err := mux.NewStream()
	if err != nil {
		panic(err)
	}

	go func() {
		i := 0
		for {
			select {
			case <-s2.Context().Done():
				return
			case msg, ok := <-s2.Read():
				if !ok {
					return
				}

				log.Println(s2.ID(), string(msg))

				if string(msg) == "ping" {
					_ = s2.Send([]byte("pong"))
				}

				if string(msg) == "pong" {
					_ = s2.Send([]byte("ping"))
				}

				i++
				if i == 10 {
					s2.Close()
				}

				time.Sleep(time.Second)
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
