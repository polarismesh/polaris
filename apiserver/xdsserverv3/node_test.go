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

package xdsserverv3

import "testing"

func Test_parseNodeID(t *testing.T) {
	type args struct {
		nodeID string
	}
	tests := []struct {
		name                 string
		args                 args
		wantRunType          string
		wantPolarisNamespace string
		wantUuid             string
		wantHostIP           string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRunType, gotPolarisNamespace, gotUuid, gotHostIP := parseNodeID(tt.args.nodeID)
			if gotRunType != tt.wantRunType {
				t.Errorf("parseNodeID() gotRunType = %v, want %v", gotRunType, tt.wantRunType)
			}
			if gotPolarisNamespace != tt.wantPolarisNamespace {
				t.Errorf("parseNodeID() gotPolarisNamespace = %v, want %v", gotPolarisNamespace, tt.wantPolarisNamespace)
			}
			if gotUuid != tt.wantUuid {
				t.Errorf("parseNodeID() gotUuid = %v, want %v", gotUuid, tt.wantUuid)
			}
			if gotHostIP != tt.wantHostIP {
				t.Errorf("parseNodeID() gotHostIP = %v, want %v", gotHostIP, tt.wantHostIP)
			}
		})
	}
}
