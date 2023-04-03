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
	"fmt"

	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	totalGroups        = 2
	confiGroupNameTemp = "fileGroup-%s-%d"
	configFileNameTemp = "file-%s-%d"
)

func MockConfigGroups(ns *apimodel.Namespace) []*apiconfig.ConfigFileGroup {
	ret := make([]*apiconfig.ConfigFileGroup, 0, totalGroups)

	for i := 0; i < totalGroups; i++ {

		ret = append(ret, &apiconfig.ConfigFileGroup{
			Name: &wrapperspb.StringValue{
				Value: fmt.Sprintf(confiGroupNameTemp, utils.NewUUID(), i),
			},
			Namespace: &wrapperspb.StringValue{
				Value: ns.Name.Value,
			},
			Comment: &wrapperspb.StringValue{
				Value: "",
			},
		})

	}

	return ret
}

func MockConfigFiles(group *apiconfig.ConfigFileGroup) []*apiconfig.ConfigFile {
	ret := make([]*apiconfig.ConfigFile, 0, totalGroups)

	for i := 0; i < totalGroups; i++ {
		name := fmt.Sprintf(configFileNameTemp, utils.NewUUID(), i)
		if i%2 == 0 {
			name = fmt.Sprintf("dir%d/", i) + name
		}
		ret = append(ret, &apiconfig.ConfigFile{
			Name: &wrapperspb.StringValue{
				Value: name,
			},
			Namespace: group.Namespace,
			Group:     group.Name,
			Content: &wrapperspb.StringValue{
				Value: `name: polarismesh`,
			},
			Format: &wrapperspb.StringValue{
				Value: "yaml",
			},
			Status: &wrapperspb.StringValue{
				Value: utils.ReleaseStatusToRelease,
			},
			Tags: []*apiconfig.ConfigFileTag{},
		})

	}

	return ret
}
