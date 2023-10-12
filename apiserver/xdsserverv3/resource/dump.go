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

package resource

import (
	"encoding/json"
	"sort"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	res "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v2"
)

func DumpSnapShotJSON(snapshot cache.ResourceSnapshot) []byte {
	data, err := json.Marshal(map[string]interface{}{
		"endpoints": ToJSONArray(snapshot.GetResources(res.EndpointType)),
		"clusters":  ToJSONArray(snapshot.GetResources(res.ClusterType)),
		"routers":   ToJSONArray(snapshot.GetResources(res.RouteType)),
		"listeners": ToJSONArray(snapshot.GetResources(res.ListenerType)),
	})
	if err != nil {
		return nil
	}
	return data
}

func YamlEncode(any interface{}) []byte {
	data, err := json.Marshal(any)
	if err != nil {
		log.Errorf("yaml encode json marshal failed error %v", err)
		return nil
	}
	o := make(map[string]interface{})
	if err = json.Unmarshal(data, &o); err != nil {
		log.Errorf("yaml encode json unmarshal failed error %v", err)
		return nil
	}
	if data, err = yaml.Marshal(o); err != nil {
		log.Errorf("yaml encode yaml marshal failed error %v", err)
		return nil
	}
	return data
}

func ToJSONArray(resources map[string]types.Resource) []json.RawMessage {
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

func ToYamlArray(resources map[string]types.Resource) []json.RawMessage {
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
