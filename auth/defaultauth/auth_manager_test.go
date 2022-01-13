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

package defaultauth

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/polarismesh/polaris-server/cache"
	"github.com/polarismesh/polaris-server/plugin"
)

func Test_defaultAuthManager_ParseToken(t *testing.T) {
	type fields struct {
		userSvr     *userServer
		strategySvr *authStrategyServer
		cacheMgn    *cache.NamingCache
		authPlugin  plugin.Auth
	}
	type args struct {
		t string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    TokenInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authMgn := &defaultAuthManager{
				userSvr:     tt.fields.userSvr,
				strategySvr: tt.fields.strategySvr,
				cacheMgn:    tt.fields.cacheMgn,
				authPlugin:  tt.fields.authPlugin,
			}
			got, err := authMgn.ParseToken(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultAuthManager.ParseToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultAuthManager.ParseToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_SiezOfInerface(t *testing.T) {

	a := unsafe.Sizeof(new(interface{}))

	t.Log(a)
	fmt.Println(unsafe.Sizeof(new(interface{})))
}
