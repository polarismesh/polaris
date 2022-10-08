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
	"testing"

	"github.com/polarismesh/polaris/common/model"
)

func Test_checkAnyElementExist(t *testing.T) {
	type args struct {
		userId     string
		waitSearch []model.ResourceEntry
		searchMaps *SearchMap
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkAnyElementExist(tt.args.userId, tt.args.waitSearch, tt.args.searchMaps); got != tt.want {
				t.Errorf("checkAnyElementExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
