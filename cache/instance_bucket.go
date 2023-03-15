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

package cache

import (
	"strconv"
	"sync"
)

type servicePortsBucket struct {
	lock sync.RWMutex
	// servicePorts service-id -> []port
	servicePorts map[string]map[string]struct{}
}

func (b *servicePortsBucket) reset() {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.servicePorts = make(map[string]map[string]struct{})
}

func (b *servicePortsBucket) appendPort(serviceID string, port int) {
	if serviceID == "" || port == 0 {
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	if _, ok := b.servicePorts[serviceID]; !ok {
		b.servicePorts[serviceID] = map[string]struct{}{}
	}

	ports := b.servicePorts[serviceID]
	ports[strconv.FormatInt(int64(port), 10)] = struct{}{}
}

func (b *servicePortsBucket) listPort(serviceID string) []string {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ret := make([]string, 0, 4)

	val, ok := b.servicePorts[serviceID]

	if !ok {
		return ret
	}

	for k := range val {
		ret = append(ret, k)
	}

	return ret
}
