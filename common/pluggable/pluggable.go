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
	"os"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	reflectV1Alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/polarismesh/polaris/common/log"
)

const (
	// envVarPolarisPluggableFolder
	envVarPolarisPluggableFolder string = "POLARIS_PLUGGABLE_SOCKETS_FOLDER"
	// defaultPolarisPluggablePath
	defaultPolarisPluggablePath = "/tmp/polaris-pluggable-sockets"
)

// onFinishedCallback is a callback to be called when a plugin is finished.
type onFinishedCallback func(name string, dialer GRPCConnectionDialer)

// onFinished
var onFinished = make(map[string]onFinishedCallback)

// AddOnFinished adds a callback to be called when a plugin is finished.
func AddOnFinished(serviceDesc string, cb onFinishedCallback) {
	_, ok := onFinished[serviceDesc]
	if ok {
		log.Fatalf("onFinished callback for %s already exists", serviceDesc)
	}
	onFinished[serviceDesc] = cb
}

// Discovery discovers all the plugins.
func Discovery(ctx context.Context) error {
	services, err := discovery(ctx)
	if err != nil {
		return err
	}
	finished(services)
	return nil
}

// finished calls the onFinished callback for the given services.
func finished(services []*pluginService) {
	for _, svc := range services {
		cb, ok := onFinished[svc.protoRef]
		if !ok {
			continue
		}

		cb(svc.name, svc.dialer)
		log.Infof("discovered pluggable component service: %s", svc.protoRef)
	}
}

// pluginService is a plugin service.
type pluginService struct {
	name     string
	protoRef string
	dialer   GRPCConnectionDialer
}

// discovery discovers all the plugins.
func discovery(ctx context.Context) ([]*pluginService, error) {
	sockFolder := socketFolder()
	files, err := pluginFiles(sockFolder)
	if err != nil {
		return nil, err
	}

	var services []*pluginService
	for _, dirEntry := range files {
		if dirEntry.IsDir() {
			continue
		}

		var discoveredServices []*pluginService
		discoveredServices, err = trySingleSocket(ctx, dirEntry, sockFolder)

		// skip non-socket files.
		if err == errNotSocket {
			continue
		}

		// return error if any other error occurs.
		if err != nil {
			return nil, err
		}

		services = append(services, discoveredServices...)
	}
	return services, nil
}

// trySingleSocket tries to discover plugins in a single socket.
func trySingleSocket(ctx context.Context, entry os.DirEntry, socketsFolder string) ([]*pluginService, error) {
	socket, err := socketName(entry)
	if err != nil {
		return nil, err
	}

	socketFullPath := filepath.Join(socketsFolder, socket)
	reflectClient, cleanup, err := dialServerReflection(ctx, socketFullPath)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	services, err := reflectClient.ListServices()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list plugin: %s's services", socket)
	}

	socketNameWithoutExt := strings.Trim(socket, filepath.Ext(socket))
	dialer := socketDialer(socketFullPath, grpc.WithBlock(), grpc.FailOnNonTempDialError(true))

	var pluginServices []*pluginService
	for _, svc := range services {
		pluginServices = append(pluginServices, &pluginService{
			protoRef: svc,
			dialer:   dialer,
			name:     socketNameWithoutExt,
		})
	}

	return pluginServices, nil
}

// dialServerReflection dials the server reflection service, returning the client and a cleanup function.
func dialServerReflection(ctx context.Context, socket string) (*grpcreflect.Client, func(), error) {
	conn, err := SocketDialContext(ctx, socket, grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}

	reflectClient := grpcreflect.NewClientV1Alpha(ctx, reflectV1Alpha.NewServerReflectionClient(conn))
	return reflectClient, reflectionConnectionCleanup(conn, reflectClient), nil
}

// reflectionConnectionCleanup closes the reflection connection.
func reflectionConnectionCleanup(conn *grpc.ClientConn, client *grpcreflect.Client) func() {
	return func() {
		client.Reset()
		if err := conn.Close(); err != nil {
			log.Errorf("error closing grpc reflection connection: %v", err)
		}
	}
}

// socketFolder returns the socket folder path specified by the environment variable.
func socketFolder() string {
	if value, ok := os.LookupEnv(envVarPolarisPluggableFolder); ok {
		return value
	}
	return defaultPolarisPluggablePath
}
