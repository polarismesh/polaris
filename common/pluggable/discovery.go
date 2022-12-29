package pluggable

import (
	"context"
	"io/fs"
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
	SocketFolderEnvVar  = "POLARIS_PLUGINS_SOCKETS_FOLDER"
	defaultSocketFolder = "/tmp/polaris-plugin-sockets"
)

// onServiceDiscoveredCallback registered a callback for when a plugin service is discovered.
type onServiceDiscoveredCallback func(name string, dialer GRPCConnectionDialer)

// onServiceDiscovered is a map of service name to callback function.
var onServiceDiscovered map[string]onServiceDiscoveredCallback

func init() {
	onServiceDiscovered = make(map[string]onServiceDiscoveredCallback)
}

// AddServiceDiscoveryCallback adds a callback function that should be called when the given service was discovered.
func AddServiceDiscoveryCallback(serviceName string, callbackFunc onServiceDiscoveredCallback) {
	onServiceDiscovered[serviceName] = callbackFunc
}

// GetSocketFolderPath returns the shared unix domain socket folder path
func GetSocketFolderPath() string {
	if value, ok := os.LookupEnv(SocketFolderEnvVar); ok {
		return value
	}
	return defaultSocketFolder
}

// service represents a service found by the discovery process.
type service struct {
	// protoRef is the proto service name
	protoRef string
	// componentName is the component name that implements such service.
	componentName string
	// dialer is the used grpc connection dialer.
	dialer GRPCConnectionDialer
}

// reflectServiceClient is the interface used to list services.
type reflectServiceClient interface {
	ListServices() ([]string, error)
	Reset()
}

// grpcConnectionCloser is used for closing the grpc connection.
type grpcConnectionCloser interface {
	grpc.ClientConnInterface
	Close() error
}

// reflectClientFactory is used for creating a reflection service client with a given socket.
type reflectClientFactory func(sock string) (reflectServiceClient, func(), error)

// serviceDiscovery search all sockets in the given path and returns all services found.
func serviceDiscovery(factory reflectClientFactory) ([]service, error) {
	componentsSocketPath := GetSocketFolderPath()
	_, err := os.Stat(componentsSocketPath)
	if err != nil {
		return nil, err
	}

	if os.IsNotExist(err) {
		return nil, nil
	}

	log.Infof("loading pluggable components under path %s", componentsSocketPath)

	var files []os.DirEntry
	files, err = os.ReadDir(componentsSocketPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not list pluggable components unix sockets")
	}

	var services []service
	for _, dirEntry := range files {
		var discoveredServices []service
		discoveredServices, err = singleSock(dirEntry, componentsSocketPath, factory)
		if err != nil {
			return nil, err
		}
		services = append(services, discoveredServices...)
	}

	log.Infof("found %d pluggable component services", len(services)-1)
	return services, nil
}

func singleSock(dirEntry os.DirEntry, componentsSocketPath string, factory reflectClientFactory) ([]service, error) {
	var services []service
	if dirEntry.IsDir() {
		return services, nil
	}

	f, err := dirEntry.Info()
	if err != nil {
		return nil, err
	}

	socket := filepath.Join(componentsSocketPath, f.Name())
	// current file is not a socket
	if !(f.Mode()&fs.ModeSocket != 0) {
		log.Warnf("could not use socket for file %s", socket)
		return services, nil
	}

	reflectClient, cleanup, err := factory(socket)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	serviceList, err := reflectClient.ListServices()
	if err != nil {
		return nil, errors.Wrap(err, "unable to list services")
	}

	componentName := f.Name()
	componentName = strings.TrimSuffix(componentName, filepath.Ext(componentName))

	dialer := socketDialer(socket, grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	for _, svc := range serviceList {
		services = append(services, service{
			componentName: componentName,
			protoRef:      svc,
			dialer:        dialer,
		})
	}

	return services, nil
}

// reflectServiceConnectionCloser is used for cleanup the stream created to be used for the reflection service.
func reflectServiceConnectionCloser(conn grpcConnectionCloser, client reflectServiceClient) func() {
	return func() {
		client.Reset()
		err := conn.Close()

		if err != nil {
			log.Errorf("error closing grpc reflection connection: %v", err)
		}
	}
}

// Discover discovers all available pluggable components services.
func Discover(ctx context.Context) error {
	factory := func(socket string) (reflectServiceClient, func(), error) {
		conn, err := SocketDial(
			ctx,
			socket,
			grpc.WithBlock(),
		)
		if err != nil {
			log.Errorf("pluggable error connecting to socket %s: %v", socket, err)
			return nil, nil, err
		}

		client := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
		return client, reflectServiceConnectionCloser(conn, client), nil
	}

	services, err := serviceDiscovery(factory)
	if err != nil {
		return err
	}

	discoveryFinishedCallback(services)
	return nil
}

// discoveryFinishedCallback is used for calling the registered callbacks when all services are discovered.
func discoveryFinishedCallback(services []service) {
	for _, svc := range services {
		cb, ok := onServiceDiscovered[svc.protoRef]
		if !ok {
			continue
		}
		cb(svc.componentName, svc.dialer)

		log.Infof("discovered pluggable component service: ", svc.protoRef)
	}
}
