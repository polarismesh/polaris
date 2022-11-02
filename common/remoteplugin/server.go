package remoteplugin

import (
	"context"

	"github.com/polarismesh/polaris/common/api/plugin"
)

// Service must implement pluginapi.PluginServer
var _ plugin.PluginServer = (Service)(nil)

// Service implements remote proto defined Plugin Server
type Service interface {
	// Call implement pluginapi.Server.Call
	Call(ctx context.Context, request *plugin.Request) (*plugin.Response, error)
}

// serverImp must implements Service.
var _ Service = (*serverImp)(nil)

// serverImp plugin serverImp.
type serverImp struct {
	Backend Service
}

// Call calls plugin backend serverImp.
func (s *serverImp) Call(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	return s.Backend.Call(ctx, req)
}

// ServerConfig plugin configs
type ServerConfig struct {
	PluginName string
}
