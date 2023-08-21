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

package sqldb

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// TestPlaceholdersN 构造占位符的测试
func TestPlaceholdersN(t *testing.T) {
	Convey("可以正常输出", t, func() {
		So(PlaceholdersN(-1), ShouldBeEmpty)
		So(PlaceholdersN(1), ShouldEqual, "?")
		So(PlaceholdersN(3), ShouldEqual, "?,?,?")
	})
}

func Test_Quick(t *testing.T) {
	slice := make([]string, 0, 1)
	slice = nil
	for i := range slice {
		t.Log(slice[i])
	}
}
