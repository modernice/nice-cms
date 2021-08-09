package grpctest

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// NewServer returns a *grpc.Server and a dialer for that server. The server s
// is started with s.Serve using a *bufconn.Listener. If init is non-nil, it
// is called with the *grpc.Server before calling s.Serve.
//
//	_, dial := NewServer(func(s *grpc.Server) {
//		proto.RegisterFooServer(s, ...)
//	})
//	conn := dial()
//	defer conn.Close()
func NewServer(init func(*grpc.Server)) (*grpc.Server, func() *grpc.ClientConn) {
	lis := bufconn.Listen(1024)
	srv := grpc.NewServer()

	if init != nil {
		init(srv)
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			panic(fmt.Errorf("grpc test server: %v", err))
		}
	}()

	return srv, func() *grpc.ClientConn {
		conn, err := grpc.Dial("bufnet", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}))
		if err != nil {
			panic(fmt.Errorf("dial grpc test server: %v", err))
		}
		return conn
	}
}
