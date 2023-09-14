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

import (
	"errors"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"

	"github.com/polarismesh/polaris/common/model"
)

func (s *Server) checkNamespaceExisted(namespaceName string) bool {
	if val := s.caches.Namespace().GetNamespace(namespaceName); val != nil {
		return true
	}
	namespace, _ := s.storage.GetNamespace(namespaceName)
	return namespace != nil
}

func convertToErrCode(err error) apimodel.Code {
	if errors.Is(err, model.ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}
	if errors.Is(err, model.ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}
	return apimodel.Code_NotAllowedAccess
}
