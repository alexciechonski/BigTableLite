package server

import (
	"net"

	"github.com/grpc-ecosystem/go-grpc-prometheus" 
	"google.golang.org/grpc"
)

func NewGRPCServer() *grpc.Server {
	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	grpc_prometheus.Register(s)

	return s
}

func NewListener(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}