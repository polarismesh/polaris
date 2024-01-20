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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	nacoshttp "github.com/polarismesh/polaris/apiserver/nacosserver/v1/http"
)

func BuildInstance(namespace string, req *restful.Request, onlybase bool) (*model.Instance, error) {
	service, err := nacoshttp.Required(req, model.ParamServiceName)
	if err != nil {
		return nil, err
	}
	host, err := nacoshttp.Required(req, model.ParamInstanceIP)
	if err != nil {
		return nil, err
	}
	portStr, err := nacoshttp.Required(req, model.ParamInstancePort)
	port, _ := strconv.ParseInt(portStr, 10, 32)
	cluster := nacoshttp.Optional(req, model.ParamClusterName, "")
	if len(cluster) == 0 {
		cluster = nacoshttp.Optional(req, model.ParamCluster, model.DefaultServiceClusterName)
	}

	nacosIns := &model.Instance{
		ClusterName: cluster,
		ServiceName: service,
		Id:          fmt.Sprintf("%s#%s#%s#%s", namespace, service, host, portStr),
		IP:          host,
		Port:        int32(port),
		Ephemeral:   true,
	}

	if !onlybase {
		weightStr := nacoshttp.Optional(req, model.ParamInstanceWeight, "1")
		weight, _ := strconv.ParseFloat(weightStr, 64)
		if weight > model.InstanceMaxWeight || weight < model.InstanceMinWeight {
			return nil, &model.NacosError{
				ErrCode: int32(model.ExceptionCode_InvalidParam),
				ErrMsg: fmt.Sprintf("instance format invalid: The weights range from %f to %f",
					model.InstanceMinWeight, model.InstanceMaxWeight),
			}
		}
		healthyStr := nacoshttp.Optional(req, model.ParamInstanceHealthy, "true")
		healthy, _ := strconv.ParseBool(healthyStr)
		enableStr := nacoshttp.Optional(req, model.ParamInstanceEnabled, "")
		if len(enableStr) == 0 {
			enableStr = nacoshttp.Optional(req, model.ParamInstanceEnable, "true")
		}
		enable, _ := strconv.ParseBool(enableStr)
		metadataStr := nacoshttp.Optional(req, model.ParamInstanceMetadata, "")
		metadata, err := parseaMetadata(metadataStr)
		if err != nil {
			return nil, err
		}
		nacosIns.Metadata = metadata
		nacosIns.Weight = weight
		nacosIns.Healthy = healthy
		nacosIns.Enabled = enable
	}

	return nacosIns, nil
}

func BuildClientBeat(req *restful.Request) (*model.ClientBeat, error) {
	beatInfo := &model.ClientBeat{}
	beatStr := nacoshttp.Optional(req, model.ParamInstanceBeat, "")
	if len(beatStr) != 0 && json.Valid([]byte(beatStr)) {
		_ = json.Unmarshal([]byte(beatStr), beatInfo)
	}
	host := nacoshttp.Optional(req, model.ParamInstanceIP, "")
	portStr := nacoshttp.Optional(req, model.ParamInstancePort, "0")
	port, _ := strconv.ParseInt(portStr, 10, 32)
	cluster := nacoshttp.Optional(req, model.ParamClusterName, model.DefaultServiceClusterName)
	if len(beatInfo.Ip) != 0 && beatInfo.Port != 0 {
		if len(beatInfo.Cluster) == 0 {
			beatInfo.Cluster = cluster
		}
	} else {
		beatInfo.Ip = host
		beatInfo.Port = int(port)
	}

	namespace := nacoshttp.Optional(req, model.ParamNamespaceID, model.DefaultNacosNamespace)
	namespace = model.ToPolarisNamespace(namespace)
	service, err := nacoshttp.Required(req, model.ParamServiceName)
	if err != nil {
		return nil, err
	}

	beatInfo.Namespace = namespace
	beatInfo.ServiceName = service

	return beatInfo, nil
}

func parseaMetadata(metadataStr string) (map[string]string, error) {
	metadata := map[string]string{}

	if json.Valid([]byte(metadataStr)) {
		_ = json.Unmarshal([]byte(metadataStr), &metadata)
	} else {
		datas := strings.Split(metadataStr, ",")
		for i := range datas {
			kv := strings.Split(datas[i], ":")
			if len(kv) != 2 {
				return nil, &model.NacosApiError{
					Err: &model.NacosError{
						ErrCode: http.StatusBadRequest,
						ErrMsg:  fmt.Sprintf("metadata format incorrect:%s", metadataStr),
					},
					DetailErrCode: model.ErrorCode_InstanceMetadataError.Code,
					ErrAbstract:   model.ErrorCode_InstanceMetadataError.Desc,
				}
			}
			metadata[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return metadata, nil
}
