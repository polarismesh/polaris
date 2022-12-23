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

package memory

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/polarismesh/polaris/plugin/cmdb/memory/mock"
)

func Test_fetcher_GetIPs(t *testing.T) {
	t.Run("total=1000", func(t *testing.T) {
		total := 1000

		port, ln := mock.RunMockCMDBServer(total)
		t.Cleanup(func() {
			ln.Close()
			t.Log("close http server")
			time.Sleep(5 * time.Second)
		})
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		f := &fetcher{
			url: url,
		}
		values, _, err := f.GetIPs()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, total, len(values))
	})

	t.Run("total=1050", func(t *testing.T) {
		total := 1050
		port, ln := mock.RunMockCMDBServer(total)
		t.Cleanup(func() {
			ln.Close()
			t.Log("close http server")
			time.Sleep(5 * time.Second)
		})
		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		f := &fetcher{
			url: url,
		}
		values, _, err := f.GetIPs()
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, total, len(values))
	})

	t.Run("total=34", func(t *testing.T) {
		total := 34
		port, ln := mock.RunMockCMDBServer(total)
		t.Cleanup(func() {
			ln.Close()
			t.Log("close http server")
			time.Sleep(5 * time.Second)
		})

		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		f := &fetcher{
			url: url,
		}
		values, _, err := f.GetIPs()

		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, total, len(values))
	})
}
