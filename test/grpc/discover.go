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
	"context"
	"fmt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/naming"
	"google.golang.org/grpc/metadata"
	"io"
	"time"
)

/**
 * @brief 统一发现函数
 */
func (c *Client) Discover(drt api.DiscoverRequest_DiscoverRequestType, service *api.Service) error {
	fmt.Printf("\ndiscover\n")

	md := metadata.Pairs("request-id", naming.NewUUID())
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	worker, err := c.Worker.Discover(ctx)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	request := &api.DiscoverRequest{
		Type:    drt,
		Service: service,
	}
	if err := worker.Send(request); err != nil {
		fmt.Printf("%v\n", err)
		return err
	}
	worker.CloseSend()

	for {
		rsp, err := worker.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			fmt.Printf("%v\n", err)
			return err
		}

		fmt.Printf("%v\n", rsp)
	}
}

/**
 * @brief 统一发现函数
 */
func (c *Client) DiscoverRequest(request *api.DiscoverRequest) (*api.DiscoverResponse, error) {
	fmt.Printf("\ndiscover\n")

	md := metadata.Pairs("request-id", naming.NewUUID())
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	worker, err := c.Worker.Discover(ctx)
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}

	if err := worker.Send(request); err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	worker.CloseSend()

	var latestResp *api.DiscoverResponse

	for {
		rsp, err := worker.Recv()
		if err != nil {
			if err == io.EOF {
				return latestResp, nil
			}

			fmt.Printf("%v\n", err)
		}

		fmt.Printf("%v\n", rsp)
		latestResp = rsp
	}

}