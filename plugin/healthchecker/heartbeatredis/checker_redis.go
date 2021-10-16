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

package heartbeatredis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/redispool"
	"github.com/polarismesh/polaris-server/plugin"
	"strconv"
	"strings"
	"time"
)

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatRedis"
	// Sep separator to divide id and timestamp
	Sep = ":"
)

// RedisHealthChecker
type RedisHealthChecker struct {
	redisPool *redispool.Pool
	cancel    context.CancelFunc
	respChan  chan *redispool.Resp
}

func (r *RedisHealthChecker) Name() string {
	return PluginName
}

func (r *RedisHealthChecker) processRedisResp(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case resp := <-r.respChan:
			if resp.Err != nil {
				log.Errorf("[Health Check][RedisCheck]id:%s set redis err:%s",
					resp.Value, resp.Err)
			}
		}
	}
}

func (r *RedisHealthChecker) Initialize(c *plugin.ConfigEntry) error {
	redisBytes, err := json.Marshal(c.Option)
	if nil != err {
		return fmt.Errorf("fail to marshal %s config entry, err is %v", PluginName, err)
	}
	config := redispool.DefaultConfig()
	if err = json.Unmarshal(redisBytes, config); nil != err {
		return fmt.Errorf("fail to unmarshal %s config entry, err is %v", PluginName, err)
	}
	r.respChan = make(chan *redispool.Resp)
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	go r.processRedisResp(ctx)
	r.redisPool = redispool.NewPool(ctx, config)
	r.redisPool.Start()
	return nil
}

func (r *RedisHealthChecker) Destroy() error {
	r.cancel()
	return nil
}

// Type type for health check plugin, only one same type plugin is allowed
func (r *RedisHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// Report process heartbeat info report
func (r *RedisHealthChecker) Report(request *plugin.ReportRequest) error {
	value := fmt.Sprintf("%d%s%s", request.CurTimeSec, Sep, request.LocalHost)
	log.Debugf("[Health Check][RedisCheck]redis set key is %s, value is %s", request.InstanceId, value)
	return r.redisPool.Set(request.InstanceId, value, r.respChan)
}

// Query query the heartbeat time
func (r *RedisHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	respCh := make(chan *redispool.Resp)
	err := r.redisPool.Get(request.InstanceId, respCh)
	if nil != err {
		return nil, err
	}
	resp := <-respCh
	if resp.Err != nil {
		log.Errorf("[Health Check][RedisCheck]addr:%s:%d, id:%s, get redis err:%s",
			request.Host, request.Port, request.InstanceId, resp.Err)
		return nil, resp.Err
	}
	value := resp.Value
	queryResp := &plugin.QueryResponse{
		Exists: resp.Exists,
	}
	if len(value) == 0 {
		return queryResp, nil
	}
	tokens := strings.Split(value, Sep)
	if len(tokens) != 2 {
		log.Errorf("[Health Check][RedisCheck]addr:%s:%d, id:%s, invalid redis value:%s",
			request.Host, request.Port, request.InstanceId, value)
		return nil, fmt.Errorf("invalid redis value %s", value)
	}
	lastHeartbeatTimeStr := tokens[0]
	lastHeartbeatTime, err := strconv.ParseInt(lastHeartbeatTimeStr, 10, 64)
	if resp.Err != nil {
		log.Errorf("[Health Check][RedisCheck]addr is %s:%d, id is %s, parse heartbeatTime %s err:%v",
			request.Host, request.Port, request.InstanceId, lastHeartbeatTimeStr, err)
		return nil, resp.Err
	}
	queryResp.Server = tokens[1]
	queryResp.LastHeartbeatSec = lastHeartbeatTime
	return queryResp, nil
}

// Report process the instance check
func (r *RedisHealthChecker) Check(request *plugin.CheckRequest) (*plugin.CheckResponse, error) {
	queryResp, err := r.Query(&request.QueryRequest)
	if nil != err {
		return nil, err
	}
	lastHeartbeatTime := queryResp.LastHeartbeatSec
	checkResp := &plugin.CheckResponse{
		LastHeartbeatTimeSec: lastHeartbeatTime,
	}
	recoverTimeSec := r.redisPool.RecoverTimeSec()
	localCurTimeSec := time.Now().Unix()
	if localCurTimeSec >= recoverTimeSec && localCurTimeSec-recoverTimeSec < int64(request.ExpireDurationSec) {
		checkResp.OnRecover = true
	}
	if request.CurTimeSec > lastHeartbeatTime {
		if request.CurTimeSec-lastHeartbeatTime >= int64(request.ExpireDurationSec) {
			//心跳超时
			checkResp.Healthy = false
			if request.Healthy {
				log.Infof("[Health Check][RedisCheck]health check expired, "+
					"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d instanceId %s",
					lastHeartbeatTime, request.CurTimeSec, request.ExpireDurationSec, request.InstanceId)
			}
			if queryResp.Exists {
				respCh := make(chan *redispool.Resp)
				err = r.redisPool.Del(request.InstanceId, respCh)
				if nil != err {
					return nil, err
				}
				resp := <-respCh
				if resp.Err != nil {
					log.Errorf("[Health Check][RedisCheck]addr is %s:%d, id is %s, delete redis err is %s",
						request.Host, request.Port, request.InstanceId, resp.Err)
					return nil, resp.Err
				}
			}
			return checkResp, nil
		}
	}
	checkResp.Healthy = true
	log.Debugf("[Health Check][RedisCheck]instanceId is %s, healthy is %v", request.InstanceId, checkResp.Healthy)
	return checkResp, nil
}

func init() {
	d := &RedisHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}
