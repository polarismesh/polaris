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
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCreateToken(t *testing.T) {
	AuthOption = DefaultAuthConfig()

	uid := "04ae4ead86e1ecf5811e32a9fbca9bfa"
	fmt.Printf("uid=%s\n", uid)

	token, err := createToken(uid, "")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("token=%s\n", token)

	password, err := bcrypt.GenerateFromPassword([]byte("polaris"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("password=%s\n", string(password))
}
