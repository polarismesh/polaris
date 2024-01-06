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

package config

var (
	availableSearch = map[string]map[string]string{
		"config_file": {
			"namespace":   "namespace",
			"group":       "group",
			"name":        "name",
			"offset":      "offset",
			"limit":       "limit",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
		"config_file_release": {
			"namespace":    "namespace",
			"group":        "group",
			"file_name":    "file_name",
			"fileName":     "file_name",
			"name":         "release_name",
			"release_name": "release_name",
			"offset":       "offset",
			"limit":        "limit",
			"order_type":   "order_type",
			"order_field":  "order_field",
			"only_active":  "only_active",
		},
		"config_file_group": {
			"namespace":   "namespace",
			"group":       "name",
			"name":        "name",
			"business":    "business",
			"department":  "department",
			"offset":      "offset",
			"limit":       "limit",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
		"config_file_release_history": {
			"namespace":   "namespace",
			"group":       "group",
			"name":        "file_name",
			"offset":      "offset",
			"limit":       "limit",
			"endId":       "endId",
			"end_id":      "endId",
			"order_type":  "order_type",
			"order_field": "order_field",
		},
	}
)

func (s *Server) checkNamespaceExisted(namespaceName string) bool {
	if val := s.caches.Namespace().GetNamespace(namespaceName); val != nil {
		return true
	}
	namespace, _ := s.storage.GetNamespace(namespaceName)
	return namespace != nil
}
