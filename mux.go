package mux

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

type Mux struct {
	await   chan []byte
	cb      func(*Stream)
	conn    io.ReadWriteCloser
	mu      sync.Mutex
	nextID  atomic.Uint64
	streams sync.Map
}

func NewMux(conn io.ReadWriteCloser, cb func(s *Stream)) *Mux {
	return &Mux{
		await:   make(chan []byte),
		cb:      cb,
		conn:    conn,
		mu:      sync.Mutex{},
		nextID:  atomic.Uint64{},
		streams: sync.Map{},
	}
}

func (m *Mux) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := m.conn.Close()

	m.streams.Range(func(key, value any) bool {
		s := value.(*Stream)
		close(s.incoming)
		return true
	})

	return err
}

func (m *Mux) NewStream() (*Stream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	streamID := m.nextID.Add(1)
	stream := NewStream(streamID, m)

	header := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint16(header[0:2], CMD_NEW_STREAM)
	binary.LittleEndian.PutUint64(header[2:10], streamID)
	_, err := m.Send(header)
	if err != nil {
		return nil, err
	}

	v, ok := <-m.await
	if !ok {
		return nil, ErrNoResponse
	}

	cmd := binary.LittleEndian.Uint16(v[0:2])

	switch cmd {
	case CMD_OK:
		m.streams.Store(streamID, stream)
		return stream, nil
	case CMD_FAIL:
		return nil, ErrNewStreamReqFail
	default:
		log.Println("received unknown", cmd)
		return nil, ErrUnknown
	}
}

func (m *Mux) Send(payload []byte) (int, error) {
	return m.conn.Write(payload)
}

func (m *Mux) Receive() {
	defer func() {
		err := recover()
		log.Println(err)
	}()

	header := make([]byte, HEADER_SIZE)
outer:
	for {
		_, err := io.ReadFull(m.conn, header)
		if err != nil {
			panic(fmt.Errorf("unable to read header %v", err))
		}

		cmd := binary.LittleEndian.Uint16(header[:2])
		streamID := binary.LittleEndian.Uint64(header[2:10])
		length := binary.LittleEndian.Uint64(header[10:18])

		switch cmd {
		case CMD_MESSAGE:
			payload := make([]byte, length)
			_, err = io.ReadFull(m.conn, payload)
			if err != nil {
				panic(fmt.Errorf("unable to read payload %w", err))
			}

			v, exist := m.streams.Load(streamID)
			if !exist {
				log.Println("error stream does not exist", streamID)
				continue outer
			}

			stream := v.(*Stream)
			stream.mu.Lock()

			go func() {
				stream.incoming <- payload
				stream.mu.Unlock()
			}()

		case CMD_NEW_STREAM:
			log.Println("new stream", streamID)

			m.mu.Lock()

			if m.cb != nil {
				m.nextID.Store(streamID)
				s := NewStream(streamID, m)
				m.streams.Store(streamID, s)
				go m.cb(s)

				m.Ok(header)
			} else {
				m.Fail(header)
			}

			m.mu.Unlock()

		case CMD_CLOSE_STREAM:
			log.Println("stream closed", streamID)

			v, ok := m.streams.LoadAndDelete(streamID)
			if !ok {
				continue
			}

			s := v.(*Stream)
			close(s.incoming)
			s.incoming = nil
			s.cancel()

		case CMD_OK:
			cloned := header
			m.await <- cloned
			runtime.Gosched()

		case CMD_FAIL:
			cloned := header
			m.await <- cloned
			runtime.Gosched()

		default:
			log.Println("unknown command", cmd)
		}
	}
}

type Stream struct {
	ctx    context.Context
	cancel context.CancelFunc

	id       uint64
	incoming chan []byte
	m        *Mux
	mu       sync.Mutex
}

func NewStream(id uint64, m *Mux) *Stream {
	ctx, cancel := context.WithCancel(context.Background())

	return &Stream{
		ctx:      ctx,
		cancel:   cancel,
		id:       id,
		incoming: make(chan []byte, 1),
		m:        m,
	}
}

func (s *Stream) ID() uint64 {
	return s.id
}

func (s *Stream) Context() context.Context {
	return s.ctx
}

func (s *Stream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	s.m.streams.Delete(s.ID())

	header := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint16(header[0:2], CMD_CLOSE_STREAM)
	binary.LittleEndian.PutUint64(header[2:10], s.ID())

	_, err := s.m.Send(header)
	if err != nil {
		return
	}

	close(s.incoming)
	s.incoming = nil
}

func (s *Stream) Read() <-chan []byte {
	return s.incoming
}

func (s *Stream) Send(payload []byte) error {
	s.m.mu.Lock()
	defer s.m.mu.Unlock()

	header := make([]byte, HEADER_SIZE)
	binary.LittleEndian.PutUint16(header[0:2], CMD_MESSAGE)
	binary.LittleEndian.PutUint64(header[2:10], s.ID())
	binary.LittleEndian.PutUint64(header[10:18], uint64(len(payload)))

	_, err := s.m.Send(header)
	if err != nil {
		return err
	}

	_, err = s.m.Send(payload)
	if err != nil {
		return err
	}

	return nil
}

func (m *Mux) Ok(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], CMD_OK)
	_, err := m.Send(b)
	if err != nil {
		log.Println("error unable to send ok response")
	}
}

func (m *Mux) Fail(b []byte) {
	binary.LittleEndian.PutUint16(b[0:2], CMD_FAIL)
	_, err := m.Send(b)
	if err != nil {
		log.Println("error unable to send ok response")
	}
}
