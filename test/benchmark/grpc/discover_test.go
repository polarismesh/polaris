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
	"os"
	"testing"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/polarismesh/polaris/common/utils"
)

// goos: linux
// goarch: amd64
// pkg: github.com/polarismesh/polaris/test/benchmark/grpc
// cpu: AMD EPYC 7K62 48-Core Processor
// Benchmark_DiscoverServicesWithoutRevision-16                7000            161858 ns/op
// --- BENCH: Benchmark_DiscoverServicesWithoutRevision-16
//     discover_test.go:42: connection server success
//     discover_test.go:53: create discover client success
//     discover_test.go:60: send msg success
//     discover_test.go:65: receive msg success
//     discover_test.go:72: get service list total : 2
//     discover_test.go:42: connection server success
//     discover_test.go:53: create discover client success
//     discover_test.go:60: send msg success
//     discover_test.go:65: receive msg success
//     discover_test.go:72: get service list total : 2
//         ... [output truncated]
// Benchmark_DiscoverServicesWithRevision-16                   6477            172417 ns/op
// --- BENCH: Benchmark_DiscoverServicesWithRevision-16
//     discover_test.go:86: connection server success
//     discover_test.go:97: create discover client success
//     discover_test.go:108: send msg success
//     discover_test.go:113: receive msg success
//     discover_test.go:123: get service list total : 2
//     discover_test.go:86: connection server success
//     discover_test.go:97: create discover client success
//     discover_test.go:108: send msg success
//     discover_test.go:113: receive msg success
//     discover_test.go:123: get service list total : 2
//         ... [output truncated]
// PASS
// ok      github.com/polarismesh/polaris/test/benchmark/grpc      2.299s

func Benchmark_DiscoverServicesWithoutRevision(b *testing.B) {
	target := "127.0.0.1:8091"
	if val := os.Getenv("BENCHMARK_SERVER_ADDRESS"); len(val) > 0 {
		target = val
	}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err)
	}
	b.Log("connection server success")
	defer func() {
		if err := conn.Close(); err != nil {
			b.Fatal(err)
		}
	}()
	client := apiservice.NewPolarisGRPCClient(conn)
	discoverClient, err := client.Discover(ctx)
	if err != nil {
		b.Fatal(err)
	}
	b.Log("create discover client success")

	for i := 0; i < b.N; i++ {
		err := discoverClient.Send(&apiservice.DiscoverRequest{
			Type:    apiservice.DiscoverRequest_SERVICES,
			Service: &apiservice.Service{},
		})
		b.Log("send msg success")
		if err != nil {
			b.Fatal(err)
		}
		resp, err := discoverClient.Recv()
		b.Log("receive msg success")
		if err != nil {
			b.Fatal(err)
		}
		if resp.GetCode().GetValue() > 300000 {
			b.Fail()
		}
		b.Logf("get service list total : %d", len(resp.GetServices()))
	}
}

func Benchmark_DiscoverServicesWithRevision(b *testing.B) {
	target := "127.0.0.1:8091"
	if val := os.Getenv("BENCHMARK_SERVER_ADDRESS"); len(val) > 0 {
		target = val
	}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		b.Fatal(err)
	}
	b.Log("connection server success")
	defer func() {
		if err := conn.Close(); err != nil {
			b.Fatal(err)
		}
	}()
	client := apiservice.NewPolarisGRPCClient(conn)
	discoverClient, err := client.Discover(ctx)
	if err != nil {
		b.Fatal(err)
	}
	b.Log("create discover client success")

	revision := ""

	for i := 0; i < b.N; i++ {
		err := discoverClient.Send(&apiservice.DiscoverRequest{
			Type: apiservice.DiscoverRequest_SERVICES,
			Service: &apiservice.Service{
				Revision: utils.NewStringValue(revision),
			},
		})
		b.Log("send msg success")
		if err != nil {
			b.Fatal(err)
		}
		resp, err := discoverClient.Recv()
		b.Log("receive msg success")
		if err != nil {
			b.Fatal(err)
		}

		code := apimodel.Code(resp.GetCode().GetValue())
		if code != apimodel.Code_ExecuteSuccess && code != apimodel.Code_DataNoChange {
			b.Fail()
		}

		b.Logf("get service list total : %d", len(resp.GetServices()))
		revision = resp.GetService().GetRevision().GetValue()
	}
}
