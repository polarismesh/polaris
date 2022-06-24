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
	"time"

	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/redispool"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
)

var log = commonlog.NamingScope()

// 把操作记录记录到日志文件中
const (
	// PluginName plugin name
	PluginName = "heartbeatRedis"
	// Sep separator to divide id and timestamp
	Sep = ":"
	// Servers key to manage hb servers
	Servers = "servers"
)

// RedisHealthChecker 心跳检测redis
type RedisHealthChecker struct {
	// 用于写入心跳数据的池
	hbPool *redispool.Pool
	// 用于检查回调的池
	checkPool *redispool.Pool
	cancel    context.CancelFunc
	statis    plugin.Statis
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
	r.hbPool = redispool.NewPool(ctx, &config, r.statis)
	r.hbPool.Start()
	r.checkPool = redispool.NewPool(ctx, &config, r.statis)
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
	r.cancel()
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
}

// IsEmpty 是否空对象
func (h *HeathCheckRecord) IsEmpty() bool {
	return len(h.LocalHost) == 0 && h.CurTimeSec == 0
}

// Serialize 序列化成字符串
func (h *HeathCheckRecord) Serialize(compatible bool) string {
	if compatible {
		return fmt.Sprintf("1%s%d%s%s", Sep, h.CurTimeSec, Sep, h.LocalHost)
	}
	return fmt.Sprintf("%d%s%s", h.CurTimeSec, Sep, h.LocalHost)
}

func parseHeartbeatValue(value string, startIdx int) (host string, curTimeSec int64, err error) {
	tokens := strings.Split(value, Sep)
	if len(tokens) != startIdx+2 {
		return "", 0, fmt.Errorf("invalid redis value %s", value)
	}
	lastHeartbeatTimeStr := tokens[startIdx]
	lastHeartbeatTime, err := strconv.ParseInt(lastHeartbeatTimeStr, 10, 64)
	if err != nil {
		return "", 0, err
	}
	host = tokens[startIdx+1]
	curTimeSec = lastHeartbeatTime
	return host, curTimeSec, nil
}

// Deserialize 反序列为对象
func (h *HeathCheckRecord) Deserialize(value string, compatible bool) error {
	if len(value) == 0 {
		return nil
	}
	var err error
	if compatible {
		h.LocalHost, h.CurTimeSec, err = parseHeartbeatValue(value, 1)
	} else {
		h.LocalHost, h.CurTimeSec, err = parseHeartbeatValue(value, 0)
	}
	return err
}

// String 字符串化
func (h HeathCheckRecord) String() string {
	return fmt.Sprintf("{LocalHost=%s, CurTimeSec=%d}", h.LocalHost, h.CurTimeSec)
}

// Report process heartbeat info report
func (r *RedisHealthChecker) Report(request *plugin.ReportRequest) error {
	value := &HeathCheckRecord{
		LocalHost:  request.LocalHost,
		CurTimeSec: request.CurTimeSec,
	}

	log.Debugf("[Health Check][RedisCheck]redis set key is %s, value is %s", request.InstanceId, *value)
	resp := r.hbPool.Set(request.InstanceId, value)
	if resp.Err != nil {
		log.Errorf("[Health Check][RedisCheck]addr:%s:%d, id:%s, set redis err:%s",
			request.Host, request.Port, request.InstanceId, resp.Err)
	}
	return nil
}

// Query queries the heartbeat time
func (r *RedisHealthChecker) Query(request *plugin.QueryRequest) (*plugin.QueryResponse, error) {
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
	return queryResp, nil
}

const maxCheckDuration = 500 * time.Second

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
	queryResp, err := r.Query(&request.QueryRequest)
	if err != nil {
		return nil, err
	}
	lastHeartbeatTime := queryResp.LastHeartbeatSec
	checkResp := &plugin.CheckResponse{
		LastHeartbeatTimeSec: lastHeartbeatTime,
	}
	recoverTimeSec := r.checkPool.RecoverTimeSec()
	localCurTimeSec := time.Now().Unix()
	// redis恢复期，不做变更
	if localCurTimeSec >= recoverTimeSec && localCurTimeSec-recoverTimeSec < int64(request.ExpireDurationSec) {
		log.Infof("[Health Check][RedisCheck]health check redis on recover, "+
			"recoverTimeSec is %d, localCurTimeSec is %d, expireDurationSec is %d, id %s",
			recoverTimeSec, localCurTimeSec, request.ExpireDurationSec, request.InstanceId)
		checkResp.StayUnchanged = true
		return checkResp, nil
	}
	curTimeSec := request.CurTimeSec()
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
		if queryResp.Exists {
			err := r.Delete(request.InstanceId)
			if err != nil {
				log.Errorf("[Health Check][RedisCheck]addr is %s:%d, id is %s, delete redis err is %s",
					request.Host, request.Port, request.InstanceId, err)
				return nil, err
			}
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

// AddToCheck add the instances to check procedure
func (r *RedisHealthChecker) AddToCheck(request *plugin.AddCheckRequest) error {
	if len(request.Instances) == 0 {
		return nil
	}
	resp := r.checkPool.Sdd(request.LocalHost, request.Instances)
	return resp.Err
}

// RemoveFromCheck AddToCheck add the instances to check procedure
func (r *RedisHealthChecker) RemoveFromCheck(request *plugin.AddCheckRequest) error {
	if len(request.Instances) == 0 {
		return nil
	}
	resp := r.checkPool.Srem(request.LocalHost, request.Instances)
	return resp.Err
}

// Delete delete the target id
func (r *RedisHealthChecker) Delete(id string) error {
	resp := r.checkPool.Del(id)
	return resp.Err
}

func init() {
	d := &RedisHealthChecker{}
	plugin.RegisterPlugin(d.Name(), d)
}
