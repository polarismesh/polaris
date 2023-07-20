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

package config

import (
	"fmt"

	"github.com/polarismesh/polaris/common/log"
)

func defaultBootstrap() Bootstrap {
	return Bootstrap{
		Logger: defaultLoggerOptions(),
		StartInOrder: map[string]interface{}{
			"open": true,
			"key":  "sz",
		},
		PolarisService: PolarisService{
			EnableRegister: true,
			Isolated:       false,
			Services: []*Service{
				{
					Name:      "polaris.checker",
					Namespace: "Polaris",
					Protocols: []string{
						"service-grpc",
					},
				},
			},
		},
	}
}

// defaultLoggerOptions creates default logger options
func defaultLoggerOptions() map[string]*log.Options {
	return map[string]*log.Options{
		"config":        newLogOptions("runtime", "config"),
		"auth":          newLogOptions("runtime", "auth"),
		"store":         newLogOptions("runtime", "store"),
		"cache":         newLogOptions("runtime", "cache"),
		"naming":        newLogOptions("runtime", "naming"),
		"healthcheck":   newLogOptions("runtime", "healthcheck"),
		"xdsv3":         newLogOptions("runtime", "xdsv3"),
		"eureka":        newLogOptions("runtime", "eureka"),
		"apiserver":     newLogOptions("runtime", "apiserver"),
		"default":       newLogOptions("runtime", "default"),
		"token-bucket":  newLogOptions("runtime", "ratelimit"),
		"discoverLocal": newLogOptions("statis", "discoverstat"),
		"local":         newLogOptions("statis", "statis"),
		"HistoryLogger": newLogOptions("operation", "history", func(opt *log.Options) {
			opt.OnlyContent = true
			opt.RotationMaxDurationForHour = 24
		}),
		"discoverEventLocal": newLogOptions("event", "discoverevent", func(opt *log.Options) {
			opt.OnlyContent = true
		}),
		"cmdb": newLogOptions("runtime", "cmdb"),
	}
}

func newLogOptions(dir, name string, options ...func(opt *log.Options)) *log.Options {
	opt := &log.Options{
		RotateOutputPath:      fmt.Sprintf("log/%s/polaris-%s.log", dir, name),
		ErrorRotateOutputPath: fmt.Sprintf("log/%s/polaris-%s-error.log", dir, name),
		RotationMaxSize:       100,
		RotationMaxAge:        7,
		RotationMaxBackups:    30,
		Compress:              true,
		OutputLevel:           "info",
	}
	for i := range options {
		options[i](opt)
	}
	return opt
}
