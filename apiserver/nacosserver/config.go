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

package nacosserver

import (
	"github.com/mitchellh/mapstructure"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	"github.com/polarismesh/polaris/common/secure"
)

type NacosConfig struct {
	ListenPort       uint32            `mapstructure:"listenPort"`
	ConnLimit        *connlimit.Config `mapstructure:"connLimit"`
	TLS              *secure.TLSConfig `mapstructure:tls`
	DefaultNamespace string            `mapstructure:defaultNamespace`
	ServerService    string            `mapstructure:"serverService"`
	ServerNamespace  string            `mapstructure:"serverNamespace"`
}

func loadNacosConfig(raw map[string]interface{}) (*NacosConfig, error) {
	defaultCfg := &NacosConfig{
		ListenPort:       model.DefaultListenPort,
		DefaultNamespace: "default",
	}
	decodeConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     defaultCfg,
	}
	decoder, err := mapstructure.NewDecoder(decodeConfig)
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(raw); err != nil {
		return nil, err
	}
	return defaultCfg, nil
}
