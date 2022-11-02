package remoteplugin

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/hashicorp/go-plugin"

	"github.com/polarismesh/polaris/common/log"
)

// Serve is a function used to serve a plugin
func Serve(ctx context.Context, serveConfig ServerConfig, svc Service) {
	if serveConfig.PluginName == "" {
		log.Fatal("plugin name is required")
	}

	go serve(serveConfig.PluginName, svc)

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Info("polaris server exited, will stop plugin server")
			os.Exit(0)
		}
	}
}

func serve(pluginName string, svc Service) {
	defer func() {
		if e := recover(); e != nil {
			log.Fatalf("plugin panic: %+v", e)
		}
	}()

	p := os.Getenv("PLUGIN_PROCS")
	if procs, err := strconv.Atoi(p); err == nil {
		runtime.GOMAXPROCS(procs)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			pluginName: NewPlugin(pluginName, svc),
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
