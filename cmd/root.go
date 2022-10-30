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
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:          "polaris-server",
		Short:        "polaris server",
		Long:         "polaris server",
		SilenceUsage: true,
	}
)

// init 初始化命令行工具
func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(revisionCmd)
}

// Execute 执行命令行解析
func Execute() {
	_ = rootCmd.Execute()
}
