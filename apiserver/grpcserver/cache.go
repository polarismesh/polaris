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

package grpcserver

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/polarismesh/polaris-server/plugin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	enableProtobufCacheKey = "enableCacheProto"
	sizeProtobufCacheKey   = "sizeCacheProto"
)

// MessageToCache
type MessageToCache func(m interface{}) *CacheObject

// Cache
type Cache interface {
	// Get
	Get(cacheType string, key string) *CacheObject
	// Put
	Put(v *CacheObject) *CacheObject
}

// CacheObject
type CacheObject struct {
	OriginVal proto.Message

	preparedVal *grpc.PreparedMsg

	CacheType string

	Key string
}

func (c *CacheObject) GetPreparedMessage() *grpc.PreparedMsg {
	return c.preparedVal
}

func (c *CacheObject) PrepareMessage(stream grpc.ServerStream) error {
	pmsg := &grpc.PreparedMsg{}
	if err := pmsg.Encode(stream, c.OriginVal); err != nil {
		log.Error("[Grpc][ProtoCache] prepare message", zap.String("key", c.Key), zap.Error(err))
		return err
	}
	c.preparedVal = pmsg
	return nil
}

// protobufCache PB object cache, reduce the overhead caused by the serialization of the PB repeated object
type protobufCache struct {
	enabled       bool
	cahceRegistry map[string]*lru.ARCCache
}

// NewCache Component a PB cache pool
func NewCache(options map[string]interface{}, cacheType []string) (Cache, error) {
	enabled, _ := options[enableProtobufCacheKey].(bool)

	if !enabled {
		return nil, nil
	}

	size, _ := options[sizeProtobufCacheKey].(int)
	if size == 0 {
		size = 128
	}

	cahceRegistry := make(map[string]*lru.ARCCache)

	for i := range cacheType {
		cache, err := lru.NewARC(size)
		if err != nil {
			return nil, fmt.Errorf("init protobuf=[%s] cache fail : %+v", cacheType[i], err)
		}
		cahceRegistry[cacheType[i]] = cache
	}

	return &protobufCache{
		enabled:       enabled,
		cahceRegistry: cahceRegistry,
	}, nil
}

// Get get value by cacheType and key
func (pc *protobufCache) Get(cacheType string, key string) *CacheObject {
	c, ok := pc.cahceRegistry[cacheType]
	if !ok {
		return nil
	}

	val, exist := c.Get(key)

	plugin.GetStatis().AddCacheCall(plugin.ComponentProtobufCache, cacheType, exist, 1)

	if val == nil {
		return nil
	}

	return val.(*CacheObject)
}

// Put save cache value
func (pc *protobufCache) Put(v *CacheObject) *CacheObject {
	if v == nil {
		return nil
	}

	cacheType := v.CacheType
	key := v.Key

	c, ok := pc.cahceRegistry[cacheType]
	if !ok {
		log.Warn("[Grpc][ProtoCache] put cache ignore", zap.String("cacheType", cacheType))
		return nil
	}

	c.Add(key, v)
	return v
}
