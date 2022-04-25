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

package grpc

import (
	"fmt"

	"google.golang.org/grpc"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// NewClient 创建GRPC客户端
func NewClient(address string) (*Client, error) {
	fmt.Printf("\nnew grpc client\n")

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	client := &Client{
		Conn:   conn,
		Worker: api.NewPolarisGRPCClient(conn),
	}

	return client, nil
}

// Client GRPC客户端
type Client struct {
	Conn   *grpc.ClientConn
	Worker api.PolarisGRPCClient
}

// Close 关闭连接
func (c *Client) Close() {
	c.Conn.Close()
}
