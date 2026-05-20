//go:build !linux

package main

import (
	"context"
	"errors"
	"net"
)

func dialEnclave(_ context.Context, _ uint32, _ uint32) (net.Conn, error) {
	return nil, errors.New("vsock is only available on linux parent instances")
}
