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

package defaultuser_test

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	defaultuser "github.com/polarismesh/polaris/auth/user"
)

// Test_CustomDesignSalt 主要用于有自定义salt需求的用户
func Test_CustomDesignSalt(t *testing.T) {
	salt := "polarismesh@2021"
	//  polaris 用户的ID
	uid := "polaris"
	fmt.Printf("uid=%s\n", uid)

	token, err := defaultuser.CreateToken(uid, "", salt)
	if err != nil {
		t.Fatal(err)
	}

	// 输出最终的 token 信息
	fmt.Printf("token=%s\n", token)

	// 对 polaris 用户的密码进行加密
	password, err := bcrypt.GenerateFromPassword([]byte("polaris"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	// 输出最终的密码值
	fmt.Printf("password=%s\n", string(password))

	time.Sleep(time.Second)
}

func TestCreateToken(t *testing.T) {
	salt := "polarismesh@2021"

	uid := "65e4789a6d5b49669adf1e9e8387549c"
	fmt.Printf("uid=%s\n", uid)

	token, err := defaultuser.CreateToken(uid, "", salt)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("token=%s\n", token)

	password, err := bcrypt.GenerateFromPassword([]byte("polaris"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("password=%s\n", string(password))

	time.Sleep(time.Second)
}

func TestDecodeToken(t *testing.T) {
	token := "o+Ad+6m9X2WOxxNrC1hgOhyiZ1L/ZHuvU6iqYRpyQAFRXHgIIH2ghUHTcaFsekUL1dT3IvDyap4R2WcSAL8="
	v, err := defaultuser.DecryptMessage([]byte("polaris@1ce410a2101348e9"), token)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(v)
}
