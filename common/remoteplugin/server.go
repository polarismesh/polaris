package remoteplugin

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"

	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/remoteplugin/proto"
)

// server plugin server.
type server struct {
	Backend Service
}

// Call calls plugin backend server.
func (s *server) Call(ctx context.Context, req *proto.Request) (*proto.Response, error) {
	return s.Backend.Call(req)
}

// gracefulExit check context and graceful exit.
func gracefulExit(ctx context.Context) {
	go func() {
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
	}()
}

// Serve is a function used to serve a plugin
func Serve(ctx context.Context, service Service) {
	gracefulExit(ctx)

	p := os.Getenv("PLUGIN_PROCS")
	if procs, err := strconv.Atoi(p); err == nil {
		runtime.GOMAXPROCS(procs)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"POLARIS_SERVER": &Plugin{Backend: service},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
