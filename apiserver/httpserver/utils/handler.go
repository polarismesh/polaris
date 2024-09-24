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

package utils

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	apiconfig "github.com/polarismesh/specification/source/go/api/v1/config_manage"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/httpserver/i18n"
	api "github.com/polarismesh/polaris/common/api/v1"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	convert          MessageToCache
	protoCache       Cache
	enableProtoCache = false
	accesslog        = commonlog.GetScopeOrDefaultByName(commonlog.APIServerLoggerName)
)

// Handler HTTP请求/回复处理器
type Handler struct {
	Request  *restful.Request
	Response *restful.Response
}

// ParseArray 解析PB数组对象
func (h *Handler) ParseArray(createMessage func() proto.Message) (context.Context, error) {
	jsonDecoder := json.NewDecoder(h.Request.Request.Body)
	return h.parseArray(createMessage, jsonDecoder)
}

// ParseArrayByText 通过字符串解析PB数组对象
func (h *Handler) ParseArrayByText(createMessage func() proto.Message, text string) (context.Context, error) {
	jsonDecoder := json.NewDecoder(bytes.NewBuffer([]byte(text)))
	return h.parseArray(createMessage, jsonDecoder)
}

func (h *Handler) parseArray(createMessage func() proto.Message, jsonDecoder *json.Decoder) (context.Context, error) {
	requestID := h.Request.HeaderParameter("Request-Id")
	// read open bracket
	_, err := jsonDecoder.Token()
	if err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, err
	}
	for jsonDecoder.More() {
		protoMessage := createMessage()
		err := UnmarshalNext(jsonDecoder, protoMessage)
		if err != nil {
			accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
			return nil, err
		}
	}
	return h.ParseHeaderContext(), nil
}

// Parse 解析请求
func (h *Handler) Parse(message proto.Message) (context.Context, error) {
	requestID := h.Request.HeaderParameter("Request-Id")
	if err := Unmarshal(h.Request.Request.Body, message); err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, err
	}
	return h.ParseHeaderContext(), nil
}

// ParseHeaderContext 将http请求header中携带的用户信息提取出来
func (h *Handler) ParseHeaderContext() context.Context {
	requestID := h.Request.HeaderParameter("Request-Id")
	platformID := h.Request.HeaderParameter("Platform-Id")
	platformToken := h.Request.HeaderParameter("Platform-Token")
	token := h.Request.HeaderParameter("Polaris-Token")
	authToken := h.Request.HeaderParameter(utils.HeaderAuthTokenKey)

	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.StringContext("request-id"), requestID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-id"), platformID)
	ctx = context.WithValue(ctx, utils.StringContext("platform-token"), platformToken)
	ctx = context.WithValue(ctx, utils.ContextRequestHeaders, h.Request.Request.Header)
	ctx = context.WithValue(ctx, utils.ContextClientAddress, h.Request.Request.RemoteAddr)
	if token != "" {
		ctx = context.WithValue(ctx, utils.StringContext("polaris-token"), token)
	}
	if authToken != "" {
		ctx = context.WithValue(ctx, utils.ContextAuthTokenKey, authToken)
	}

	var operator string
	addrSlice := strings.Split(h.Request.Request.RemoteAddr, ":")
	if len(addrSlice) == 2 {
		operator = "HTTP:" + addrSlice[0]
		if platformID != "" {
			operator += "(" + platformID + ")"
		}
	}
	if staffName := h.Request.HeaderParameter("Staffname"); staffName != "" {
		operator = staffName
	}
	ctx = context.WithValue(ctx, utils.StringContext("operator"), operator)

	return ctx
}

// ParseFile 解析上传的配置文件
func (h *Handler) ParseFile() ([]*apiconfig.ConfigFile, error) {
	requestID := h.Request.HeaderParameter("Request-Id")
	h.Request.Request.Body = http.MaxBytesReader(h.Response, h.Request.Request.Body, utils.MaxRequestBodySize)

	file, fileHeader, err := h.Request.Request.FormFile(utils.ConfigFileFormKey)
	if err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, err
	}
	defer file.Close()

	accesslog.Info("[Config][Handler] parse upload file.",
		zap.String("filename", fileHeader.Filename),
		zap.Int64("filesize", fileHeader.Size),
		zap.String("fileheader", fmt.Sprintf("%v", fileHeader.Header)),
	)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
		return nil, err
	}
	filename := fileHeader.Filename
	contentType := http.DetectContentType(buf.Bytes())

	if contentType == "application/zip" && strings.HasSuffix(filename, ".zip") {
		return getConfigFilesFromZIP(buf.Bytes())
	}
	accesslog.Error("invalid content type",
		utils.ZapRequestID(requestID),
		zap.String("content-type", contentType),
		zap.String("filename", filename),
	)
	return nil, errors.New("invalid content type")

}

func getConfigFilesFromZIP(data []byte) ([]*apiconfig.ConfigFile, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	extractFileContent := func(f *zip.File) ([]byte, error) {
		rc, err := f.Open()
		if err != nil {
			accesslog.Error(err.Error(), zap.String("filename", f.Name))
			return nil, err
		}
		defer rc.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, rc); err != nil {
			accesslog.Error(err.Error(), zap.String("filename", f.Name))
			return nil, err
		}
		return buf.Bytes(), nil
	}
	var (
		configFiles []*apiconfig.ConfigFile
		metas       map[string]*utils.ConfigFileMeta
	)
	// 提取元数据文件
	for _, file := range zr.File {
		if file.Name == utils.ConfigFileMetaFileName {
			content, err := extractFileContent(file)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(content, &metas); err != nil {
				accesslog.Error(err.Error(), zap.String("filename", file.Name))
				return nil, err
			}
			break
		}
	}
	// 提取配置文件
	for _, file := range zr.File {
		// 跳过目录文件和元数据文件
		if file.FileInfo().IsDir() || file.Name == utils.ConfigFileMetaFileName {
			continue
		}
		// 提取文件内容
		content, err := extractFileContent(file)
		if err != nil {
			return nil, err
		}
		// 解析文件组和文件名
		var (
			group string
			name  string
		)
		tokens := strings.SplitN(file.Name, "/", 2)
		switch len(tokens) {
		case 2:
			group = tokens[0]
			name = tokens[1]
		case 1:
			name = tokens[0]
		default:
			accesslog.Error("invalid config file", zap.String("filename", file.Name))
			return nil, errors.New("invalid config file")
		}

		// 解析文件扩展名
		format := path.Ext(file.Name)
		if format == "" {
			format = utils.FileFormatText
		} else {
			format = format[1:]
		}
		cf := &apiconfig.ConfigFile{
			Group:   utils.NewStringValue(group),
			Name:    utils.NewStringValue(name),
			Content: utils.NewStringValue(string(content)),
			Format:  utils.NewStringValue(format),
		}
		if meta, ok := metas[file.Name]; ok {
			if meta.Comment != "" {
				cf.Comment = utils.NewStringValue(meta.Comment)
			}
			for k, v := range meta.Tags {
				cf.Tags = append(cf.Tags, &apiconfig.ConfigFileTag{
					Key:   utils.NewStringValue(k),
					Value: utils.NewStringValue(v),
				})
			}
		}
		configFiles = append(configFiles, cf)
	}
	return configFiles, nil
}

// WriteHeader 仅返回Code
func (h *Handler) WriteHeader(polarisCode uint32, httpStatus int) {
	requestID := h.Request.HeaderParameter(utils.PolarisRequestID)
	h.Request.SetAttribute(utils.PolarisCode, polarisCode) // api统计的时候，用该code

	// 对于非200000的返回，补充实际的code到header中
	if polarisCode != api.ExecuteSuccess {
		h.Response.AddHeader(utils.PolarisCode, fmt.Sprintf("%d", polarisCode))
		h.Response.AddHeader(utils.PolarisMessage, api.Code2Info(polarisCode))
	}
	h.Response.AddHeader("Request-Id", requestID)
	h.Response.WriteHeader(httpStatus)
}

// WriteHeaderAndProto 返回Code和Proto
func (h *Handler) WriteHeaderAndProto(obj api.ResponseMessage) {
	requestID := h.Request.HeaderParameter(utils.PolarisRequestID)
	h.Request.SetAttribute(utils.PolarisCode, obj.GetCode().GetValue())
	status := api.CalcCode(obj)

	if status != http.StatusOK {
		accesslog.Error(obj.String(), utils.ZapRequestID(requestID))
	}
	if code := obj.GetCode().GetValue(); code != api.ExecuteSuccess {
		h.Response.AddHeader(utils.PolarisCode, fmt.Sprintf("%d", code))
		h.Response.AddHeader(utils.PolarisMessage, api.Code2Info(code))
	}
	h.Response.AddHeader(utils.PolarisRequestID, requestID)
	h.Response.WriteHeader(status)

	if err := h.handleResponse(h.i18nAction(obj)); err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
	}
}

// WriteHeaderAndProtoV2 返回Code和Proto
func (h *Handler) WriteHeaderAndProtoV2(obj api.ResponseMessageV2) {
	requestID := h.Request.HeaderParameter(utils.PolarisRequestID)
	h.Request.SetAttribute(utils.PolarisCode, obj.GetCode())
	status := api.CalcCodeV2(obj)

	if status != http.StatusOK {
		accesslog.Error(obj.String(), utils.ZapRequestID(requestID))
	}
	if code := obj.GetCode(); code != api.ExecuteSuccess {
		h.Response.AddHeader(utils.PolarisCode, fmt.Sprintf("%d", code))
		h.Response.AddHeader(utils.PolarisMessage, api.Code2Info(code))
	}

	h.Response.AddHeader(utils.PolarisRequestID, requestID)
	h.Response.WriteHeader(status)

	m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
	err := m.Marshal(h.Response, obj)
	if err != nil {
		accesslog.Error(err.Error(), utils.ZapRequestID(requestID))
	}
}

// HTTPResponse http答复简单封装
func HTTPResponse(req *restful.Request, rsp *restful.Response, code uint32) {
	handler := &Handler{
		Request:  req,
		Response: rsp,
	}
	resp := api.NewResponse(apimodel.Code(code))
	handler.WriteHeaderAndProto(resp)
}

// i18nAction 依据resp.code进行国际化resp.info信息
// 当与header中的信息不匹配时, 则使用原文, 后续通过新定义code的方式增量解决
// 当header的msg 与 resp.info一致时, 根据resp.code国际化信息
func (h *Handler) i18nAction(obj api.ResponseMessage) api.ResponseMessage {
	hMsg := h.Response.Header().Get(utils.PolarisMessage)
	info := obj.GetInfo()
	if hMsg != info.GetValue() {
		return obj
	}
	code := obj.GetCode()
	msg, err := i18n.Translate(
		code.GetValue(), h.Request.QueryParameter("lang"), h.Request.HeaderParameter("Accept-Language"))
	if msg == "" || err != nil {
		return obj
	}
	*info = wrappers.StringValue{Value: msg}
	return obj
}

// ParseQueryParams 解析并获取HTTP的query params
func ParseQueryParams(req *restful.Request) map[string]string {
	queryParams := make(map[string]string)
	for key, value := range req.Request.URL.Query() {
		if len(value) > 0 {
			if key == "keys" || key == "values" {
				queryParams[key] = strings.Join(value, ",")
			} else {
				queryParams[key] = value[0] // 暂时默认只支持一个查询
			}
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

func (h *Handler) handleResponse(obj api.ResponseMessage) error {
	if !enableProtoCache {
		m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
		return m.Marshal(h.Response, obj)
	}
	cacheVal := convert(obj)
	if cacheVal == nil {
		m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
		return m.Marshal(h.Response, obj)
	}
	if saveVal := protoCache.Get(cacheVal.CacheType, cacheVal.Key); saveVal != nil {
		if len(saveVal.GetBuf()) > 0 {
			_, err := h.Response.Write(saveVal.GetBuf())
			return err
		}
		return nil
	}

	if err := cacheVal.Marshal(obj); err != nil {
		accesslog.Warn("[Api-http][ProtoCache] prepare message fail, direct send msg", zap.String("key", cacheVal.Key),
			zap.Error(err))
		m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
		return m.Marshal(h.Response, obj)
	}

	cacheVal, ok := protoCache.Put(cacheVal)
	if !ok || cacheVal == nil {
		accesslog.Warn("[Api-http][ProtoCache] put cache ignore", zap.String("key", cacheVal.Key),
			zap.String("cacheType", cacheVal.CacheType))
		m := jsonpb.Marshaler{Indent: " ", EmitDefaults: true}
		return m.Marshal(h.Response, obj)
	}
	if len(cacheVal.GetBuf()) > 0 {
		_, err := h.Response.Write(cacheVal.GetBuf())
		return err
	}
	return nil
}

func InitProtoCache(option map[string]interface{}, cacheTypes []string, discoverCacheConvert MessageToCache) {
	enableProtoCache = true
	cache, err := NewCache(option, cacheTypes)
	if err != nil {
		accesslog.Warn("[Api-http][Discover] new protobuf cache", zap.Error(err))
	}
	if cache != nil {
		protoCache = cache
		convert = discoverCacheConvert
	}
}

func UnmarshalNext(j *json.Decoder, m proto.Message) error {
	var jsonpbMarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
	return jsonpbMarshaler.UnmarshalNext(j, m)
}

func Unmarshal(j io.Reader, m proto.Message) error {
	var jsonpbMarshaler = jsonpb.Unmarshaler{AllowUnknownFields: true}
	return jsonpbMarshaler.Unmarshal(j, m)
}
