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
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/polarismesh/polaris/common/utils"
)

const (
	FetchSuccess    = 200000 // 成功
	FetchForbiden   = 401000 // 无权限
	FetchBadRequest = 400001 // 分页参数错误
	FetchException  = 500000 // 服务端错误
)

// IPs
type IPs struct {
	Hosts   map[string]IP
	Mask    []IP
	Backoff *IP
}

// Fetcher fetcher by get cmdb data
type Fetcher interface {
	GetIPs() ([]IPInfo, IPs, error)
}

type fetcher struct {
	url   string
	token string
}

// GetIPs get all ips from server
func (f *fetcher) GetIPs() ([]IPInfo, IPs, error) {
	ret := IPs{
		Hosts: map[string]IP{},
		Mask:  make([]IP, 0, 8),
	}

	values, err := f.getFromRemote()
	if err != nil {
		return nil, IPs{}, err
	}

	for i := range values {
		item := values[i]
		if item.Type == Host {
			data, _ := NewIP(item)
			ret.Hosts[item.IP] = data
			continue
		}

		if item.Type == Mask {
			data, err := NewIP(item)
			if err != nil {
				return nil, ret, err
			}
			ret.Mask = append(ret.Mask, data)
			continue
		}

		if item.Type == Backoff {
			data, _ := NewIP(item)
			ret.Backoff = &data
		}
	}

	return values, ret, nil
}

func (f *fetcher) getFromRemote() ([]IPInfo, error) {
	if f.url == "" {
		return []IPInfo{}, nil
	}

	requestId := utils.NewUUID()

	total := 0
	curQuery := 0
	pageNo := 0
	first := true
	values := make([]IPInfo, 0, 1)
	for {
		if curQuery >= total && !first {
			break
		}
		pageNo++
		body, err := json.Marshal(Request{
			RequestID: requestId,
			PageNo:    int64(pageNo),
			PageSize:  100,
		})
		if err != nil {
			return nil, err
		}
		hreq, err := http.NewRequest(http.MethodPost, f.url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		hreq.Header.Set("Authorization", f.token)
		hreq.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(hreq)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		queryResp := &Response{}
		if err := json.Unmarshal(data, queryResp); err != nil {
			return nil, err
		}
		if queryResp.Code != FetchSuccess {
			return nil, errors.New(queryResp.Info)
		}
		if total == 0 {
			total = int(queryResp.Total)
			values = make([]IPInfo, 0, total)
		}
		curQuery += int(queryResp.Size)
		values = append(values, queryResp.Data...)
		first = false
	}

	return values, nil
}
