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

package xdsserverv3

import (
	"encoding/json"
	"sort"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	res "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v2"
)

func dumpSnapShot(snapshot cache.ResourceSnapshot) []byte {
	return yamlEncode(map[string]interface{}{
		"endpoints": toJSONArray(snapshot.GetResources(res.EndpointType)),
		"clusters":  toJSONArray(snapshot.GetResources(res.ClusterType)),
		"routers":   toJSONArray(snapshot.GetResources(res.RouteType)),
		"listeners": toJSONArray(snapshot.GetResources(res.ListenerType)),
	})
}

func dumpSnapShotJSON(snapshot cache.ResourceSnapshot) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"endpoints": toJSONArray(snapshot.GetResources(res.EndpointType)),
		"clusters":  toJSONArray(snapshot.GetResources(res.ClusterType)),
		"routers":   toJSONArray(snapshot.GetResources(res.RouteType)),
		"listeners": toJSONArray(snapshot.GetResources(res.ListenerType)),
	})
	return data
}

func yamlEncode(any interface{}) []byte {
	data, _ := json.Marshal(any)
	o := make(map[string]interface{})
	_ = json.Unmarshal(data, &o)
	data, _ = yaml.Marshal(o)
	return data
}

func toJSONArray(resources map[string]types.Resource) []json.RawMessage {
	list := make([]resouceWithName, 0, len(resources))
	for name, x := range resources {
		list = append(list, resouceWithName{resource: x, name: name})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].name < list[j].name
	})

	messages := make([]json.RawMessage, 0, len(resources))
	for _, x := range list {
		data, _ := protojson.Marshal(x.resource)
		messages = append(messages, data)
	}
	return messages
}

type resouceWithName struct {
	resource types.Resource
	name     string
}
