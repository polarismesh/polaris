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

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	namespaceName = "test-ns-%v-%v"
)

/**
 * @brief 创建测试命名空间
 */
func CreateNamespaces() []*apimodel.Namespace {
	var namespaces []*apimodel.Namespace
	for index := 0; index < 2; index++ {
		name := fmt.Sprintf(namespaceName, utils.NewUUID(), index)

		namespace := &apimodel.Namespace{
			Name:    utils.NewStringValue(name),
			Comment: utils.NewStringValue("test"),
			Owners:  utils.NewStringValue("test"),
		}
		namespaces = append(namespaces, namespace)
	}

	return namespaces
}

/**
 * @brief 更新测试命名空间
 */
func UpdateNamespaces(namespaces []*apimodel.Namespace) {
	for _, namespace := range namespaces {
		namespace.Comment = utils.NewStringValue("update")
		namespace.Owners = utils.NewStringValue("update")
	}
}
