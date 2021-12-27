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

package naming

import (
	"context"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

func (svr *serverAuthAbility) CreateInstances(ctx context.Context, reqs []*api.Instance) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) CreateInstance(ctx context.Context, req *api.Instance) *api.Response {

}

func (svr *serverAuthAbility) DeleteInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) DeleteInstance(ctx context.Context, req *api.Instance) *api.Response {

}

func (svr *serverAuthAbility) DeleteInstancesByHost(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) DeleteInstanceByHost(ctx context.Context, req *api.Instance) *api.Response {

}

func (svr *serverAuthAbility) UpdateInstances(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UpdateInstance(ctx context.Context, req *api.Instance) *api.Response {

}

func (svr *serverAuthAbility) UpdateInstancesIsolate(ctx context.Context, req []*api.Instance) *api.BatchWriteResponse {

}

func (svr *serverAuthAbility) UpdateInstanceIsolate(ctx context.Context, req *api.Instance) *api.Response {

}

func (svr *serverAuthAbility) GetInstances(query map[string]string) *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) GetInstancesCount() *api.BatchQueryResponse {

}

func (svr *serverAuthAbility) CleanInstance(ctx context.Context, req *api.Instance) *api.Response {

}
