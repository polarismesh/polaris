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
	"io"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/grpc/metadata"

	"github.com/polarismesh/polaris/common/utils"
)

// Discover 统一发现函数
func (c *Client) Discover(drt apiservice.DiscoverRequest_DiscoverRequestType, service *apiservice.Service, hook func(resp *apiservice.DiscoverResponse)) error {
	fmt.Printf("\ndiscover\n")

	md := metadata.Pairs("request-id", utils.NewUUID())
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	worker, err := c.Worker.Discover(ctx)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	request := &apiservice.DiscoverRequest{
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

		hook(rsp)
	}
}

// DiscoverRequest 统一发现函数
func (c *Client) DiscoverRequest(request *apiservice.DiscoverRequest) (*apiservice.DiscoverResponse, error) {
	fmt.Printf("\ndiscover\n")

	md := metadata.Pairs("request-id", utils.NewUUID())
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

	var latestResp *apiservice.DiscoverResponse

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
