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

package namespace

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	commonstore "github.com/polarismesh/polaris/common/store"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var _ NamespaceOperateServer = (*Server)(nil)

func (s *Server) allowAutoCreate() bool {
	return s.cfg.AutoCreate
}

func AllowAutoCreate(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, model.ContextKeyAutoCreateNamespace{}, true)
	return ctx
}

// CreateNamespaces 批量创建命名空间
func (s *Server) CreateNamespaces(ctx context.Context, req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	if checkError := checkBatchNamespace(req); checkError != nil {
		return checkError
	}

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, namespace := range req {
		response := s.CreateNamespace(ctx, namespace)
		api.Collect(responses, response)
	}

	return responses
}

// CreateNamespaceIfAbsent 创建命名空间，如果不存在
func (s *Server) CreateNamespaceIfAbsent(ctx context.Context, req *apimodel.Namespace) (string, *apiservice.Response) {
	if resp := checkCreateNamespace(req); resp != nil {
		return "", resp
	}
	name := req.GetName().GetValue()
	val, err := s.loadNamespace(name)
	if err != nil {
		return name, nil
	}
	if val == "" && !s.allowAutoCreate() {
		ctxVal := ctx.Value(model.ContextKeyAutoCreateNamespace{})
		if ctxVal == nil || ctxVal.(bool) != true {
			return "", api.NewResponse(apimodel.Code_NotFoundNamespace)
		}
	}
	ret, err, _ := s.createNamespaceSingle.Do(name, func() (interface{}, error) {
		return s.CreateNamespace(ctx, req), nil
	})
	if err != nil {
		return "", api.NewResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}
	var (
		resp = ret.(*apiservice.Response)
		code = resp.GetCode().GetValue()
	)
	if code == uint32(apimodel.Code_ExecuteSuccess) || code == uint32(apimodel.Code_ExistedResource) {
		return name, nil
	}
	return "", resp
}

// CreateNamespace 创建单个命名空间
func (s *Server) CreateNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	// 参数检查
	if checkError := checkCreateNamespace(req); checkError != nil {
		return checkError
	}

	namespaceName := req.GetName().GetValue()

	// 检查是否存在
	namespace, err := s.storage.GetNamespace(namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if namespace != nil {
		return api.NewNamespaceResponse(apimodel.Code_ExistedResource, req)
	}

	//
	data := s.createNamespaceModel(req)

	// 存储层操作
	if err := s.storage.AddNamespace(data); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}

	log.Info("create namespace", utils.RequestID(ctx), zap.String("name", namespaceName))
	out := &apimodel.Namespace{
		Name:  req.GetName(),
		Token: utils.NewStringValue(data.Token),
	}

	s.RecordHistory(namespaceRecordEntry(ctx, req, model.OCreate))
	_ = s.afterNamespaceResource(ctx, req, data, false)
	return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, out)
}

/**
 * @brief 创建存储层命名空间模型
 */
func (s *Server) createNamespaceModel(req *apimodel.Namespace) *model.Namespace {
	namespace := &model.Namespace{
		Name:            req.GetName().GetValue(),
		Comment:         req.GetComment().GetValue(),
		Owner:           req.GetOwners().GetValue(),
		Token:           utils.NewUUID(),
		ServiceExportTo: model.ExportToMap(req.GetServiceExportTo()),
	}
	return namespace
}

// DeleteNamespaces 批量删除命名空间
func (s *Server) DeleteNamespaces(ctx context.Context, req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	if checkError := checkBatchNamespace(req); checkError != nil {
		return checkError
	}

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, namespace := range req {
		response := s.DeleteNamespace(ctx, namespace)
		api.Collect(responses, response)
	}

	return responses
}

// DeleteNamespace 删除单个命名空间
func (s *Server) DeleteNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	requestID, _ := ctx.Value(utils.StringContext("request-id")).(string)

	// 参数检查
	if checkError := checkReviseNamespace(ctx, req); checkError != nil {
		return checkError
	}

	tx, err := s.storage.CreateTransaction()
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	defer func() { _ = tx.Commit() }()

	// 检查是否存在
	namespace, err := tx.LockNamespace(req.GetName().GetValue())
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if namespace == nil {
		return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, req)
	}

	// // 鉴权
	// if ok := s.authority.VerifyNamespace(namespace.Token, parseNamespaceToken(ctx, req)); !ok {
	// 	return api.NewNamespaceResponse(api.Unauthorized, req)
	// }

	// 判断属于该命名空间的服务是否都已经被删除
	total, err := s.getServicesCountWithNamespace(namespace.Name)
	if err != nil {
		log.Error("get services count with namespace err",
			utils.ZapRequestID(requestID),
			zap.String("err", err.Error()))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if total != 0 {
		log.Error("the removed namespace has remain services", utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(apimodel.Code_NamespaceExistedServices, req)
	}

	// 判断属于该命名空间的服务是否都已经被删除
	total, err = s.getConfigGroupCountWithNamespace(namespace.Name)
	if err != nil {
		log.Error("get config group count with namespace err",
			utils.ZapRequestID(requestID),
			zap.String("err", err.Error()))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if total != 0 {
		log.Error("the removed namespace has remain config-group", utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(apimodel.Code_NamespaceExistedConfigGroups, req)
	}

	// 存储层操作
	if err := tx.DeleteNamespace(namespace.Name); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(requestID))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}

	s.caches.Service().CleanNamespace(namespace.Name)

	log.Info("delete namespace", utils.RequestID(ctx), zap.String("name", namespace.Name))
	s.RecordHistory(namespaceRecordEntry(ctx, req, model.ODelete))

	_ = s.afterNamespaceResource(ctx, req, &model.Namespace{Name: req.GetName().GetValue()}, true)

	return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, req)
}

// UpdateNamespaces 批量修改命名空间
func (s *Server) UpdateNamespaces(ctx context.Context, req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	if checkError := checkBatchNamespace(req); checkError != nil {
		return checkError
	}

	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for _, namespace := range req {
		response := s.UpdateNamespace(ctx, namespace)
		api.Collect(responses, response)
	}

	return responses
}

// UpdateNamespace 修改单个命名空间
func (s *Server) UpdateNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	// 参数检查
	if resp := checkReviseNamespace(ctx, req); resp != nil {
		return resp
	}

	// 权限校验
	namespace, resp := s.checkNamespaceAuthority(ctx, req)
	if resp != nil {
		return resp
	}

	rid := utils.ParseRequestID(ctx)
	// 修改
	s.updateNamespaceAttribute(req, namespace)

	// 存储层操作
	if err := s.storage.UpdateNamespace(namespace); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}

	msg := fmt.Sprintf("update namespace: name=%s", namespace.Name)
	log.Info(msg, utils.ZapRequestID(rid))
	s.RecordHistory(namespaceRecordEntry(ctx, req, model.OUpdate))

	if err := s.afterNamespaceResource(ctx, req, namespace, false); err != nil {
		return api.NewNamespaceResponse(apimodel.Code_ExecuteException, req)
	}

	return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, req)
}

/**
 * @brief 修改命名空间属性
 */
func (s *Server) updateNamespaceAttribute(req *apimodel.Namespace, namespace *model.Namespace) {
	if req.GetComment() != nil {
		namespace.Comment = req.GetComment().GetValue()
	}

	if req.GetOwners() != nil {
		namespace.Owner = req.GetOwners().GetValue()
	}

	exportTo := map[string]struct{}{}
	for i := range req.GetServiceExportTo() {
		exportTo[req.GetServiceExportTo()[i].GetValue()] = struct{}{}
	}

	namespace.ServiceExportTo = exportTo
}

// UpdateNamespaceToken 更新命名空间token
func (s *Server) UpdateNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	if resp := checkReviseNamespace(ctx, req); resp != nil {
		return resp
	}
	namespace, resp := s.checkNamespaceAuthority(ctx, req)
	if resp != nil {
		return resp
	}

	rid := utils.ParseRequestID(ctx)
	// 生成token
	token := utils.NewUUID()

	// 存储层操作
	if err := s.storage.UpdateNamespaceToken(namespace.Name, token); err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}

	msg := fmt.Sprintf("update namespace token: name=%s", namespace.Name)
	log.Info(msg, utils.ZapRequestID(rid))
	s.RecordHistory(namespaceRecordEntry(ctx, req, model.OUpdateToken))

	out := &apimodel.Namespace{
		Name:  req.GetName(),
		Token: utils.NewStringValue(token),
	}

	return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, out)
}

// GetNamespaces 查询命名空间
func (s *Server) GetNamespaces(ctx context.Context, query map[string][]string) *apiservice.BatchQueryResponse {
	filter, offset, limit, checkError := checkGetNamespace(query)
	if checkError != nil {
		return checkError
	}

	namespaces, amount, err := s.storage.GetNamespaces(filter, offset, limit)
	if err != nil {
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	out := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	out.Amount = utils.NewUInt32Value(amount)
	out.Size = utils.NewUInt32Value(uint32(len(namespaces)))
	var totalServiceCount, totalInstanceCount, totalHealthInstanceCount uint32
	for _, namespace := range namespaces {
		nsCntInfo := s.caches.Service().GetNamespaceCntInfo(namespace.Name)
		api.AddNamespace(out, &apimodel.Namespace{
			Id:                       utils.NewStringValue(namespace.Name),
			Name:                     utils.NewStringValue(namespace.Name),
			Comment:                  utils.NewStringValue(namespace.Comment),
			Owners:                   utils.NewStringValue(namespace.Owner),
			Ctime:                    utils.NewStringValue(commontime.Time2String(namespace.CreateTime)),
			Mtime:                    utils.NewStringValue(commontime.Time2String(namespace.ModifyTime)),
			TotalServiceCount:        utils.NewUInt32Value(nsCntInfo.ServiceCount),
			TotalInstanceCount:       utils.NewUInt32Value(nsCntInfo.InstanceCnt.TotalInstanceCount),
			TotalHealthInstanceCount: utils.NewUInt32Value(nsCntInfo.InstanceCnt.HealthyInstanceCount),
			ServiceExportTo:          namespace.ListServiceExportTo(),
		})
		totalServiceCount += nsCntInfo.ServiceCount
		totalInstanceCount += nsCntInfo.InstanceCnt.TotalInstanceCount
		totalHealthInstanceCount += nsCntInfo.InstanceCnt.HealthyInstanceCount
	}
	api.AddNamespaceSummary(out, &apimodel.Summary{
		TotalServiceCount:        totalServiceCount,
		TotalInstanceCount:       totalInstanceCount,
		TotalHealthInstanceCount: totalHealthInstanceCount,
	})
	return out
}

// GetNamespaceToken 获取命名空间的token
func (s *Server) GetNamespaceToken(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	if resp := checkReviseNamespace(ctx, req); resp != nil {
		return resp
	}

	namespace, resp := s.checkNamespaceAuthority(ctx, req)
	if resp != nil {
		return resp
	}

	// s.RecordHistory(namespaceRecordEntry(ctx, req, model.OGetToken))
	// 构造返回数据
	out := &apimodel.Namespace{
		Name:  req.GetName(),
		Token: utils.NewStringValue(namespace.Token),
	}
	return api.NewNamespaceResponse(apimodel.Code_ExecuteSuccess, out)
}

// 根据命名空间查询服务总数
func (s *Server) getServicesCountWithNamespace(namespace string) (uint32, error) {
	filter := map[string]string{"namespace": namespace}
	total, _, err := s.storage.GetServices(filter, nil, nil, 0, 1)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// 根据命名空间查询配置分组总数
func (s *Server) getConfigGroupCountWithNamespace(namespace string) (uint32, error) {
	total, err := s.storage.CountConfigGroups(namespace)
	if err != nil {
		return 0, err
	}
	return uint32(total), nil
}

// loadNamespace
func (s *Server) loadNamespace(name string) (string, error) {
	if val := s.caches.Namespace().GetNamespace(name); val != nil {
		return name, nil
	}
	val, err := s.storage.GetNamespace(name)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return val.Name, nil
}

// 检查namespace的权限，并且返回namespace
func (s *Server) checkNamespaceAuthority(
	ctx context.Context, req *apimodel.Namespace) (*model.Namespace, *apiservice.Response) {
	rid := utils.ParseRequestID(ctx)
	namespaceName := req.GetName().GetValue()
	// namespaceToken := parseNamespaceToken(ctx, req)

	// 检查是否存在
	namespace, err := s.storage.GetNamespace(namespaceName)
	if err != nil {
		log.Error(err.Error(), utils.ZapRequestID(rid))
		return nil, api.NewNamespaceResponse(commonstore.StoreCode2APICode(err), req)
	}
	if namespace == nil {
		return nil, api.NewNamespaceResponse(apimodel.Code_NotFoundResource, req)
	}

	// 鉴权
	// if ok := s.authority.VerifyNamespace(namespace.Token, namespaceToken); !ok {
	// 	return nil, api.NewNamespaceResponse(api.Unauthorized, req)
	// }

	return namespace, nil
}

// 检查批量请求
func checkBatchNamespace(req []*apimodel.Namespace) *apiservice.BatchWriteResponse {
	if len(req) == 0 {
		return api.NewBatchWriteResponse(apimodel.Code_EmptyRequest)
	}

	if len(req) > utils.MaxBatchSize {
		return api.NewBatchWriteResponse(apimodel.Code_BatchSizeOverLimit)
	}

	return nil
}

// 检查创建命名空间请求参数
func checkCreateNamespace(req *apimodel.Namespace) *apiservice.Response {
	if req == nil {
		return api.NewNamespaceResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := utils.CheckResourceName(req.GetName()); err != nil {
		return api.NewNamespaceResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	return nil
}

// 检查删除/修改命名空间请求参数
func checkReviseNamespace(ctx context.Context, req *apimodel.Namespace) *apiservice.Response {
	if req == nil {
		return api.NewNamespaceResponse(apimodel.Code_EmptyRequest, req)
	}

	if err := utils.CheckResourceName(req.GetName()); err != nil {
		return api.NewNamespaceResponse(apimodel.Code_InvalidNamespaceName, req)
	}
	return nil
}

// 检查查询命名空间请求参数
func checkGetNamespace(query map[string][]string) (map[string][]string, int, int, *apiservice.BatchQueryResponse) {
	filter := make(map[string][]string)

	if value := query["name"]; len(value) > 0 {
		filter["name"] = value
	}

	if value := query["owner"]; len(value) > 0 {
		filter["owner"] = value
	}

	offset, err := utils.CheckQueryOffset(query["offset"])
	if err != nil {
		return nil, 0, 0, api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	limit, err := utils.CheckQueryLimit(query["limit"])
	if err != nil {
		return nil, 0, 0, api.NewBatchQueryResponse(apimodel.Code_InvalidParameter)
	}

	return filter, offset, limit, nil
}

// 返回命名空间请求的token
// 默认先从req中获取，不存在，则使用header的
func parseNamespaceToken(ctx context.Context, req *apimodel.Namespace) string {
	if reqToken := req.GetToken().GetValue(); reqToken != "" {
		return reqToken
	}

	if headerToken := utils.ParseToken(ctx); headerToken != "" {
		return headerToken
	}

	return ""
}

// 生成命名空间的记录entry
func namespaceRecordEntry(ctx context.Context, req *apimodel.Namespace, opt model.OperationType) *model.RecordEntry {
	marshaler := jsonpb.Marshaler{}
	datail, _ := marshaler.MarshalToString(req)
	return &model.RecordEntry{
		ResourceType:  model.RNamespace,
		ResourceName:  req.GetName().GetValue(),
		Namespace:     req.GetName().GetValue(),
		OperationType: opt,
		Operator:      utils.ParseOperator(ctx),
		Detail:        datail,
		HappenTime:    time.Now(),
	}
}
