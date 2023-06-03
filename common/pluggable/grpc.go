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
	"time"

	"github.com/pkg/errors"
	"github.com/polaris-contrib/polaris-server-remote-plugin-common/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SetupTimeout is the timeout for setting up a connection.
const SetupTimeout = 5 * time.Second

// GRPCConnectionDialer defines the function to dial a grpc connection.
type GRPCConnectionDialer func(ctx context.Context, name string) (*grpc.ClientConn, error)

// SocketDialContext dials a gRPC connection using a socket.
func SocketDialContext(ctx context.Context, socket string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	unixSock := "unix://" + socket

	// disable TLS as default when using socket
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	dialCtx, cancel := context.WithTimeout(ctx, SetupTimeout)
	defer cancel()

	grpcConn, err := grpc.DialContext(dialCtx, unixSock, opts...)
	if err != nil {
		return nil, err
	}

	return grpcConn, nil
}

// GRPCPluginClient defines the interface for a gRPC plugin client，
// polaris server will call the plugin client's Ping method to check if the plugin is alive.
type GRPCPluginClient interface {
	// Ping checks if the plugin is alive.
	Ping(ctx context.Context, in *api.PingRequest, opts ...grpc.CallOption) (*api.PongResponse, error)
}

// GRPCConnector defines the connector for a gRPC plugin.
type GRPCConnector struct {
	// pluginClient is the client that is used to communicate with the plugin, exposed for plugin logic layer.
	pluginClient GRPCPluginClient
	// dialer use to dial a grpc connection.
	dialer GRPCConnectionDialer
	// conn is the grpc client connection.
	conn *grpc.ClientConn
	// clientFactory is the factory to create a grpc client.
	clientFactory func(grpc.ClientConnInterface) GRPCPluginClient
}

// NewGRPCConnectorWithDialer creates a new grpc connector for the given client factory and dialer.
func NewGRPCConnectorWithDialer(
	dialer GRPCConnectionDialer, factory func(grpc.ClientConnInterface) GRPCPluginClient) *GRPCConnector {
	return &GRPCConnector{
		dialer:        dialer,
		clientFactory: factory,
	}
}

// Dial init a grpc connection to the plugin server and create a grpc client.
func (g *GRPCConnector) Dial(ctx context.Context, name string) error {
	conn, err := g.dialer(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "unable to open GRPC connection using the dialer")
	}

	g.conn = conn
	g.pluginClient = g.clientFactory(conn)
	return nil
}

// PluginClient returns the grpc client.
func (g *GRPCConnector) PluginClient() GRPCPluginClient {
	return g.pluginClient
}

// socketDialer returns a GRPCConnectionDialer that dials a grpc connection using a socket.
func socketDialer(socket string, opts ...grpc.DialOption) GRPCConnectionDialer {
	return func(ctx context.Context, name string) (*grpc.ClientConn, error) {
		return SocketDialContext(ctx, socket, opts...)
	}
}

// Ping checks if the plugin is alive.
func (g *GRPCConnector) Ping(ctx context.Context) error {
	_, err := g.pluginClient.Ping(ctx, &api.PingRequest{}, grpc.WaitForReady(true))
	return err
}

// Close closes the underlying gRPC connection.
func (g *GRPCConnector) Close() error {
	return g.conn.Close()
}
