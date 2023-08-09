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

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_serviceNamespaceBucket(t *testing.T) {
	mockData := genModelService(10)
	bucket := newServiceNamespaceBucket()

	for key := range mockData {
		bucket.addService(mockData[key])
	}
	bucket.reloadRevision()

	oldRevision := bucket.revision
	assert.NotEmpty(t, bucket.revision)
	bucket.reloadRevision()
	assert.Equal(t, oldRevision, bucket.revision)

	oldRevision = bucket.revision

	newMockData := genModelServiceByNamespace(10, "MockNamespace")
	for key := range newMockData {
		bucket.addService(newMockData[key])
	}

	bucket.reloadRevision()
	assert.NotEmpty(t, bucket.revision)
	assert.NotEqual(t, oldRevision, bucket.revision)

	allRevision, ret := bucket.ListAllServices()
	assert.Equal(t, int64(len(mockData)+len(newMockData)), int64(len(ret)))

	oneRevision, ret := bucket.ListServices("MockNamespace")
	assert.Equal(t, int64(len(newMockData)), int64(len(ret)))

	oldRevision = bucket.revision
	bucket.removeService(newMockData["MockNamespace-ID-1"])
	bucket.reloadRevision()
	assert.NotEmpty(t, bucket.revision)
	assert.NotEqual(t, oldRevision, bucket.revision)
	allRevision2, ret := bucket.ListAllServices()
	assert.NotEqual(t, allRevision, allRevision2)
	assert.Equal(t, int64(len(mockData)+len(newMockData)-1), int64(len(ret)))

	oneRevision2, ret := bucket.ListServices("MockNamespace")
	assert.NotEqual(t, oneRevision, oneRevision2)
	assert.Equal(t, int64(len(newMockData)-1), int64(len(ret)))
}
