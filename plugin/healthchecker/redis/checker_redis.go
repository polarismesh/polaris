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
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	commonlog "github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/redispool"
	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/plugin"
)

var log = commonlog.GetScopeOrDefaultByName(commonlog.HealthcheckLoggerName)

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatRedis"
	// Sep separator to divide id and timestamp
	Sep = ":"
	// Servers key to manage hb servers
	Servers = "servers"
	// CountSep separator to divide server and count
	CountSep = "|"
)

// RedisHealthChecker 心跳检测redis
type RedisHealthChecker struct {
	// 用于写入心跳数据的池
	hbPool redispool.Pool
	// 用于检查回调的池
	checkPool      redispool.Pool
	cancel         context.CancelFunc
	statis         plugin.Statis
	suspendTimeSec int64
}

// Name plugin name
func (r *RedisHealthChecker) Name() string {
	return PluginName
}

// Initialize initialize plugin
func (r *RedisHealthChecker) Initialize(c *plugin.ConfigEntry) error {
	redisBytes, err := json.Marshal(c.Option)
	if err != nil {
		return fmt.Errorf("fail to marshal %s config entry, err is %v", PluginName, err)
	}
	var config redispool.Config
	if err = json.Unmarshal(redisBytes, &config); err != nil {
		return fmt.Errorf("fail to unmarshal %s config entry, err is %v", PluginName, err)
	}
	r.statis = plugin.GetStatis()
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	r.hbPool = redispool.NewRedisPool(ctx, &config, r.statis)
	r.hbPool.Start()
	r.checkPool = redispool.NewRedisPool(ctx, &config, r.statis)
	r.checkPool.Start()
	if err = r.registerSelf(); err != nil {
		return fmt.Errorf("fail to register %s to redis, err is %v", utils.LocalHost, err)
	}
	return nil
}

func (r *RedisHealthChecker) registerSelf() error {
	localhost := utils.LocalHost
	resp := r.checkPool.Sdd(Servers, []string{localhost})
	return resp.Err
}

// Destroy plugin destroy
func (r *RedisHealthChecker) Destroy() error {
	if nil != r.cancel {
		r.cancel()
	}
	return nil
}

// Type for health check plugin, only one same type plugin is allowed
func (r *RedisHealthChecker) Type() plugin.HealthCheckType {
	return plugin.HealthCheckerHeartbeat
}

// HeathCheckRecord 心跳记录
type HeathCheckRecord struct {
	LocalHost  string
	CurTimeSec int64
	Count      int64
}

// IsEmpty 是否空对象
func (h *HeathCheckRecord) IsEmpty() bool {
	return len(h.LocalHost) == 0 && h.CurTimeSec == 0
}

// Serialize 序列化成字符串
func (h *HeathCheckRecord) Serialize(compatible bool) string {
	if compatible {
		return fmt.Sprintf("1%s%d%s%s%s%d", Sep, h.CurTimeSec, Sep, h.LocalHost, CountSep, h.Count)
	}
	return fmt.Sprintf("%d%s%s%s%d", h.CurTimeSec, Sep, h.LocalHost, CountSep, h.Count)
}

func parseHeartbeatValue(value string, startIdx int) (host string, curTimeSec int64, count int64, err error) {
	tokens := strings.Split(value, Sep)
	if len(tokens) < startIdx+2 {
		return "", 0, 0, fmt.Errorf("invalid redis value %s", value)
	}
	lastHeartbeatTimeStr := tokens[startIdx]
	lastHeartbeatTime, err := strconv.ParseInt(lastHeartbeatTimeStr, 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	host = tokens[startIdx+1]
	curTimeSec = lastHeartbeatTime
	countSepIndex := strings.LastIndex(host, CountSep)
	var countValue int64
	if countSepIndex > 0 && countSepIndex < len(host) {
		countStr := host[countSepIndex+1:]
		countValue, err = strconv.ParseInt(countStr, 10, 64)
		if err != nil {
			return "", 0, 0, err
		}
	}
	return host, curTimeSec, countValue, nil
}

// Deserialize 反序列为对象
func (h *HeathCheckRecord) Deserialize(value string, compatible bool) error {
	if len(value) == 0 {
		return nil
	}
	var err error
	if compatible {
		h.LocalHost, h.CurTimeSec, h.Count, err = parseHeartbeatValue(value, 1)
	} else {
		h.LocalHost, h.CurTimeSec, h.Count, err = parseHeartbeatValue(value, 0)
	}
	return err
}

// String 字符串化
func (h HeathCheckRecord) String() string {
	return fmt.Sprintf("{LocalHost=%s, CurTimeSec=%d}", h.LocalHost, h.CurTimeSec)
}

// Report process heartbeat info report
func (r *RedisHealthChecker) Report(ctx context.Context, request *plugin.ReportRequest) error {
	value := &HeathCheckRecord{
		LocalHost:  request.LocalHost,
		CurTimeSec: request.CurTimeSec,
		Count:      request.Count,
	}

	log.Debugf("[Health Check][RedisCheck]redis set key is %s, value is %s", request.InstanceId, *value)
	resp := r.hbPool.Set(request.InstanceId, value)
	if resp.Err != nil {
		log.Errorf("[Health Check][RedisCheck]addr:%s:%d, id:%s, set redis err:%s",
			request.Host, request.Port, request.InstanceId, resp.Err)
		return resp.Err
	}
	return nil
}

// Query queries the heartbeat time
func (r *RedisHealthChecker) Query(ctx context.Context, request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
	resp := r.checkPool.Get(request.InstanceId)
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
	heathCheckRecord := &HeathCheckRecord{}
	err := heathCheckRecord.Deserialize(value, resp.Compatible)
	if err != nil {
		log.Errorf("[Health Check][RedisCheck]addr is %s:%d, id is %s, parse %s err:%v",
			request.Host, request.Port, request.InstanceId, value, err)
		return nil, err
	}
	queryResp.Server = heathCheckRecord.LocalHost
	queryResp.LastHeartbeatSec = heathCheckRecord.CurTimeSec
	queryResp.Count = heathCheckRecord.Count
	return queryResp, nil
}

func (r *RedisHealthChecker) BatchQuery(ctx context.Context, request *plugin.BatchQueryRequest) (*plugin.BatchQueryResponse, error) {
	keys := make([]string, 0, len(request.Requests))
	for i := range request.Requests {
		keys = append(keys, request.Requests[i].InstanceId)
	}

	resp := r.checkPool.MGet(keys)
	if resp.Err != nil {
		log.Errorf("[Health Check][RedisCheck] mget redis err:%s", resp.Err)
		return nil, resp.Err
	}
	values := resp.Values
	queryResp := &plugin.BatchQueryResponse{
		Responses: make([]*plugin.QueryResponse, 0, len(values)),
	}
	if len(values) == 0 {
		return queryResp, nil
	}
	for i := range values {
		subRsp := &plugin.QueryResponse{}
		value := values[i]
		if value == nil {
			subRsp.Exists = false
		} else {
			heathCheckRecord := &HeathCheckRecord{}
			if err := heathCheckRecord.Deserialize(fmt.Sprintf("%+v", value), resp.Compatible); err != nil {
				log.Errorf("[Health Check][RedisCheck] mget parse %s err:%v", value, err)
				return nil, err
			}
			subRsp.Server = heathCheckRecord.LocalHost
			subRsp.LastHeartbeatSec = heathCheckRecord.CurTimeSec
			subRsp.Count = heathCheckRecord.Count
			subRsp.Exists = true
		}
	}
	return queryResp, nil
}

const maxCheckDuration = 500 * time.Second

func (r *RedisHealthChecker) skipCheck(instanceId string, expireDurationSec int64) bool {
	suspendTimeSec := r.SuspendTimeSec()
	localCurTimeSec := commontime.CurrentMillisecond() / 1000
	if suspendTimeSec > 0 && localCurTimeSec >= suspendTimeSec && localCurTimeSec-suspendTimeSec < expireDurationSec {
		log.Infof("[Health Check][RedisCheck]health check redis suspended, "+
			"suspendTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, instanceId)
		return true
	}
	recoverTimeSec := r.checkPool.RecoverTimeSec()
	// redis恢复期，不做变更
	if recoverTimeSec > 0 && localCurTimeSec >= recoverTimeSec && localCurTimeSec-recoverTimeSec < expireDurationSec {
		log.Infof("[Health Check][RedisCheck]health check redis on recover, "+
			"recoverTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			suspendTimeSec, localCurTimeSec, expireDurationSec, instanceId)
		return true
	}
	return false
}

// Check Report process the instance check
func (r *RedisHealthChecker) Check(request *plugin.CheckRequest) (*plugin.CheckResponse, error) {
	var startTime = time.Now()
	defer func() {
		var timePass = time.Since(startTime)
		if timePass >= maxCheckDuration {
			log.Warnf("[Health Check][RedisCheck]check %s cost %s duration, greater than max %s duration",
				request.InstanceId, timePass, maxCheckDuration)
		}
	}()
	queryResp, err := r.Query(context.Background(), &request.QueryRequest)
	if err != nil {
		return nil, err
	}
	lastHeartbeatTime := queryResp.LastHeartbeatSec
	checkResp := &plugin.CheckResponse{
		LastHeartbeatTimeSec: lastHeartbeatTime,
	}
	curTimeSec := request.CurTimeSec()
	if r.skipCheck(request.InstanceId, int64(request.ExpireDurationSec)) {
		checkResp.StayUnchanged = true
		return checkResp, nil
	}
	// 出现时间倒退，不对心跳状态做变更
	if curTimeSec < lastHeartbeatTime {
		log.Infof("[Health Check][RedisCheck]time reverse, curTime is %d, last heartbeat time is %d, id %s",
			curTimeSec, lastHeartbeatTime, request.InstanceId)
		checkResp.StayUnchanged = true
		return checkResp, nil
	}
	// 正常进行心跳中
	checkResp.Regular = true
	if curTimeSec-lastHeartbeatTime >= int64(request.ExpireDurationSec) {
		// 心跳超时
		checkResp.Healthy = false
		if request.Healthy {
			log.Infof("[Health Check][RedisCheck]health check expired, "+
				"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d instanceId %s",
				lastHeartbeatTime, curTimeSec, request.ExpireDurationSec, request.InstanceId)
		} else {
			checkResp.StayUnchanged = true
		}
	} else {
		// 心跳恢复
		checkResp.Healthy = true
		if !request.Healthy {
			log.Infof("[Health Check][RedisCheck]health check resumed, "+
				"last hb timestamp is %d, curTimeSec is %d, expireDurationSec is %d instanceId %s",
				lastHeartbeatTime, curTimeSec, request.ExpireDurationSec, request.InstanceId)
		} else {
			checkResp.StayUnchanged = true
		}
	}
	log.Debugf("[Health Check][RedisCheck]instanceId is %s, healthy is %v", request.InstanceId, checkResp.Healthy)
	return checkResp, nil
}

// Delete delete the target id
func (r *RedisHealthChecker) Delete(ctx context.Context, id string) error {
	resp := r.checkPool.Del(id)
	return resp.Err
}

// Suspend checker for an entire expired interval
func (r *RedisHealthChecker) Suspend() {
	curTimeMilli := commontime.CurrentMillisecond() / 1000
	log.Infof("[Health Check][RedisCheck] suspend checker, start time %d", curTimeMilli)
	atomic.StoreInt64(&r.suspendTimeSec, curTimeMilli)
}

// SuspendTimeSec get suspend time in seconds
func (r *RedisHealthChecker) SuspendTimeSec() int64 {
	return atomic.LoadInt64(&r.suspendTimeSec)
}

func (r *RedisHealthChecker) DebugHandlers() []model.DebugHandler {
	return []model.DebugHandler{}
}

func init() {
	d := &RedisHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}
