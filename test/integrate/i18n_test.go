//go:build integration
// +build integration

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
package test

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/test/integrate/http"
)

// TestI18n 测试国际化信息
func TestI18n(t *testing.T) {
	t.Log("test i18n")
	type args struct {
		lang string
		want string
	}
	tests := []args{
		{lang: "zh", want: "命名空间名称非法"},
		{lang: "en", want: "invalid namespace name"},
	}
	for _, item := range tests {
		ret, err := reqCreateIllegalNamespace(item.lang)
		if err != nil {
			t.Fatalf("create namespace fail for i18n test: %s", err.Error())
		}
		if msg := ret.GetInfo().Value; msg != item.want {
			t.Errorf("test i18n by create namespace resp msg = %v, want: %v", msg, item.want)
		}
	}
}

func reqCreateIllegalNamespace(lang string) (*apiservice.BatchWriteResponse, error) {
	c := http.NewClient(httpserverAddress, httpserverVersion)
	url := fmt.Sprintf("http://%v/naming/%v/namespaces?lang=%s", c.Address, c.Version, lang)
	body, err := http.JSONFromNamespaces([]*apimodel.Namespace{{
		Name:    utils.NewStringValue("+$#@+"),
		Comment: utils.NewStringValue("test"),
		Owners:  utils.NewStringValue("test"),
	}})
	if err != nil {
		return nil, err
	}
	response, err := c.SendRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	ret := &apiservice.BatchWriteResponse{}
	if ierr := jsonpb.Unmarshal(response.Body, ret); ierr != nil {
		return nil, ierr
	}
	return ret, nil
}
