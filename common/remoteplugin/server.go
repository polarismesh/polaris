package remoteplugin

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"

	rp "github.com/polarismesh/polaris/common/api/plugin"
	"github.com/polarismesh/polaris/common/log"
)

// server plugin server.
type server struct {
	Backend Service
}

// Call calls plugin backend server.
func (s *server) Call(ctx context.Context, req *rp.Request) (*rp.Response, error) {
	return s.Backend.Call(ctx, req)
}

// Serve is a function used to serve a plugin
func Serve(ctx context.Context, svc Service) {
	go serve(svc)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Info("polaris server exited, stop plugin server")
			os.Exit(0)
		}
	}
}

func serve(svc Service) {
	p := os.Getenv("PLUGIN_PROCS")
	if procs, err := strconv.Atoi(p); err == nil {
		runtime.GOMAXPROCS(procs)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"POLARIS_SERVER": &Plugin{Backend: svc},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
