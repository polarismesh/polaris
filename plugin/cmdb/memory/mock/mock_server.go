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

package mock

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

// IPType ip type
type IPType string

const (
	Host    IPType = "host"
	Mask    IPType = "mask"
	Backoff IPType = "backoff"
)

// Request request cmdb data
type Request struct {
	RequestID string `json:"request_id"`
	PageNo    int64  `json:"page_no"`
	PageSize  int64  `json:"page_size"`
}

// Response response cmdb data
type Response struct {
	Total    int      `json:"total"`
	Size     int      `json:"size"`
	Code     int      `json:"code"`
	Info     string   `json:"info"`
	Priority string   `json:"priority"`
	Data     []IPInfo `json:"data"`
}

// IPInfo ip info
type IPInfo struct {
	IP     string       `json:"ip"`
	Type   IPType       `json:"type"`
	Region LocationInfo `json:"region"`
	Zone   LocationInfo `json:"zone"`
	Campus LocationInfo `json:"campus"`
}

// LocationInfo
type LocationInfo struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

func RunMockCMDBServer(cnt int) (int, net.Listener) {
	initData(cnt)
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 0))
	if err != nil {
		log.Fatalf("[ERROR]fail to listen tcp, err is %v", err)
	}

	port := ln.Addr().(*net.TCPAddr).Port
	log.Printf("listen port : %d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)
	go func() {
		_ = http.Serve(ln, mux)
	}()
	return port, ln
}

var (
	IPInfos []IPInfo
)

func initData(cnt int) {
	IPInfos = []IPInfo{}
	for i := 0; i < cnt; i++ {
		ipv4 := IPv4Int(RandomIpv4Int())

		IPInfos = append(IPInfos, IPInfo{
			IP:   ipv4.ip().String(),
			Type: Host,
			Region: LocationInfo{
				Name: "ap-gz",
			},
			Zone: LocationInfo{
				Name: fmt.Sprintf("ap-gz-%d", i),
				Metadata: map[string]interface{}{
					"id": i,
				},
			},
		})
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		resp := Response{
			Code: 500001,
			Info: err.Error(),
		}

		if ret, err := json.Marshal(resp); err != nil {
			log.Printf("json marshal %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			_, _ = w.Write(ret)
		}
		return
	}

	req := Request{}
	if err := json.Unmarshal(data, &req); err != nil {
		resp := Response{
			Code: 500001,
			Info: err.Error(),
		}

		if ret, err := json.Marshal(resp); err != nil {
			log.Printf("json marshal %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			_, _ = w.Write(ret)
		}
		return
	}

	pageNo := req.PageNo
	pageSize := req.PageSize

	offset := (pageNo - 1) * pageSize
	end := offset + pageSize
	if int(offset) > len(IPInfos) {
		resp := Response{
			Code:  200000,
			Total: len(IPInfos),
			Size:  0,
			Data:  []IPInfo{},
		}

		if ret, err := json.Marshal(resp); err != nil {
			log.Printf("json marshal %+v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			_, _ = w.Write(ret)
		}
		return
	}
	if int(end) > len(IPInfos) {
		end = int64(len(IPInfos))
	}

	values := IPInfos[offset:end]

	resp := Response{
		Code:  200000,
		Total: len(IPInfos),
		Size:  len(values),
		Data:  values,
	}

	if ret, err := json.Marshal(resp); err != nil {
		log.Printf("json marshal %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		_, _ = w.Write(ret)
	}
	return
}

type IPv4Int uint32

func (i IPv4Int) ip() net.IP {
	ip := make(net.IP, net.IPv6len)
	copy(ip, net.IPv4zero)
	binary.BigEndian.PutUint32(ip.To4(), uint32(i))
	return ip.To16()
}

func RandomIpv4Int() uint32 {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()
}
