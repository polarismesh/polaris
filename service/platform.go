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
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	platformFilterAttributes = map[string]bool{
		"id":         true,
		"name":       true,
		"owner":      true,
		"department": true,
		"offset":     true,
		"limit":      true,
	}
)

// CreatePlatforms 批量创建平台信息
func (s *Server) CreatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	if checkErr := checkBatchPlatform(req); checkErr != nil {
		return checkErr
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, platform := range req {
		response := s.CreatePlatform(ctx, platform)
		responses.Collect(response)
	}

	return api.FormatBatchWriteResponse(responses)
}

// CreatePlatform 创建单个平台
func (s *Server) CreatePlatform(ctx context.Context, req *api.Platform) *api.Response {
	requestID := ParseRequestID(ctx)

	// 参数检查
	if err := checkPlatformParams(req); err != nil {
		return err
	}

	// 检查平台信息是否存在
	platform, err := s.storage.GetPlatformById(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return api.NewPlatformResponse(api.StoreLayerException, req)
	}

	if platform != nil {
		return api.NewPlatformResponse(api.ExistedResource, req)
	}

	// 存储层操作
	data := createPlatformModel(req)
	if err := s.storage.CreatePlatform(data); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return wrapperPlatformStoreResponse(req, err)
	}

	msg := fmt.Sprintf("create platform: id=%s", req.GetId().GetValue())
	log.Info(msg, ZapRequestID(requestID))

	// todo 打印操作记录

	// 返回请求结果
	req.Token = utils.NewStringValue(data.Token)

	return api.NewPlatformResponse(api.ExecuteSuccess, req)
}

// UpdatePlatforms 批量修改平台
func (s *Server) UpdatePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	if checkErr := checkBatchPlatform(req); checkErr != nil {
		return checkErr
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, platform := range req {
		response := s.UpdatePlatform(ctx, platform)
		responses.Collect(response)
	}

	return api.FormatBatchWriteResponse(responses)
}

// UpdatePlatform 修改平台信息
func (s *Server) UpdatePlatform(ctx context.Context, req *api.Platform) *api.Response {
	requestID := ParseRequestID(ctx)

	// 参数检查
	if err := checkPlatformParams(req); err != nil {
		return err
	}
	// 检查token
	if token := parsePlatformToken(ctx, req); token == "" {
		return api.NewPlatformResponse(api.InvalidPlatformToken, req)
	}

	platform, resp := s.checkRevisePlatform(ctx, req)
	if resp != nil {
		return resp
	}

	// 判断是否需要更新
	needUpdate := s.updatePlatformAttribute(req, platform)
	if !needUpdate {
		log.Info("update platform data no change, no need update", ZapRequestID(requestID),
			zap.String("platform", req.String()))
		return api.NewPlatformResponse(api.NoNeedUpdate, req)
	}

	// 执行存储层操作
	if err := s.storage.UpdatePlatform(platform); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return wrapperPlatformStoreResponse(req, err)
	}

	msg := fmt.Sprintf("update platform: %+v", platform)
	log.Info(msg, ZapRequestID(requestID))
	// todo 操作记录

	return api.NewPlatformResponse(api.ExecuteSuccess, req)
}

// DeletePlatforms 批量删除平台信息
func (s *Server) DeletePlatforms(ctx context.Context, req []*api.Platform) *api.BatchWriteResponse {
	if checkErr := checkBatchPlatform(req); checkErr != nil {
		return checkErr
	}

	responses := api.NewBatchWriteResponse(api.ExecuteSuccess)
	for _, platform := range req {
		response := s.DeletePlatform(ctx, platform)
		responses.Collect(response)
	}

	return api.FormatBatchWriteResponse(responses)
}

// DeletePlatform 删除平台信息
func (s *Server) DeletePlatform(ctx context.Context, req *api.Platform) *api.Response {
	requestID := ParseRequestID(ctx)

	// 参数检查
	if err := checkDeletePlatformParams(ctx, req); err != nil {
		return err
	}

	// 检查平台是否存在并鉴权
	if _, resp := s.checkRevisePlatform(ctx, req); resp != nil {
		if resp.GetCode().GetValue() == api.NotFoundPlatform {
			return api.NewPlatformResponse(api.ExecuteSuccess, req)
		}
		return resp
	}

	// 执行存储层操作
	if err := s.storage.DeletePlatform(req.GetId().GetValue()); err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return wrapperPlatformStoreResponse(req, err)
	}

	msg := fmt.Sprintf("delete platform: id=%s", req.GetId().GetValue())
	log.Info(msg, ZapRequestID(requestID))

	// todo 操作记录

	return api.NewPlatformResponse(api.ExecuteSuccess, req)
}

// GetPlatforms 查询平台信息
func (s *Server) GetPlatforms(ctx context.Context, query map[string]string) *api.BatchQueryResponse {
	for key := range query {
		if _, ok := platformFilterAttributes[key]; !ok {
			log.Errorf("get platforms attribute(%s) is not allowed", key)
			return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, key+" is not allowed")
		}
	}

	// 处理offset和limit
	offset, limit, err := ParseOffsetAndLimit(query)
	if err != nil {
		return api.NewBatchQueryResponseWithMsg(api.InvalidParameter, err.Error())
	}

	total, platforms, err := s.storage.GetPlatforms(query, offset, limit)
	if err != nil {
		log.Errorf("get platforms store err: %s", err.Error())
		return api.NewBatchQueryResponse(api.StoreLayerException)
	}

	resp := api.NewBatchQueryResponse(api.ExecuteSuccess)
	resp.Amount = utils.NewUInt32Value(total)
	resp.Size = utils.NewUInt32Value(uint32(len(platforms)))
	resp.Platforms = platforms2API(platforms)
	return resp
}

// GetPlatformToken 查询平台Token
func (s *Server) GetPlatformToken(ctx context.Context, req *api.Platform) *api.Response {
	// 参数检查
	if err := checkDeletePlatformParams(ctx, req); err != nil {
		return err
	}

	// 检查平台是否存在并鉴权
	platform, resp := s.checkRevisePlatform(ctx, req)
	if resp != nil {
		return resp
	}

	req.Token = utils.NewStringValue(platform.Token)
	return api.NewPlatformResponse(api.ExecuteSuccess, req)
}

// checkBatchPlatform 检查批量请求
func checkBatchPlatform(req []*api.Platform) *api.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(api.EmptyRequest)
	}

	if len(req) > MaxBatchSize {
		return api.NewBatchWriteResponse(api.BatchSizeOverLimit)
	}

	return nil
}

// checkPlatformParams 检查创建/修改平台参数
func checkPlatformParams(req *api.Platform) *api.Response {
	if req == nil {
		return api.NewPlatformResponse(api.EmptyRequest, req)
	}

	if err := checkPlatformID(req.GetId()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformID, req, err.Error())
	}

	if err := checkPlatformName(req.GetName()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformName, req, err.Error())
	}

	if err := checkPlatformDomain(req.GetDomain()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformDomain, req, err.Error())
	}

	if err := checkPlatformQPS(req.GetQps()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformQPS, req, err.Error())
	}

	if err := checkResourceOwners(req.GetOwner()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformOwner, req, err.Error())
	}

	if err := checkPlatformDepartment(req.GetDepartment()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformDepartment, req, err.Error())
	}

	if err := checkPlatformComment(req.GetComment()); err != nil {
		return api.NewPlatformResponseWithMsg(api.InvalidPlatformComment, req, err.Error())
	}

	return nil
}

// checkDeletePlatformParams 检查删除平台参数
func checkDeletePlatformParams(ctx context.Context, req *api.Platform) *api.Response {
	if req == nil {
		return api.NewPlatformResponse(api.EmptyRequest, req)
	}

	if err := checkPlatformID(req.GetId()); err != nil {
		return api.NewPlatformResponse(api.InvalidPlatformID, req)
	}

	// 检查token
	if token := parsePlatformToken(ctx, req); token == "" {
		return api.NewPlatformResponse(api.InvalidPlatformToken, req)
	}

	return nil
}

// checkRevisePlatform 修改和删除平台信息的公共检查
func (s *Server) checkRevisePlatform(ctx context.Context, req *api.Platform) (*model.Platform, *api.Response) {
	requestID := ParseRequestID(ctx)

	// 检查平台是否存在
	platform, err := s.storage.GetPlatformById(req.GetId().GetValue())
	if err != nil {
		log.Error(err.Error(), ZapRequestID(requestID))
		return nil, api.NewPlatformResponse(api.StoreLayerException, req)
	}
	if platform == nil {
		return nil, api.NewPlatformResponse(api.NotFoundPlatform, req)
	}

	return platform, nil
}

// checkPlatformID 检查平台ID
func checkPlatformID(id *wrappers.StringValue) error {
	if id == nil {
		return errors.New("id is nil")
	}

	if id.GetValue() == "" {
		return errors.New("id is empty")
	}

	// 允许0-9 a-z A-Z - . _
	regStr := "^[0-9A-Za-z-._]+$"
	ok, err := regexp.MatchString(regStr, id.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("platform id contains invalid character")
	}

	if utf8.RuneCountInString(id.GetValue()) > MaxPlatformIDLength {
		return errors.New("platform id too long")
	}

	return nil
}

// checkPlatformName 检查平台Name
func checkPlatformName(name *wrappers.StringValue) error {
	if name.GetValue() == "" {
		return errors.New("name is empty")
	}

	if utf8.RuneCountInString(name.GetValue()) > MaxPlatformNameLength {
		return errors.New("name too long")
	}

	return nil
}

// checkPlatformDomain 检查平台域名
func checkPlatformDomain(domain *wrappers.StringValue) error {
	if domain.GetValue() == "" {
		return errors.New("domain is empty")
	}

	if utf8.RuneCountInString(domain.GetValue()) > MaxPlatformDomainLength {
		return errors.New("domain too long")
	}

	return nil
}

// checkPlatformQPS 检查QPS
func checkPlatformQPS(qps *wrappers.UInt32Value) error {
	if qps.GetValue() == 0 {
		return errors.New("qps is empty")
	}

	if qps.GetValue() > MaxPlatformQPS {
		return errors.New("qps too long")
	}
	return nil
}

// checkPlatformDepartment 检查部门
func checkPlatformDepartment(department *wrappers.StringValue) error {
	if department.GetValue() == "" {
		return errors.New("department is empty")
	}

	if utf8.RuneCountInString(department.GetValue()) > MaxDepartmentLength {
		return errors.New("department too long")
	}

	return nil
}

// checkPlatformComment 检查描述
func checkPlatformComment(comment *wrappers.StringValue) error {
	if comment.GetValue() == "" {
		return errors.New("comment is empty")
	}

	if utf8.RuneCountInString(comment.GetValue()) > MaxCommentLength {
		return errors.New("comment too long")
	}
	return nil
}

// createPlatformModel 创建存储层模型
func createPlatformModel(req *api.Platform) *model.Platform {
	platform := &model.Platform{
		ID:         req.GetId().GetValue(),
		Name:       req.GetName().GetValue(),
		Domain:     req.GetDomain().GetValue(),
		QPS:        req.GetQps().GetValue(),
		Token:      utils.NewUUID(),
		Owner:      req.GetOwner().GetValue(),
		Department: req.GetDepartment().GetValue(),
		Comment:    req.GetComment().GetValue(),
	}

	return platform
}

// platforms2API platform数组转换为[]*api.Platform
func platforms2API(platforms []*model.Platform) []*api.Platform {
	out := make([]*api.Platform, 0, len(platforms))
	for _, entry := range platforms {
		out = append(out, platform2Api(entry))
	}

	return out
}

// platform2Api model.Platform转为api.Platform
func platform2Api(platform *model.Platform) *api.Platform {
	if platform == nil {
		return nil
	}

	// token不返回
	out := &api.Platform{
		Id:         utils.NewStringValue(platform.ID),
		Name:       utils.NewStringValue(platform.Name),
		Domain:     utils.NewStringValue(platform.Domain),
		Qps:        utils.NewUInt32Value(platform.QPS),
		Owner:      utils.NewStringValue(platform.Owner),
		Department: utils.NewStringValue(platform.Department),
		Comment:    utils.NewStringValue(platform.Comment),
		Ctime:      utils.NewStringValue(commontime.Time2String(platform.CreateTime)),
		Mtime:      utils.NewStringValue(commontime.Time2String(platform.ModifyTime)),
	}

	return out
}

// updatePlatformAttribute 修改平台字段
func (s *Server) updatePlatformAttribute(req *api.Platform, platform *model.Platform) bool {
	needUpdate := false

	if req.GetName() != nil && req.GetName().GetValue() != platform.Name {
		platform.Name = req.GetName().GetValue()
		needUpdate = true
	}

	if req.GetDomain() != nil && req.GetDomain().GetValue() != platform.Domain {
		platform.Domain = req.GetDomain().GetValue()
		needUpdate = true
	}

	if req.GetQps() != nil && req.GetQps().GetValue() != platform.QPS {
		platform.QPS = req.GetQps().GetValue()
		needUpdate = true
	}

	if req.GetOwner() != nil && req.GetOwner().GetValue() != platform.Owner {
		platform.Owner = req.GetOwner().GetValue()
		needUpdate = true
	}

	if req.GetDepartment() != nil && req.GetDepartment().GetValue() != platform.Department {
		platform.Department = req.GetDepartment().GetValue()
		needUpdate = true
	}

	if req.GetComment() != nil && req.GetComment().GetValue() != platform.Comment {
		platform.Comment = req.GetComment().GetValue()
		needUpdate = true
	}

	return needUpdate
}

// wrapperPlatformStoreResponse 封装存储层错误
func wrapperPlatformStoreResponse(platform *api.Platform, err error) *api.Response {
	resp := storeError2Response(err)
	if resp == nil {
		return nil
	}

	resp.Platform = platform
	return resp
}

// parsePlatformToken 获取平台的token信息
func parsePlatformToken(ctx context.Context, req *api.Platform) string {
	if token := req.GetToken().GetValue(); token != "" {
		return token
	}

	return ParseToken(ctx)
}
