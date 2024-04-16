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

package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	restful "github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/apiserver/nacosserver/model"
	"github.com/polarismesh/polaris/common/utils"
)

// WrirteSimpleResponse .
func WrirteSimpleResponse(data string, code int, resp *restful.Response) {
	resp.WriteHeader(code)
	_, _ = resp.Write([]byte(data))
}

// WrirteNacosResponse .
func WrirteNacosResponse(data interface{}, resp *restful.Response) {
	_ = resp.WriteAsJson(data)
}

// WrirteNacosResponseWithCode .
func WrirteNacosResponseWithCode(code int, data interface{}, resp *restful.Response) {
	_ = resp.WriteAsJson(data)
}

// WrirteNacosErrorResponse .
func WrirteNacosErrorResponse(data error, resp *restful.Response) {
	if nerr, ok := data.(*model.NacosError); ok {
		resp.WriteHeader(int(nerr.ErrCode))
		_, _ = resp.Write([]byte(nerr.Error()))
		return
	}
	_ = resp.WriteError(http.StatusInternalServerError, data)
}

// Handler HTTP请求/回复处理器
type Handler struct {
	Request  *restful.Request
	Response *restful.Response
}

func (h *Handler) postParseMessage(requestID string) (context.Context, error) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)

	var operator string
	addrSlice := strings.Split(h.Request.Request.RemoteAddr, ":")
	if len(addrSlice) == 2 {
		operator = "HTTP:" + addrSlice[0]
	}
	ctx = context.WithValue(ctx, utils.StringContext("operator"), operator)

	return ctx, nil
}

// ParseHeaderContext 将http请求header中携带的用户信息提取出来
func (h *Handler) ParseHeaderContext() context.Context {
	requestID := h.Request.HeaderParameter("Request-Id")

	ctx := context.Background()
	if requestID == "" {
		requestID = utils.NewUUID()
	}
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)

	var operator string
	addrSlice := strings.Split(h.Request.Request.RemoteAddr, ":")
	if len(addrSlice) == 2 {
		operator = "HTTP:" + addrSlice[0]
	}
	ctx = context.WithValue(ctx, utils.StringContext("operator"), operator)
	ctx = context.WithValue(ctx, utils.ContextClientAddress, h.Request.Request.RemoteAddr)
	return ctx
}

func (h *Handler) ProcessZip(consumer func(f *zip.File, data []byte)) error {
	req := h.Request
	rsp := h.Response

	req.Request.Body = http.MaxBytesReader(rsp, req.Request.Body, utils.MaxRequestBodySize)

	file, _, err := req.Request.FormFile(utils.ConfigFileFormKey)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		return err
	}

	data := buf.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	extractFileContent := func(f *zip.File) ([]byte, error) {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, rc); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// 提取元数据文件
	for _, file := range zr.File {
		content, err := extractFileContent(file)
		if err != nil {
			return err
		}
		consumer(file, content)
	}

	return nil
}

// ParseQueryParams 解析并获取HTTP的query params
func ParseQueryParams(req *restful.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, value := range req.Request.URL.Query() {
		if len(value) > 0 {
			queryParams[key] = value[0] // 暂时默认只支持一个查询
		}
	}

	return queryParams
}

// ParseJsonBody parse http body as json object
func ParseJsonBody(req *restful.Request, value interface{}) error {
	body, err := io.ReadAll(req.Request.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, value); err != nil {
		return err
	}
	return nil
}

func Optional(req *restful.Request, key, defaultVal string) string {
	val := req.QueryParameter(key)
	val = strings.TrimSpace(val)
	if len(val) == 0 {
		return defaultVal
	}
	return val
}

func Required(req *restful.Request, key string) (string, error) {
	val := req.QueryParameter(key)
	val = strings.TrimSpace(val)
	if len(val) == 0 {
		return "", fmt.Errorf("key: %s required", key)
	}
	return val, nil
}

func RequiredInt(req *restful.Request, key string) (int, error) {
	strValue, err := Required(req, key)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strValue)
	if err != nil {
		return 0, fmt.Errorf("key: %s is not a number", key)
	}
	return value, nil
}
