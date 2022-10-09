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

	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
)

const (
	totalGroups        = 2
	confiGroupNameTemp = "fileGroup-%s-%d"
	configFileNameTemp = "file-%s-%d"
)

func MockConfigGroups(ns *api.Namespace) []*api.ConfigFileGroup {
	ret := make([]*api.ConfigFileGroup, 0, totalGroups)

	for i := 0; i < totalGroups; i++ {

		ret = append(ret, &api.ConfigFileGroup{
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

func MockConfigFiles(group *api.ConfigFileGroup) []*api.ConfigFile {
	ret := make([]*api.ConfigFile, 0, totalGroups)

	for i := 0; i < totalGroups; i++ {

		ret = append(ret, &api.ConfigFile{
			Name: &wrapperspb.StringValue{
				Value: fmt.Sprintf(confiGroupNameTemp, utils.NewUUID(), i),
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
			Tags: []*api.ConfigFileTag{},
		})

	}

	return ret
}
