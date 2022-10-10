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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/polarismesh/polaris/bootstrap"
)

var (
	configFilePath = ""

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "start running",
		Long:  "start running",
		Run: func(_ *cobra.Command, _ []string) {
			config, err := bootstrap.Start(configFilePath)
			if err != nil {
				fmt.Printf("[ERROR] %v\n", err)
			}

			fmt.Println(config)
			fmt.Println("finish starting server")
		},
	}
)

// init 解析命令参数
func init() {
	startCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "polaris-server.yaml", "config file path")
}
