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

package model

import (
	"net/http"

	api "github.com/polarismesh/polaris/common/api/v1"
)

type DebugHandlerGroup struct {
	Name     string
	Handlers []DebugHandler
}

type DebugHandler struct {
	Desc    string
	Path    string
	Handler http.HandlerFunc
}

type CommonResponse struct {
	Code uint32      `json:"code"`
	Info string      `json:"info"`
	Data interface{} `json:"data"`
}

func NewCommonResponse(code uint32) *CommonResponse {
	return &CommonResponse{
		Code: code,
		Info: api.Code2Info(code),
	}
}
