package tools

import "errors"

var ErrTimeout = errors.New("timeout")

var ErrClosedConnection = errors.New("closed connection")
