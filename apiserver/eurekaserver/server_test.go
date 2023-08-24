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

package eurekaserver

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parsePeersToReplicate(t *testing.T) {
	type args struct {
		defaultNamespace  string
		replicatePeerObjs []interface{}
	}

	defaultNamespace := "default"

	tests := []struct {
		name string
		args args
		want map[string][]string
	}{
		{
			name: "empty",
			args: args{
				defaultNamespace:  defaultNamespace,
				replicatePeerObjs: []interface{}{},
			},
			want: map[string][]string{},
		},
		{
			name: "single-default",
			args: args{
				defaultNamespace: defaultNamespace,
				replicatePeerObjs: []interface{}{
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
			},
			want: map[string][]string{
				defaultNamespace: {
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
			},
		},
		{
			name: "multi-namespace",
			args: args{
				defaultNamespace: defaultNamespace,
				replicatePeerObjs: []interface{}{
					"127.0.0.1:8761",
					"127.0.0.1:8762",
					map[interface{}]interface{}{
						"ns1": []interface{}{
							"127.0.0.1:8763",
							"127.0.0.1:8764",
						},
					},
					map[interface{}]interface{}{
						"ns2": []interface{}{
							"127.0.0.1:8765",
							"127.0.0.1:8766",
						},
					},
				},
			},
			want: map[string][]string{
				defaultNamespace: {
					"127.0.0.1:8761",
					"127.0.0.1:8762",
				},
				"ns1": {
					"127.0.0.1:8763",
					"127.0.0.1:8764",
				},
				"ns2": {
					"127.0.0.1:8765",
					"127.0.0.1:8766",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parsePeersToReplicate(tt.args.defaultNamespace, tt.args.replicatePeerObjs),
				"parsePeersToReplicate(%v, %v)", tt.args.defaultNamespace, tt.args.replicatePeerObjs)
		})
	}
}

func TestEurekaServer_Stop(t *testing.T) {
	t.Run("stop server", func(t *testing.T) {
		eurekaServer := &EurekaServer{
			server: &http.Server{
				Addr: ":8761",
			},
			workers: &ApplicationsWorkers{},
		}
		go eurekaServer.server.ListenAndServe()
		eurekaServer.Stop()
	})
}
