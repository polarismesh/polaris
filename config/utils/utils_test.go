/*
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
package utils

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
)

func TestCalMd5(t *testing.T) {
	assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", CalMd5(""))
	assert.Equal(t, "acbd18db4cc2f85cedef654fccc4a4d8", CalMd5("foo"))

	str := "38c5ee9532f037a20b93d0f804cf111fca4003e451d09a692d9dea8032308d9c64eda9047fcd5e850284a49b1a0cfb2ecd45"
	assert.Equal(t, "02f463eb799797e2a978fb1a2ae2991e", CalMd5(str))
}

func TestCheckResourceName(t *testing.T) {
	w := &wrappers.StringValue{Value: "123abc"}
	err := CheckResourceName(w)
	assert.Equal(t, err, nil)
}

func TestCheckFileName(t *testing.T) {
	w := &wrappers.StringValue{Value: "123abc.test.log"}
	err := CheckFileName(w)
	assert.Equal(t, err, nil)
}
