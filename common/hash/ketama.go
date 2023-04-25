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

package hash

import (
	"crypto/sha1"
	"fmt"
	"sort"
)

// Bucket single bucket of hash ring
type Bucket struct {
	Host   string
	Weight uint32
}

type continuumPoint struct {
	bucket Bucket
	point  uint
}

// Continuum consistent hash ring
type Continuum struct {
	ring points
}

type points []continuumPoint

// Less 比较大小
func (c points) Less(i, j int) bool { return c[i].point < c[j].point }

// Len 长度
func (c points) Len() int { return len(c) }

// Swap 交换
func (c points) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func sha1Digest(in string) []byte {
	h := sha1.New()
	h.Write([]byte(in))
	return h.Sum(nil)
}

func HashString(in string) uint {
	digest := sha1Digest(in)
	return uint(digest[3])<<24 | uint(digest[2])<<16 | uint(digest[1])<<8 | uint(digest[0])
}

// New hash ring
func New(buckets map[Bucket]bool) *Continuum {
	numBuckets := len(buckets)

	if numBuckets == 0 {
		return nil
	}

	ring := make(points, 0, numBuckets*160)

	var totalWeight uint32
	for bucket := range buckets {
		totalWeight += bucket.Weight
	}

	for bucket := range buckets {
		pct := float64(bucket.Weight) / float64(totalWeight)

		// this is the equivalent of C's promotion rules, but in Go, to maintain exact compatibility with the C library
		limit := int(pct * 40.0 * float64(numBuckets))

		for k := 0; k < limit; k++ {
			/* 40 hashes, 4 numbers per hash = 160 points per bucket */
			ss := fmt.Sprintf("%s-%d", bucket.Host, k)
			digest := sha1Digest(ss)

			for h := 0; h < 4; h++ {
				point := continuumPoint{
					point:  uint(digest[3+h*4])<<24 | uint(digest[2+h*4])<<16 | uint(digest[1+h*4])<<8 | uint(digest[h*4]),
					bucket: bucket,
				}
				ring = append(ring, point)
			}
		}
	}

	sort.Sort(ring)

	return &Continuum{
		ring: ring,
	}
}

// Hash hash string to lookup node
func (c *Continuum) Hash(h uint) string {
	if len(c.ring) == 0 {
		return ""
	}

	// the above md5 is way more expensive than this branch
	var i uint
	i = uint(sort.Search(len(c.ring), func(i int) bool { return c.ring[i].point >= h }))
	if i >= uint(len(c.ring)) {
		i = 0
	}

	return c.ring[i].bucket.Host
}
