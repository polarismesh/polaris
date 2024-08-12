package policy

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	cachetypes "github.com/polarismesh/polaris/cache/api"
	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/model"
	authcommon "github.com/polarismesh/polaris/common/model/auth"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
)

// CreateRoles 批量创建角色
func (svr *Server) CreateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := svr.CreateRole(ctx, reqs[i])
		api.Collect(responses, rsp)
	}
	return api.FormatBatchWriteResponse(responses)
}

// CreateRole 创建角色
func (svr *Server) CreateRole(ctx context.Context, req *apisecurity.Role) *apiservice.Response {
	req.Owner = utils.ParseOwnerID(ctx)

	saveData := &authcommon.Role{}
	saveData.FromSpec(req)

	if err := svr.storage.AddRole(saveData); err != nil {
		log.Error("[Auth][Role] create role into store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	return api.NewResponse(apimodel.Code_ExecuteSuccess)
}

// UpdateRoles 批量更新角色
func (svr *Server) UpdateRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := svr.UpdateRole(ctx, reqs[i])
		api.Collect(responses, rsp)
	}
	return api.FormatBatchWriteResponse(responses)
}

// UpdateRole 批量更新角色
func (svr *Server) UpdateRole(ctx context.Context, req *apisecurity.Role) *apiservice.Response {
	newData := &authcommon.Role{}
	newData.FromSpec(req)

	saveData, err := svr.storage.GetRole(newData.ID)
	if err != nil {
		log.Error("[Auth][Role] get one role from store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData == nil {
		log.Error("[Auth][Role] not find expect role", utils.RequestID(ctx),
			zap.String("id", newData.ID))
		return api.NewAuthResponse(apimodel.Code_NotFoundResource)
	}

	newData.Name = saveData.Name
	newData.Owner = saveData.Owner

	if err := svr.storage.AddRole(newData); err != nil {
		log.Error("[Auth][Role] update role into store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	return api.NewResponse(apimodel.Code_ExecuteSuccess)
}

// DeleteRoles 批量删除角色
func (svr *Server) DeleteRoles(ctx context.Context, reqs []*apisecurity.Role) *apiservice.BatchWriteResponse {
	responses := api.NewBatchWriteResponse(apimodel.Code_ExecuteSuccess)
	for i := range reqs {
		rsp := svr.DeleteRole(ctx, reqs[i])
		api.Collect(responses, rsp)
	}
	return api.FormatBatchWriteResponse(responses)
}

// DeleteRole 批量删除角色
func (svr *Server) DeleteRole(ctx context.Context, req *apisecurity.Role) *apiservice.Response {
	newData := &authcommon.Role{}
	newData.FromSpec(req)

	saveData, err := svr.storage.GetRole(newData.ID)
	if err != nil {
		log.Error("[Auth][Role] get one role from store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if saveData == nil {
		return api.NewAuthResponse(apimodel.Code_ExecuteSuccess)
	}

	tx, err := svr.storage.StartTx()
	if err != nil {
		log.Error("[Auth][Role] start tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := svr.storage.DeleteRole(tx, newData); err != nil {
		log.Error("[Auth][Role] update role into store", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}
	if err := svr.storage.CleanPrincipalPolicies(tx, authcommon.Principal{
		PrincipalID:   saveData.ID,
		PrincipalType: authcommon.PrincipalRole,
	}); err != nil {
		log.Error("[Auth][Role] clean role link policies", utils.RequestID(ctx),
			zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	if err := tx.Commit(); err != nil {
		log.Error("[Auth][Role] delete role commit tx", utils.RequestID(ctx), zap.Error(err))
		return api.NewAuthResponse(commonstore.StoreCode2APICode(err))
	}

	return api.NewResponse(apimodel.Code_ExecuteSuccess)
}

// GetRoles 查询角色列表
func (svr *Server) GetRoles(ctx context.Context, filters map[string]string) *apiservice.BatchQueryResponse {
	offset, limit, _ := utils.ParseOffsetAndLimit(filters)

	total, ret, err := svr.cacheMgr.Role().Query(ctx, cachetypes.RoleSearchArgs{
		Filters: filters,
		Offset:  offset,
		Limit:   limit,
	})
	if err != nil {
		log.Error("[Auth][Role] query roles list", utils.RequestID(ctx), zap.Error(err))
		return api.NewBatchQueryResponse(commonstore.StoreCode2APICode(err))
	}

	rsp := api.NewBatchQueryResponse(apimodel.Code_ExecuteSuccess)
	rsp.Amount = utils.NewUInt32Value(total)
	rsp.Size = utils.NewUInt32Value(uint32(len(ret)))

	for i := range ret {
		if err := api.AddAnyDataIntoBatchQuery(rsp, ret[i].ToSpec()); err != nil {
			log.Error("[Auth][Role] add role to query list", utils.RequestID(ctx), zap.Error(err))
			return api.NewBatchQueryResponse(apimodel.Code_ExecuteException)
		}
	}
	return rsp
}

func recordRoleEntry(ctx context.Context, req *apisecurity.Role, data *authcommon.Role, op model.OperationType) *model.RecordEntry {
	marshaler := jsonpb.Marshaler{}
	detail, _ := marshaler.MarshalToString(req)

	entry := &model.RecordEntry{
		ResourceType:  model.RAuthRole,
		ResourceName:  fmt.Sprintf("%s(%s)", data.Name, data.ID),
		OperationType: op,
		Operator:      utils.ParseOperator(ctx),
		Detail:        detail,
		HappenTime:    time.Now(),
	}

	return entry
}
