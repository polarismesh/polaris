/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package pluggable

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// SetupTimeout is the timeout for the setup of a plugin.
	// We can change it when some plugins need more time to set up.
	SetupTimeout = 5 * time.Second
)

// GRPCConnectionDialer is a function that dials a gRPC connection.
type GRPCConnectionDialer func(ctx context.Context, name string) (*grpc.ClientConn, error)

// SocketDialContext dials a gRPC connection using a socket.
func SocketDialContext(ctx context.Context, socket string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	unixSock := "unix://" + socket
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dialCtx, cancel := context.WithTimeout(ctx, SetupTimeout)
	defer cancel()

	grpcConn, err := grpc.DialContext(dialCtx, unixSock, opts...)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, fmt.Errorf("timed out dialing socket %s", socket)
		}

		return nil, fmt.Errorf("failed to dial socket %s: %w", socket, err)
	}

	return grpcConn, nil
}

// GRPCPluginClient defines the interface for a grpc plugin client, so all grpc plugin service must implement the
// Ping method, and polaris server will call it to check the health of the plugin.
type GRPCPluginClient interface {
	Ping(ctx context.Context, in *api.PingRequest, opts ...grpc.CallOption) (*api.PongResponse, error)
}

// GRPCConnector is a connector that uses underlying gRPC protocol for common operations.
type GRPCConnector struct {
	// Client is the client that is used to communicate with the plugin, exposed for plugin logic layer.
	Client        GRPCPluginClient
	dialer        GRPCConnectionDialer
	conn          *grpc.ClientConn
	clientFactory func(grpc.ClientConnInterface) GRPCPluginClient
}

// Dial opens a grpcConnection and creates a new client instance.
func (g *GRPCConnector) Dial(ctx context.Context, name string) error {
	grpcConn, err := g.dialer(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "unable to open GRPC connection using the dialer")
	}
	g.conn = grpcConn

	g.Client = g.clientFactory(grpcConn)
	return nil
}

// socketDialer creates a dialer for the given socket.
func socketDialer(socket string, additionalOpts ...grpc.DialOption) GRPCConnectionDialer {
	return func(ctx context.Context, name string) (*grpc.ClientConn, error) {
		return SocketDialContext(ctx, socket, additionalOpts...)
	}
}

// Ping pings the grpc component.
// It uses "WaitForReady" avoiding failing in transient failures.
func (g *GRPCConnector) Ping(ctx context.Context) error {
	_, err := g.Client.Ping(ctx, &api.PingRequest{}, grpc.WaitForReady(true))
	return err
}

// Close closes the underlying gRPC connection.
func (g *GRPCConnector) Close() error {
	return g.conn.Close()
}

// NewGRPCConnectorWithDialer creates a new grpc connector for the given client factory and dialer.
func NewGRPCConnectorWithDialer(
	dialer GRPCConnectionDialer, factory func(grpc.ClientConnInterface) GRPCPluginClient) *GRPCConnector {
	return &GRPCConnector{
		dialer:        dialer,
		clientFactory: factory,
	}
}
