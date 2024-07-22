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

package admin

import (
	"context"
	"errors"
	"runtime/debug"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	api "github.com/polarismesh/polaris/common/api/v1"
	connlimit "github.com/polarismesh/polaris/common/conn/limit"
	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/model/admin"
	commonstore "github.com/polarismesh/polaris/common/store"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

func (s *Server) HasMainUser(ctx context.Context, user apisecurity.User) (bool, error) {
	return false, nil
}

func (s *Server) InitMainUser(ctx context.Context, user apisecurity.User) error {
	return nil
}

func (s *Server) GetServerConnections(_ context.Context, req *admin.ConnReq) (*admin.ConnCountResp, error) {
	if req.Protocol == "" {
		return nil, errors.New("missing param protocol")
	}

	lis := connlimit.GetLimitListener(req.Protocol)
	if lis == nil {
		return nil, errors.New("not found the protocol")
	}

	var resp = admin.ConnCountResp{
		Protocol: req.Protocol,
		Total:    lis.GetListenerConnCount(),
		Host:     map[string]int32{},
	}
	if req.Host != "" {
		resp.Host[req.Host] = lis.GetHostConnCount(req.Host)
	} else {
		lis.Range(func(host string, count int32) bool {
			resp.Host[host] = count
			return true
		})
	}

	return &resp, nil
}

func (s *Server) GetServerConnStats(_ context.Context, req *admin.ConnReq) (*admin.ConnStatsResp, error) {
	if req.Protocol == "" {
		return nil, errors.New("missing param protocol")
	}

	lis := connlimit.GetLimitListener(req.Protocol)
	if lis == nil {
		return nil, errors.New("not found the protocol")
	}

	var resp admin.ConnStatsResp

	resp.Protocol = req.Protocol
	resp.ActiveConnTotal = lis.GetListenerConnCount()

	stats := lis.GetHostConnStats(req.Host)
	resp.StatsTotal = len(stats)

	// 过滤amount
	if req.Amount > 0 {
		for _, stat := range stats {
			if stat.Amount >= int32(req.Amount) {
				resp.Stats = append(resp.Stats, stat)
			}
		}
	} else {
		resp.Stats = stats
	}

	resp.StatsSize = len(resp.Stats)
	if len(resp.Stats) == 0 {
		resp.Stats = make([]*connlimit.HostConnStat, 0)
	}

	return &resp, nil
}

func (s *Server) CloseConnections(_ context.Context, reqs []admin.ConnReq) error {
	for _, entry := range reqs {
		listener := connlimit.GetLimitListener(entry.Protocol)
		if listener == nil {
			log.Warnf("[MAINTAIN] not found listener for protocol(%s)", entry.Protocol)
			continue
		}

		if entry.Port != 0 {
			if conn := listener.GetHostConnection(entry.Host, entry.Port); conn != nil {
				log.Infof("[MAINTAIN] address(%s:%d) to be closed", entry.Host, entry.Port)
				_ = conn.Close()
				continue
			}
		}

		log.Infof("[MAINTAIN] host(%s) connections to be closed", entry.Host)
		activeConns := listener.GetHostActiveConns(entry.Host)
		for k := range activeConns {
			if activeConns[k] != nil {
				_ = activeConns[k].Close()
			}
		}
	}

	return nil
}

func (s *Server) FreeOSMemory(_ context.Context) error {
	log.Info("[MAINTAIN] start doing free os memory")
	// 防止并发释放
	start := time.Now()
	s.mu.Lock()
	debug.FreeOSMemory()
	s.mu.Unlock()
	log.Infof("[MAINTAIN] finish doing free os memory, used time: %v", time.Since(start))
	return nil
}

func (s *Server) CleanInstance(ctx context.Context, req *apiservice.Instance) *apiservice.Response {
	getInstanceID := func() (string, *apiservice.Response) {
		if req.GetId() != nil {
			if req.GetId().GetValue() == "" {
				return "", api.NewInstanceResponse(apimodel.Code_InvalidInstanceID, req)
			}
			return req.GetId().GetValue(), nil
		}
		return utils.CheckInstanceTetrad(req)
	}

	instanceID, resp := getInstanceID()
	if resp != nil {
		return resp
	}
	if err := s.storage.CleanInstance(instanceID); err != nil {
		log.Error("Clean instance",
			zap.String("err", err.Error()), utils.ZapRequestID(utils.ParseRequestID(ctx)))
		return api.NewInstanceResponse(commonstore.StoreCode2APICode(err), req)
	}

	log.Info("Clean instance", utils.ZapRequestID(utils.ParseRequestID(ctx)), utils.ZapInstanceID(instanceID))
	return api.NewInstanceResponse(apimodel.Code_ExecuteSuccess, req)
}

func (s *Server) BatchCleanInstances(ctx context.Context, batchSize uint32) (uint32, error) {
	return s.storage.BatchCleanDeletedInstances(10*time.Minute, batchSize)
}

func (s *Server) GetLastHeartbeat(_ context.Context, req *apiservice.Instance) *apiservice.Response {
	return s.healthCheckServer.GetLastHeartbeat(req)
}

func (s *Server) GetLogOutputLevel(_ context.Context) ([]admin.ScopeLevel, error) {
	scopes := commonlog.Scopes()
	out := make([]admin.ScopeLevel, 0, len(scopes))
	for k := range scopes {
		out = append(out, admin.ScopeLevel{
			Name:  k,
			Level: scopes[k].GetOutputLevel().Name(),
		})
	}

	return out, nil
}

func (s *Server) SetLogOutputLevel(_ context.Context, scope string, level string) error {
	return commonlog.SetLogOutputLevel(scope, level)
}

func (s *Server) ListLeaderElections(_ context.Context) ([]*admin.LeaderElection, error) {
	return s.storage.ListLeaderElections()
}

func (s *Server) ReleaseLeaderElection(_ context.Context, electKey string) error {
	return s.storage.ReleaseLeaderElection(electKey)

}

func (svr *Server) GetCMDBInfo(ctx context.Context) ([]model.LocationView, error) {
	cmdb := plugin.GetCMDB()
	if cmdb == nil {
		return []model.LocationView{}, nil
	}

	ret := make([]model.LocationView, 0, 32)
	_ = cmdb.Range(func(host string, location *model.Location) (bool, error) {
		ret = append(ret, model.LocationView{
			IP:       host,
			Region:   location.Proto.GetRegion().GetValue(),
			Zone:     location.Proto.GetZone().GetValue(),
			Campus:   location.Proto.GetCampus().GetValue(),
			RegionID: location.RegionID,
			ZoneID:   location.ZoneID,
			CampusID: location.CampusID,
		})
		return true, nil
	})

	return ret, nil
}
