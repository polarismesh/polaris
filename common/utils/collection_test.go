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

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	commonhash "github.com/polarismesh/polaris/common/hash"
)

func Test_SegmentMap(t *testing.T) {
	segmentMap := NewSegmentMap[string, string](32, func(k string) int {
		return commonhash.Fnv32(k)
	})

	total := 100
	for i := 0; i < total; i++ {
		segmentMap.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
	}

	assert.Equal(t, uint64(total), segmentMap.Count())
	soltEntries := map[int]int{}
	for i := range segmentMap.solts {
		soltEntries[i] = len(segmentMap.solts[i])
	}

	t.Logf("%#v", soltEntries)

	key := fmt.Sprintf("key-%d", 0)
	val, exist := segmentMap.Get(key)
	assert.True(t, exist)
	assert.Equal(t, val, fmt.Sprintf("value-%d", 0))

	delRet := segmentMap.Del(key)
	assert.True(t, delRet)
	val, exist = segmentMap.Get(key)
	assert.False(t, exist)

	oldVal, ok := segmentMap.PutIfAbsent(key, key)
	assert.True(t, ok)
	assert.Equal(t, "", oldVal)

	oldVal, ok = segmentMap.PutIfAbsent(key, key)
	assert.False(t, ok)
	assert.Equal(t, key, oldVal)
}

func Test_SyncSegmentMap(t *testing.T) {
	segmentMap := NewSegmentMap[string, *SegmentMap[string, string]](32, func(k string) int {
		return commonhash.Fnv32(k)
	})

	total := 100
	for i := 0; i < total; i++ {
		go func(i int) {
			subMap, _ := segmentMap.ComputeIfAbsent(fmt.Sprintf("key-%d", i), func(k string) *SegmentMap[string, string] {
				return NewSegmentMap[string, string](128, func(k string) int {
					return commonhash.Fnv32(k)
				})
			})
			subMap.Put(fmt.Sprintf("key-%d", i), fmt.Sprintf("key-%d", i))
		}(i)
	}

	for {
		count := 0
		segmentMap.Range(func(k string, v *SegmentMap[string, string]) {
			v.Range(func(k string, v string) {
				t.Logf("%s %s", k, v)
			})
			count++
		})
		if count == total {
			break
		}
	}
}
