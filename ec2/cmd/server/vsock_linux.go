//go:build linux

package main

import (
	"context"
	"net"

	"github.com/mdlayher/vsock"
)

func dialEnclave(ctx context.Context, cid uint32, port uint32) (net.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return vsock.Dial(cid, port, nil)
}
