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

package maintain

import (
	"context"
	"errors"
	"runtime/debug"
	"time"

	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	commonlog "github.com/polarismesh/polaris-server/common/log"
)

func (s *Server) GetServerConnections(_ context.Context, req *ConnReq) (*ConnCountResp, error) {
	if req.Protocol == "" {
		return nil, errors.New("missing param protocol")
	}

	lis := connlimit.GetLimitListener(req.Protocol)
	if lis == nil {
		return nil, errors.New("not found the protocol")
	}

	var resp = ConnCountResp{
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

func (s *Server) GetServerConnStats(_ context.Context, req *ConnReq) (*ConnStatsResp, error) {
	if req.Protocol == "" {
		return nil, errors.New("missing param protocol")
	}

	lis := connlimit.GetLimitListener(req.Protocol)
	if lis == nil {
		return nil, errors.New("not found the protocol")
	}

	var resp ConnStatsResp

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

func (s *Server) CloseConnections(_ context.Context, reqs []ConnReq) error {
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

func (s *Server) CleanInstance(ctx context.Context, req *api.Instance) *api.Response {
	return s.namingServer.CleanInstance(ctx, req)
}

func (s *Server) GetLastHeartbeat(_ context.Context, req *api.Instance) *api.Response {
	return s.healthCheckServer.GetLastHeartbeat(req)
}

func (s *Server) GetLogOutputLevel(_ context.Context) (map[string]string, error) {
	scopes := commonlog.Scopes()
	out := make(map[string]string, len(scopes))
	for k := range scopes {
		out[k] = scopes[k].GetOutputLevel().Name()
	}

	return out, nil
}

func (s *Server) SetLogOutputLevel(_ context.Context, scope string, level string) error {
	return commonlog.SetLogOutputLevel(scope, level)
}
