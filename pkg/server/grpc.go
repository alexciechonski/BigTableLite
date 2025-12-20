package server

import (
	"net"

	"google.golang.org/grpc"
)

func NewGRPCServer() *grpc.Server {
	return grpc.NewServer()
}

func NewListener(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}
