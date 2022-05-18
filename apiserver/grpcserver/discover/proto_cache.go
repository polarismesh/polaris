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

package discover

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"google.golang.org/grpc"
)

const (
	enableProtobufCacheKey = "enableCacheProto"
	sizeProtobufCacheKey   = "sizeCacheProto"
)

// ProtobufCache pb对象缓存，降低由于pb重复对象序列化带来的开销
type ProtobufCache struct {
	enabled       bool
	cahceRegistry map[api.DiscoverResponse_DiscoverResponseType]*lru.ARCCache
}

// newProtobufCache 构件一个 pb 缓存池
func newProtobufCache(options map[string]interface{}) (*ProtobufCache, error) {
	enabled, _ := options[enableProtobufCacheKey].(bool)

	if !enabled {
		return &ProtobufCache{
			enabled: enabled,
		}, nil
	}

	size, _ := options[sizeProtobufCacheKey].(int)
	if size == 0 {
		size = 128
	}

	cahceRegistry := make(map[api.DiscoverResponse_DiscoverResponseType]*lru.ARCCache)
	respTypes := []api.DiscoverResponse_DiscoverResponseType{
		api.DiscoverResponse_INSTANCE,
		api.DiscoverResponse_SERVICES,
		api.DiscoverResponse_CIRCUIT_BREAKER,
		api.DiscoverResponse_ROUTING,
		api.DiscoverResponse_RATE_LIMIT,
	}

	for i := range respTypes {
		cache, err := lru.NewARC(size)
		if err != nil {
			return nil, fmt.Errorf("init protobuf=[%s] cache fail : %+v", respTypes[i].String(), err)
		}
		cahceRegistry[respTypes[i]] = cache
	}

	return &ProtobufCache{
		enabled:       enabled,
		cahceRegistry: cahceRegistry,
	}, nil
}

// OnRecv Treatment when receiving the request
func (pc *ProtobufCache) OnRecv(stream grpc.ServerStream, m interface{}) interface{} {
	return m
}

// OnSend Ready to send data processing
func (pc *ProtobufCache) OnSend(stream grpc.ServerStream, m interface{}) interface{} {
	if !pc.enabled {
		return m
	}
	resp, ok := m.(*api.DiscoverResponse)
	if !ok {
		return m
	}
	if resp.GetCode().GetValue() != api.ExecuteSuccess {
		return m
	}
	// 计算缓存数据的 key 信息
	keyProto := fmt.Sprintf("%s-%s-%s", resp.Service.Namespace.GetValue(), resp.Service.Name.GetValue(),
		resp.Service.Revision.GetValue())
	value, ok := pc.cahceRegistry[resp.Type].Get(keyProto)

	defer func() {
		plugin.GetStatis().AddCacheCall(plugin.ComponentProtobufCache, resp.Type.String(), ok, 1)
	}()

	if !ok {
		// 没有缓存
		pmsg := &grpc.PreparedMsg{}
		errEncode := pmsg.Encode(stream, m)
		if errEncode != nil {
			log.Infof("SendMsg encode err %v %v", keyProto, errEncode)
			return m
		} else {
			// 添加缓存
			pc.cahceRegistry[resp.Type].Add(keyProto, pmsg)
			log.Debugf("SendMsg add cache %v", keyProto)
			return pmsg
		}
	}
	return value
}
