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

package config

import (
	"log"
	"strconv"
	"sync"
	"testing"
)

func TestCenter_WatchEvent(t *testing.T) {
	c := NewEventCenter()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()
			cb := func(event Event) bool {
				event.Message = i
				log.Println("handler event: ", event.EventType, "msg:", event.Message)
				return true
			}

			eventType := "test_" + strconv.Itoa(i)
			log.Println("eventType2: ", eventType)
			c.WatchEvent(eventType, cb)
		}(i, &wg)

		go func(i int, wg *sync.WaitGroup) {
			defer wg.Done()

			eventType := "test_" + strconv.Itoa(i)
			log.Println("eventType: ", eventType)
			c.handleEvent(Event{
				EventType: eventType,
			})
		}(i, &wg)
	}

	wg.Wait()
	log.Println("test event watch end")
}
