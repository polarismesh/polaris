package xdsserverv3

import (
	"encoding/json"
	"sort"

	res "github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
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
	json.Unmarshal(data, &o)
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
