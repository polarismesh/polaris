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

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	status := 0
	start := time.Now()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v / request\n", time.Now().Format("2006-01-02 15:04:05"))
		host, _ := os.Hostname()
		if status == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "[%v] Internal Server Error, interval: %v", host,
				time.Now().Sub(start)/time.Nanosecond)
			start = time.Now()
		} else {
			fmt.Fprintln(w, fmt.Sprintf("[%v] hello", host))
		}
	})
	http.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		status = 1
		_, _ = w.Write([]byte("ok"))
	})
	http.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
		status = 0
		_, _ = w.Write([]byte("ok"))
	})
	http.HandleFunc("/healthCheck", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v /healthCheck request\n", time.Now().Format("2006-01-02 15:04:05"))
		if status == 1 {
			time.Sleep(5 * time.Second)
		} else {
			_, _ = w.Write([]byte("ok"))
		}
	})

	http.ListenAndServe(":8090", nil)
}
