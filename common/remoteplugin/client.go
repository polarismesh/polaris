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

package remoteplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-plugin"

	"github.com/polarismesh/polaris/common/remoteplugin/proto"
)

// Config remote plugin config
type Config struct {
	// MaxProcs the max proc number, current plugin can use.
	MaxProcs int
	// Args plugin args
	Args []string
}

// client wraps a remote plugin client.
type client struct {
	proto.PluginClient
}

// Client is a plugin client, It's primarily used to call request.
type Client struct {
	sync.Mutex
	pluginName   string         // the name of the plugin, used for manage.
	pluginPath   string         // the full path of the plugin, go-plugin cmd start plugin according plugin path.
	on           bool           // represents the plugin is opened or not.
	enable       bool           // represents the plugin is enabled or not
	service      *client        // service is the plugin-grpc-service client, polaris-server run in grpc-client side.
	config       *Config        // the setup config of the plugin
	pluginClient *plugin.Client // the go-plugin client, polaris-server run in grpc-client side.
}

// Call invokes the function synchronously.
func (c *Client) Call(ctx context.Context, request *proto.Request) (*proto.Response, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	return c.service.Call(ctx, request)
}

// disable set plugin is disabled.
func (c *Client) disable() error {
	c.Lock()
	c.enable = false
	c.on = false
	c.Unlock()
	if c.pluginClient != nil {
		c.pluginClient.Kill()
	}
	return nil
}

// open set plugin is enabled.
func (c *Client) open() error {
	c.Lock()
	c.enable = true
	c.Unlock()

	return c.Check()
}

func (c *Client) Check() error {
	c.Lock()
	defer c.Unlock()

	if !c.enable {
		return errors.New("plugin " + c.pluginName + " disable")
	}

	if c.pluginClient != nil && !c.pluginClient.Exited() {
		return nil
	}
	c.on = false

	cmd := exec.Command(c.pluginPath, c.config.Args...)

	procs := 1
	if c.config != nil && c.config.MaxProcs >= 0 && c.config.MaxProcs <= 4 {
		procs = c.config.MaxProcs
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PLUGIN_PROCS=%d", procs))
	pluginClient := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		Plugins:          PluginMap,
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	rpcClient, err := pluginClient.Client()
	if err != nil {
		return err
	}

	raw, err := rpcClient.Dispense("POLARIS_SERVER")
	if err != nil {
		return err
	}

	c.pluginClient = pluginClient
	c.service = raw.(*client)
	c.on = true

	return nil
}

// newClient new client
func newClient(name string, config *Config) (*Client, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	fullName := dir + string(os.PathSeparator) + name
	if _, err = os.Stat(fullName); err != nil {
		return nil, fmt.Errorf("check plugin fiel stat error: %w", err)
	}

	c := new(Client)
	c.enable = true
	c.pluginName = name
	c.pluginPath = fullName
	c.config = config
	if c.config == nil {
		c.config = &Config{1, nil}
	}

	if err = c.Check(); err != nil {
		return nil, fmt.Errorf("fail to check client: %w", err)
	}
	return c, nil
}
