package pluggable

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/polarismesh/polaris/common/log"
)

// GRPCConnectionDialer grpc connection dialer.
type GRPCConnectionDialer = func(ctx context.Context, name string) (*grpc.ClientConn, error)

// GRPCConnector is a connector that uses underlying gRPC protocol for common operations.
type GRPCConnector struct {
	// Context is the component shared context
	Context context.Context
	// Cancel is used for cancelling inflight requests
	Cancel context.CancelFunc
	// Client is the proto client.
	Client        api.PluginClient
	dialer        GRPCConnectionDialer
	conn          *grpc.ClientConn
	clientFactory func(grpc.ClientConnInterface) api.PluginClient
}

// socketDialer creates a dialer for the given socket.
func socketDialer(socket string, additionalOpts ...grpc.DialOption) GRPCConnectionDialer {
	return func(ctx context.Context, name string) (*grpc.ClientConn, error) {
		return SocketDial(ctx, socket, additionalOpts...)
	}
}

// SocketDial creates a grpc connection using the given socket.
func SocketDial(ctx context.Context, socket string, additionalOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	udsSocket := "unix:///" + socket
	log.Debugf("using socket defined at '%s'", udsSocket)
	additionalOpts = append(additionalOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dallierCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	grpcConn, err := grpc.DialContext(dallierCtx, udsSocket, additionalOpts...)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, errors.Wrapf(err, "'%s' is not responding", socket)
		}
		return nil, errors.Wrapf(err, "unable to open GRPC connection using socket '%s'", udsSocket)
	}

	return grpcConn, nil
}

// Dial opens a grpcConnection and creates a new client instance.
func (g *GRPCConnector) Dial(name string) error {
	grpcConn, err := g.dialer(g.Context, name)
	if err != nil {
		return errors.Wrapf(err, "unable to open GRPC connection using the dialer")
	}
	g.conn = grpcConn

	g.Client = g.clientFactory(grpcConn)

	return nil
}

// Ping pings the grpc component.
// It uses "WaitForReady" avoiding failing in transient failures.
func (g *GRPCConnector) Ping() error {
	_, err := g.Client.Ping(g.Context, &api.PingRequest{}, grpc.WaitForReady(true))
	return err
}

// Close closes the underlying gRPC connection and cancel all inflight requests.
func (g *GRPCConnector) Close() error {
	g.Cancel()

	return g.conn.Close()
}

// NewGRPCConnectorWithDialer creates a new grpc connector for the given client factory and dialer.
func NewGRPCConnectorWithDialer(dialer GRPCConnectionDialer, factory func(grpc.ClientConnInterface) api.PluginClient) *GRPCConnector {
	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCConnector{
		Context:       ctx,
		Cancel:        cancel,
		dialer:        dialer,
		clientFactory: factory,
	}
}
