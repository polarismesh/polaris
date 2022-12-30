package pluggable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

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

// Discovery discovers plugins.
func Discovery(ctx context.Context) error {
	services, err := tryDiscoveryPlugins(ctx)
	if err != nil {
		return err
	}
	finished(services)
	return nil
}

// pluginService represents a service found by the discovery process.
type pluginService struct {
	name     string
	protoRef string
	dialer   GRPCConnectionDialer
}

// socketFolder returns the socket folder path specified by the environment variable.
func socketFolder() string {
	if value, ok := os.LookupEnv(envVarPolarisPluggableFolder); ok {
		return value
	}
	return defaultPolarisPluggablePath
}

// tryDiscoveryPlugins tries to discover plugins.
func tryDiscoveryPlugins(ctx context.Context) ([]pluginService, error) {
	sockFolder := socketFolder()
	_, err := os.Stat(sockFolder)
	if os.IsNotExist(err) {
		log.Infof("socket folder %s does not exist, skip plugin discovery", sockFolder)
		return nil, nil
	}

	if err != nil {
		log.Errorf("failed to stat socket folder %s: %v", sockFolder, err)
		return nil, err
	}

	var files []os.DirEntry
	files, err = os.ReadDir(sockFolder)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read socket folder %s", sockFolder)
	}

	var services []pluginService
	for _, dirEntry := range files {
		var discoveredServices []pluginService
		discoveredServices, err = trySingleSocket(ctx, dirEntry, sockFolder)
		if err != nil {
			return nil, err
		}
		services = append(services, discoveredServices...)
	}
	return services, nil
}

// trySingleSocket tries to discover plugins in a single socket.
func trySingleSocket(ctx context.Context, entry os.DirEntry, socketsFolder string) ([]pluginService, error) {
	if entry.IsDir() {
		return nil, nil
	}

	f, err := entry.Info()
	if err != nil {
		return nil, err
	}

	socket := filepath.Join(socketsFolder, f.Name())
	if !isSocket(f) {
		log.Infof("file %s is not a socket, skip", socket)
		return nil, nil
	}

	reflectClient, cleanup, err := dialServerReflection(ctx, socket)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	serviceList, err := reflectClient.ListServices()
	if err != nil {
		return nil, errors.Wrap(err, "unable to list services")
	}

	var services []pluginService
	sockName := f.Name()
	sockName = strings.TrimSuffix(sockName, filepath.Ext(sockName))
	dialer := socketDialer(socket, grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	for _, svc := range serviceList {
		services = append(services, pluginService{
			protoRef: svc,
			dialer:   dialer,
			name:     sockName,
		})
	}

	return services, nil
}

// dialServerReflection dials the server reflection service, returning the client and a cleanup function.
func dialServerReflection(ctx context.Context, socket string) (*grpcreflect.Client, func(), error) {
	conn, err := SocketDialContext(ctx, socket, grpc.WithBlock())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial reflection service with socket %s: %w", socket, err)
	}
	client := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	return client, reflectionConnectionCleanup(conn, client), nil
}

// reflectionConnectionCleanup closes the reflection connection.
func reflectionConnectionCleanup(conn *grpc.ClientConn, client *grpcreflect.Client) func() {
	return func() {
		client.Reset()
		err := conn.Close()

		if err != nil {
			log.Errorf("error closing grpc reflection connection: %v", err)
		}
	}
}

func finished(services []pluginService) {
	for _, svc := range services {
		cb, ok := onFinished[svc.protoRef]
		if !ok {
			continue
		}
		cb(svc.name, svc.dialer)
		log.Infof("discovered pluggable component service: ", svc.protoRef)
	}
}

// isSocket returns true if the file is a socket.
func isSocket(file os.FileInfo) bool {
	return file.Mode()&os.ModeSocket != 0
}
