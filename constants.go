package mux

import "errors"

const (
	HEADER_SIZE = 32

	CMD_MESSAGE      = 1
	CMD_NEW_STREAM   = 2
	CMD_CLOSE_STREAM = 3
	CMD_OK           = 14
	CMD_FAIL         = 15
)

var (
	ErrNewStreamReqFail = errors.New("new stream request failed")
	ErrNoResponse       = errors.New("no response")
	ErrUnknown          = errors.New("unknown response")
	ErrStreamNotFound   = errors.New("stream not found")
)
