package v1

import (
	"context"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/plugin"
)

// 批量获取心跳记录
func (g *DiscoverServer) BatchGetHeartbeat(ctx context.Context,
	req *apiservice.GetHeartbeatsRequest) (*apiservice.GetHeartbeatsResponse, error) {
	checker, ok := g.healthCheckServer.Checkers()[int32(apiservice.HealthCheck_HEARTBEAT)]
	if !ok {
		return &apiservice.GetHeartbeatsResponse{}, nil
	}
	keys := req.GetInstanceIds()
	records := make([]*apiservice.HeartbeatRecord, 0, len(keys))
	for i := range keys {
		resp, err := checker.Query(&plugin.QueryRequest{
			InstanceId: keys[i],
		})
		if err != nil {
			return nil, err
		}
		record := &apiservice.HeartbeatRecord{
			InstanceId:       keys[i],
			LastHeartbeatSec: resp.LastHeartbeatSec,
			Exist:            resp.Exists,
		}
		records = append(records, record)
	}
	return &apiservice.GetHeartbeatsResponse{
		Records: records,
	}, nil
}

// 批量删除心跳记录
func (g *DiscoverServer) BatchDelHeartbeat(ctx context.Context,
	req *apiservice.DelHeartbeatsRequest) (*apiservice.DelHeartbeatsResponse, error) {
	checker, ok := g.healthCheckServer.Checkers()[int32(apiservice.HealthCheck_HEARTBEAT)]
	if !ok {
		return &apiservice.DelHeartbeatsResponse{}, nil
	}
	keys := req.GetInstanceIds()
	for i := range keys {
		_ = checker.Delete(keys[i])
	}
	return &apiservice.DelHeartbeatsResponse{}, nil
}
