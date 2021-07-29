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

package batch

import (
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/store"
)

// store code 2 api code
func StoreCode2APICode(err error) uint32 {
	code := store.Code(err)
	switch {
	case code == store.EmptyParamsErr:
		return api.InvalidParameter
	case code == store.OutOfRangeErr:
		return api.InvalidParameter
	case code == store.DataConflictErr:
		return api.DataConflict
	case code == store.NotFoundNamespace:
		return api.NotFoundNamespace
	case code == store.NotFoundService:
		return api.NotFoundService
	case code == store.NotFoundMasterConfig:
		return api.NotFoundMasterConfig
	case code == store.NotFoundTagConfigOrService:
		return api.NotFoundTagConfigOrService
	case code == store.ExistReleasedConfig:
		return api.ExistReleasedConfig
	case code == store.DuplicateEntryErr:
		return api.ExistedResource
	default:
		return api.StoreLayerException
	}
}
