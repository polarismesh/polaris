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

package eurekaserver

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
)

// hashKey in map
func hasKey(values map[string]bool, key string) bool {
	_, ok := values[key]
	return ok
}

// ObjectToString interface type to string
func ObjectToString(value interface{}) string {
	switch m := value.(type) {
	case string:
		return m
	case int:
		return strconv.Itoa(m)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func getParamFromEurekaRequestHeader(req *restful.Request, headerName string) string {
	headerValue := req.HeaderParameter(headerName)
	if len(headerValue) > 0 {
		return headerValue
	} else {
		headerValue = req.HeaderParameter(strings.ToLower(headerName))
		return headerValue
	}

}

func getAuthFromEurekaRequestHeader(req *restful.Request) (string, error) {
	token := ""
	basicInfo := strings.TrimPrefix(req.Request.Header.Get("Authorization"), "Basic ")
	if len(basicInfo) != 0 {
		ret, err := base64.StdEncoding.DecodeString(basicInfo)
		if err != nil {
			return "", err
		}
		info := string(ret)
		token = strings.Split(info, ":")[1]
	}
	return token, nil
}

func writeHeader(httpStatus int, rsp *restful.Response) {
	rsp.AddHeader(restful.HEADER_ContentType, restful.MIME_XML)
	rsp.WriteHeader(httpStatus)
}
