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
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/service/batch"
	"github.com/polarismesh/polaris/store"
)

// some options config
const (
	// QueryDefaultOffset default query offset
	QueryDefaultOffset = 0
	// QueryDefaultLimit default query limit
	QueryDefaultLimit = 100
	// QueryMaxLimit default query max
	QueryMaxLimit = 100

	// MaxMetadataLength metadata max length
	MaxMetadataLength = 64

	MaxBusinessLength   = 64
	MaxOwnersLength     = 1024
	MaxDepartmentLength = 1024
	MaxCommentLength    = 1024

	// service表
	MaxDbServiceNameLength      = 128
	MaxDbServiceNamespaceLength = 64
	MaxDbServicePortsLength     = 8192
	MaxDbServiceBusinessLength  = 128
	MaxDbServiceDeptLength      = 1024
	MaxDbServiceCMDBLength      = 1024
	MaxDbServiceCommentLength   = 1024
	MaxDbServiceOwnerLength     = 1024
	MaxDbServiceToken           = 2048

	// instance表
	MaxDbInsHostLength     = 128
	MaxDbInsProtocolLength = 32
	MaxDbInsVersionLength  = 32
	MaxDbInsLogicSetLength = 128

	// circuitbreaker表
	MaxDbCircuitbreakerName       = 128
	MaxDbCircuitbreakerNamespace  = 64
	MaxDbCircuitbreakerBusiness   = 64
	MaxDbCircuitbreakerDepartment = 1024
	MaxDbCircuitbreakerComment    = 1024
	MaxDbCircuitbreakerOwner      = 1024
	MaxDbCircuitbreakerVersion    = 32

	// platform表
	MaxPlatformIDLength     = 32
	MaxPlatformNameLength   = 128
	MaxPlatformDomainLength = 1024
	MaxPlatformQPS          = 65535

	// ratelimit表
	MaxDbRateLimitName = 64

	// MaxDbRoutingName routing_config_v2 表
	MaxDbRoutingName = 64
)

// checkResourceName 检查资源Name
var resourceNameRE = regexp.MustCompile("^[0-9A-Za-z-./:_]+$")

func checkResourceName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New(utils.NilErrString)
	}

	if len(name.GetValue()) == 0 {
		return errors.New(utils.EmptyErrString)
	}

	ok := resourceNameRE.MatchString(name.GetValue())
	if !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// checkResourceOwners 检查资源Owners
func checkResourceOwners(owners *wrappers.StringValue) error {
	if owners == nil {
		return errors.New(utils.NilErrString)
	}

	if owners.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	if utf8.RuneCountInString(owners.GetValue()) > MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}

// checkInstanceHost 检查服务实例Host
func checkInstanceHost(host *wrappers.StringValue) error {
	if host == nil {
		return errors.New(utils.NilErrString)
	}

	if host.GetValue() == "" {
		return errors.New(utils.EmptyErrString)
	}

	return nil
}

// checkInstancePort 检查服务实例Port
func checkInstancePort(port *wrappers.UInt32Value) error {
	if port == nil {
		return errors.New(utils.NilErrString)
	}

	if port.GetValue() <= 0 {
		return errors.New(utils.EmptyErrString)
	}

	return nil
}

// checkMetadata 检查metadata的个数; 最大是64个
// key/value是否符合要求
func checkMetadata(meta map[string]string) error {
	if meta == nil {
		return nil
	}

	if len(meta) > MaxMetadataLength {
		return errors.New("metadata is too long")
	}

	/*regStr := "^[0-9A-Za-z-._*]+$"
	  matchFunc := func(str string) error {
	  	if str == "" {
	  		return nil
	  	}
	  	ok, err := regexp.MatchString(regStr, str)
	  	if err != nil {
	  		log.Errorf("regexp match string(%s) err: %s", str, err.Error())
	  		return err
	  	}
	  	if !ok {
	  		log.Errorf("metadata string(%s) contains invalid character", str)
	  		return errors.New("contain invalid character")
	  	}
	  	return nil
	  }
	  for key, value := range meta {
	  	if err := matchFunc(key); err != nil {
	  		return err
	  	}
	  	if err := matchFunc(value); err != nil {
	  		return err
	  	}
	  }*/

	return nil
}

// checkQueryOffset 检查查询参数Offset
func checkQueryOffset(offset []string) (int, error) {
	if len(offset) == 0 {
		return 0, nil
	}

	if len(offset) > 1 {
		return 0, errors.New("unique")
	}

	value, err := strconv.Atoi(offset[0])
	if err != nil {
		return 0, err
	}

	if value < 0 {
		return 0, errors.New("invalid")
	}

	return value, nil
}

// checkQueryLimit 检查查询参数Limit
func checkQueryLimit(limit []string) (int, error) {
	if len(limit) == 0 {
		return MaxQuerySize, nil
	}

	if len(limit) > 1 {
		return 0, errors.New("unique")
	}

	value, err := strconv.Atoi(limit[0])
	if err != nil {
		return 0, err
	}

	if value < 0 {
		return 0, errors.New("invalid")
	}

	if value > MaxQuerySize {
		value = MaxQuerySize
	}

	return value, nil
}

// storeError2Response store code
func storeError2Response(err error) *api.Response {
	if err == nil {
		return nil
	}
	return api.NewResponseWithMsg(batch.StoreCode2APICode(err), err.Error())
}

// CalculateInstanceID 计算实例ID
// Deprecated: use common/utils.CalculateInstanceID instead
func CalculateInstanceID(namespace string, service string, vpcID string, host string, port uint32) (string, error) {
	h := sha1.New()
	var str string
	// 兼容带有vpcID的instance
	if vpcID == "" {
		str = fmt.Sprintf("%s##%s##%s##%d", namespace, service, host, port)
	} else {
		str = fmt.Sprintf("%s##%s##%s##%s##%d", namespace, service, vpcID, host, port)
	}

	if _, err := io.WriteString(h, str); err != nil {
		return "", err
	}

	out := hex.EncodeToString(h.Sum(nil))
	return out, nil
}

// CalculateRuleID 计算规则ID
// Deprecated: use common/utils.CalculateRuleID instead
func CalculateRuleID(name, namespace string) string {
	return name + "." + namespace
}

// ParseQueryOffset 格式化处理offset参数
// Deprecated: use common/utils.ParseQueryOffset instead
func ParseQueryOffset(offset string) (uint32, error) {
	if offset == "" {
		return QueryDefaultOffset, nil
	}

	tmp, err := strconv.ParseUint(offset, 10, 32)
	if err != nil {
		log.Warnf("[Server][Query] attribute(offset:%s) is invalid, parse err: %s",
			offset, err.Error())
		return 0, err
	}

	return uint32(tmp), nil
}

// ParseQueryLimit 格式化处理limit参数
// Deprecated: use common/utils.ParseQueryLimit instead
func ParseQueryLimit(limit string) (uint32, error) {
	if limit == "" {
		return QueryDefaultLimit, nil
	}

	tmp, err := strconv.ParseUint(limit, 10, 32)
	if err != nil {
		log.Errorf("[Server][Query] attribute(offset:%s) is invalid, parse err: %s",
			limit, err.Error())
		return 0, err
	}
	if tmp > QueryMaxLimit {
		tmp = QueryMaxLimit
	}

	return uint32(tmp), nil
}

// ParseOffsetAndLimit 统一格式化处理Offset和limit参数
// Deprecated: use common/utils.ParseOffsetAndLimit instead
func ParseOffsetAndLimit(query map[string]string) (uint32, uint32, error) {
	ofs, err := ParseQueryOffset(query["offset"])
	if err != nil {
		return 0, 0, err
	}
	delete(query, "offset")

	var lmt uint32
	lmt, err = ParseQueryLimit(query["limit"])
	if err != nil {
		return 0, 0, err
	}
	delete(query, "limit")

	return ofs, lmt, nil
}

// ParseInstanceArgs 解析服务实例的 ip 和 port 查询参数
// Deprecated: use common/utils.ParseInstanceArgs instead
func ParseInstanceArgs(query map[string]string) (*store.InstanceArgs, error) {
	if len(query) == 0 {
		return nil, nil
	}
	hosts, ok := query["host"]
	if !ok {
		return nil, fmt.Errorf("port parameter can not be used alone without host")
	}
	res := &store.InstanceArgs{}
	res.Hosts = strings.Split(hosts, ",")
	ports, ok := query["port"]
	if !ok {
		return res, nil
	}

	portSlices := strings.Split(ports, ",")
	for _, portStr := range portSlices {
		port, err := strconv.ParseUint(portStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%s can not parse as uint, err is %s", portStr, err.Error())
		}
		res.Ports = append(res.Ports, uint32(port))
	}
	return res, nil
}

// ParseRequestID 从ctx中获取Request-ID
// Deprecated: use common/utils.ParseRequestID instead
func ParseRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(utils.StringContext("request-id")).(string)
	return rid
}

// ParseToken 从ctx中获取token
// Deprecated: use common/utils.ParseToken instead
func ParseToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	token, _ := ctx.Value(utils.StringContext("polaris-token")).(string)
	return token
}

// ParseOperator 从ctx中获取operator
// Deprecated: use common/utils.ParseOperator instead
func ParseOperator(ctx context.Context) string {
	defaultOperator := "Polaris"
	if ctx == nil {
		return defaultOperator
	}

	if operator, _ := ctx.Value(utils.StringContext("operator")).(string); operator != "" {
		return operator
	}

	return defaultOperator
}

// ParsePlatformID 从ctx中获取Platform-Id
// Deprecated: use common/utils.ParsePlatformID instead
func ParsePlatformID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	pid, _ := ctx.Value(utils.StringContext("platform-id")).(string)
	return pid
}

// ParsePlatformToken 从ctx中获取Platform-Token
// Deprecated: use common/utils.ParsePlatformToken instead
func ParsePlatformToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	pToken, _ := ctx.Value(utils.StringContext("platform-token")).(string)
	return pToken
}

// ZapRequestID 生成Request-ID的日志描述
// Deprecated: use common/utils.ZapRequestID instead
func ZapRequestID(id string) zap.Field {
	return zap.String("request-id", id)
}

// ZapPlatformID 生成Platform-ID的日志描述
// Deprecated: use common/utils.ZapPlatformID instead
func ZapPlatformID(id string) zap.Field {
	return zap.String("platform-id", id)
}

// ZapInstanceID 生成instanceID的日志描述
// Deprecated: use common/utils.ZapInstanceID instead
func ZapInstanceID(id string) zap.Field {
	return zap.String("instance", id)
}

// CheckDbStrFieldLen 检查name字段是否超过DB中对应字段的最大字符长度限制
// Deprecated: use common/utils.CheckDbStrFieldLen instead
func CheckDbStrFieldLen(param *wrappers.StringValue, dbLen int) error {
	if param.GetValue() != "" && utf8.RuneCountInString(param.GetValue()) > dbLen {
		errMsg := fmt.Sprintf("length of %s is over %d", param.GetValue(), dbLen)
		return errors.New(errMsg)
	}
	return nil
}

// CheckDbMetaDataFieldLen 检查metadata的K,V是否超过DB中对应字段的最大字符长度限制
// Deprecated: use common/utils.CheckDbMetaDataFieldLen instead
func CheckDbMetaDataFieldLen(metaData map[string]string) error {
	for k, v := range metaData {
		if utf8.RuneCountInString(k) > 128 || utf8.RuneCountInString(v) > 4096 {
			errMsg := fmt.Sprintf("metadata:length of key(%s) or value(%s) exceeds size(key:128,value:4096)", k, v)
			return errors.New(errMsg)
		}
	}
	return nil
}

// verifyAuthByPlatform 使用平台ID鉴权
func (s *Server) verifyAuthByPlatform(ctx context.Context, sPlatformID string) bool {
	// 判断平台鉴权插件是否开启
	if s.auth == nil {
		return false
	}
	// 若服务无平台ID，则采用默认方式鉴权
	if sPlatformID == "" {
		return false
	}

	platformID := utils.ParsePlatformID(ctx)
	platformToken := utils.ParsePlatformToken(ctx)

	if s.auth.Allow(platformID, platformToken) && platformID == sPlatformID {
		return true
	}
	return false
}
