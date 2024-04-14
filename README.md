# mux

`mux` is a simple stream multiplexing implementation written in go.

## usage

A mux can be instantiated over any type that implements the `io.ReadWriterCloser`
interface.

`NewMux` takes a new stream handler that will be called when the other party
sends a new stream request. This can be requested by either party at any
time.

If no stream handler callback is provided, calls to the `mux.NewStream` function
will fail when the other party requests.

```go
m := mux.NewMux(conn, newStreamHandler)
defer m.Close()

go m.Receive()

func newStreamHandler(s *Stream) {
    // ...
}
```

`mux.NewStream` creates a new stream on both sides of the connection.

```go
stream, err := m.NewStream()
if err != nil {
    // ...
}
```

`Stream.Read()` returns a read only channel which returns `[]byte`.

```go
for {
    select {
    case m, ok := <-stream.Read()
        // ...
    }
}
```

`Stream.Send([]byte)` sends a message to the client.

```go
err := stream.Send([]byte("hello world"))
```
