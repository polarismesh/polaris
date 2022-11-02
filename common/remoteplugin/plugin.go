package remoteplugin

import (
	"context"
	"sync"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pluginapi "github.com/polarismesh/polaris/common/api/plugin"
	"github.com/polarismesh/polaris/common/log"
)

var pluginSet = make(map[string]*Client)
var locker sync.Mutex

// Register called by plugin client and start up the plugin main process.
func Register(config *Config) (*Client, error) {
	locker.Lock()
	defer locker.Unlock()

	name := config.Name
	if c, ok := pluginSet[name]; ok {
		return c, nil
	}

	c, err := newClient(config)
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

// Plugin must implement plugin.GRPCPlugin
var _ plugin.GRPCPlugin = (*Plugin)(nil)

// Plugin is the implementation of plugin.GRPCPlugin, so we can serve/consume this.
type Plugin struct {
	plugin.Plugin
	Name     string
	IsRemote bool
	Backend  Service
}

// NewPlugin returns a new Plugin.
func NewPlugin(name string, backend Service) *Plugin {
	return &Plugin{Name: name, Backend: backend}
}

// GRPCServer implements plugin.Plugin GRPCServer method.
func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	if p.IsRemote {
		log.Infof("plugin %s running in remote mode, skip grpc serverImp setup", p.Name)
		return nil
	}
	pluginapi.RegisterPluginServer(s, &serverImp{Backend: p.Backend})
	return nil
}

// GRPCClient implements plugin.Plugin GRPCClient method.
func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &client{PluginClient: pluginapi.NewPluginClient(c)}, nil
}
