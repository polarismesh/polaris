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

package srand

import (
	"math/rand"
	"testing"
)

func BenchmarkSingleCore(b *testing.B) {

	b.Run("scalable_rand-Intn()", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Intn(1000)
		}
	})

	b.Run("math/rand-Intn()", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = rand.Intn(1000)
		}
	})

}

func BenchmarkMultipleCore(b *testing.B) {

	b.Run("scalable_rand-Intn()", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = Intn(1000)
			}
		})
	})

	b.Run("math/rand-Intn()", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = rand.Intn(1000)
			}
		})
	})

}
