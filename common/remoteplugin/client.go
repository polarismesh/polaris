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
	"os/exec"
	"sync"

	"github.com/hashicorp/go-plugin"

	pluginapi "github.com/polarismesh/polaris/common/api/plugin"
)

// client wraps a remote plugin client.
type client struct {
	pluginapi.PluginClient
}

// Client is a plugin client, It's primarily used to call request.
type Client struct {
	sync.Mutex
	pluginName   string         // the name of the plugin, used for manage.
	pluginPath   string         // the full path of the plugin, go-plugin cmd start plugin according plugin path.
	on           bool           // represents the plugin is opened or not.
	enable       bool           // represents the plugin is enabled or not
	service      *client        // service is the plugin-grpc-service client, polaris-serverImp run in grpc-client side.
	config       *Config        // the setup config of the plugin
	pluginClient *plugin.Client // the go-plugin client, polaris-serverImp run in grpc-client side.
}

// Call invokes the function synchronously.
func (c *Client) Call(ctx context.Context, request *pluginapi.Request) (*pluginapi.Response, error) {
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

// newClient returns a new client
func newClient(config *Config) (*Client, error) {
	c := new(Client)
	c.enable = true
	c.pluginName = config.Name
	if config.Mode == PluginRumModelRemote {
	} else {
		config.repairConfig()
		fullPath, err := config.pluginLoadPath()
		if err != nil {
			return nil, err
		}
		c.pluginPath = fullPath
	}

	c.config = config
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("fail to check client: %w", err)
	}
	return c, nil
}

// Check checks client still alive, create if not alive
func (c *Client) Check() error {
	c.Lock()
	defer c.Unlock()

	if !c.enable {
		return errors.New("plugin " + c.pluginName + " disable")
	}

	// plugin still alive, return as early as possible.
	if c.pluginClient != nil && !c.pluginClient.Exited() {
		return nil
	}

	return c.recreate()
}

// recreate
func (c *Client) recreate() error {
	c.on = false

	var pluginClient *plugin.Client

	if c.config.Mode == PluginRumModelLocal {
		pluginClient = c.recreateLocal()
	} else if c.config.Mode == PluginRumModelRemote {

	} else {
		return fmt.Errorf("unkown plugin run mode: %s", c.config.Mode)
	}

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

func (c *Client) recreateLocal() *plugin.Client {
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

	return pluginClient
}
