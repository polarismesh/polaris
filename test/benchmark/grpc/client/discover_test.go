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

package benchmark_grpc

import (
	"context"
	"fmt"
	"os"
	"testing"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/test/integrate/http"
	"github.com/polarismesh/polaris/test/integrate/resource"
)

func init() {
	// prepareCreateService()
	prepareCreateRouterRule()
}

func prepareCreateService() {
	target := "127.0.0.1:8090"
	if val := os.Getenv("BENCHMARK_SERVER_HTTP_ADDRESS"); len(val) > 0 {
		target = val
	}

	httpClient := http.NewClient(target, "v1")

	svcs := resource.CreateServicesWithTotal(&apimodel.Namespace{
		Name: utils.NewStringValue("mock_ns"),
	}, 100)

	if _, err := httpClient.CreateServices(svcs); err != nil {
		panic(err)
	}
}

func prepareCreateRouterRule() {
	target := "127.0.0.1:8090"
	if val := os.Getenv("BENCHMARK_SERVER_HTTP_ADDRESS"); len(val) > 0 {
		target = val
	}

	_ = http.NewClient(target, "v1")
}

func prepareDiscoverClient(b *testing.B) (apiservice.PolarisGRPC_DiscoverClient, *grpc.ClientConn) {
	target := "127.0.0.1:8091"
	if val := os.Getenv("BENCHMARK_SERVER_ADDRESS"); len(val) > 0 {
		target = val
	}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	b.Log("connection server success")
	client := apiservice.NewPolarisGRPCClient(conn)
	discoverClient, err := client.Discover(ctx)
	if err != nil {
		panic(err)
	}
	b.Log("create discover client success")
	return discoverClient, conn
}

func Benchmark_DiscoverServicesWithoutRevision(b *testing.B) {
	discoverClient, conn := prepareDiscoverClient(b)
	defer conn.Close()

	fmt.Println("begin do benchmark")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		err := discoverClient.Send(&apiservice.DiscoverRequest{
			Type:    apiservice.DiscoverRequest_SERVICES,
			Service: &apiservice.Service{},
		})
		if err != nil {
			b.Fatal(err)
		}
		resp, err := discoverClient.Recv()
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		if resp.GetCode().GetValue() > 300000 {
			b.Fatal(resp)
		}
	}
}
