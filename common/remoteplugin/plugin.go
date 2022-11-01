package remoteplugin

import (
	"context"
	"sync"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pluginapi "github.com/polarismesh/polaris/common/api/plugin"
)

var pluginSet = make(map[string]*Client)
var locker sync.Mutex

// Register called by plugin client and start up the plugin main process.
func Register(name string, config *Config) (*Client, error) {
	locker.Lock()
	defer locker.Unlock()

	if c, ok := pluginSet[name]; ok {
		return c, nil
	}

	c, err := newClient(name, config)
	if err != nil {
		return nil, err
	}
	pluginSet[name] = c

	return c, nil
}

// Handshake go-plugin client handshake config.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "POLARIS_PLUGIN",
	MagicCookieValue: "ON",
}

// PluginMap plugin map.
var PluginMap = map[string]plugin.Plugin{
	"POLARIS_SERVER": &Plugin{},
}

// Service is a service that Implemented by plugin main process
type Service interface {
	Call(ctx context.Context, request *pluginapi.Request) (*pluginapi.Response, error)
}

// Plugin is the implementation of plugin.GRPCPlugin, so we can serve/consume this.
type Plugin struct {
	plugin.Plugin
	Backend Service
}

// GRPCServer implements plugin.Plugin GRPCServer method.
func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginapi.RegisterPluginServer(s, &server{Backend: p.Backend})
	return nil
}

// GRPCClient implements plugin.Plugin GRPCClient method.
func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &client{PluginClient: pluginapi.NewPluginClient(c)}, nil
}
