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

package httpserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/connlimit"
	commonlog "github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
)

// GetMaintainAccessServer 运维接口
func (h *HTTPServer) GetMaintainAccessServer() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/maintain/v1").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/apiserver/conn").To(h.GetServerConnections))
	ws.Route(ws.GET("/apiserver/conn/stats").To(h.GetServerConnStats))
	ws.Route(ws.POST("apiserver/conn/close").To(h.CloseConnections))
	ws.Route(ws.POST("/memory/free").To(h.FreeOSMemory))
	ws.Route(ws.POST("/instance/clean").Consumes(restful.MIME_JSON).To(h.CleanInstance))
	ws.Route(ws.GET("/instance/heartbeat").To(h.GetLastHeartbeat))
	ws.Route(ws.GET("/log/outputlevel").To(h.GetLogOutputLevel))
	ws.Route(ws.PUT("/log/outputlevel").To(h.SetLogOutputLevel))
	return ws
}

// GetServerConnections 查看server的连接数
// query参数：protocol，必须，查看指定协议server
//           host，可选，查看指定host
func (h *HTTPServer) GetServerConnections(req *restful.Request, rsp *restful.Response) {
	params := parseQueryParams(req)
	protocol := params["protocol"]
	host := params["host"]
	if protocol == "" {
		_ = rsp.WriteErrorString(http.StatusBadRequest, "missing param protocol")
		return
	}

	lis := connlimit.GetLimitListener(protocol)
	if lis == nil {
		_ = rsp.WriteErrorString(http.StatusBadRequest, "not found the protocol")
		return
	}

	var out struct {
		Protocol string
		Total    int32
		Host     map[string]int32
	}

	out.Protocol = protocol
	out.Total = lis.GetListenerConnCount()
	out.Host = make(map[string]int32)
	if host != "" {
		out.Host[host] = lis.GetHostConnCount(host)
	} else {
		lis.Range(func(host string, count int32) bool {
			out.Host[host] = count
			return true
		})
	}

	_ = rsp.WriteEntity(out)
}

// GetServerConnStats 获取连接缓存里面的统计信息
func (h *HTTPServer) GetServerConnStats(req *restful.Request, rsp *restful.Response) {
	params := parseQueryParams(req)
	protocol := params["protocol"]
	host := params["host"]
	if protocol == "" {
		_ = rsp.WriteErrorString(http.StatusBadRequest, "missing param protocol")
		return
	}

	lis := connlimit.GetLimitListener(protocol)
	if lis == nil {
		_ = rsp.WriteErrorString(http.StatusBadRequest, "not found the protocol")
		return
	}
	var out struct {
		Protocol        string
		ActiveConnTotal int32
		StatsTotal      int
		StatsSize       int
		Stats           []*connlimit.HostConnStat
	}
	out.Protocol = protocol
	out.ActiveConnTotal = lis.GetListenerConnCount()

	stats := lis.GetHostConnStats(host)
	out.Stats = stats
	out.StatsTotal = len(stats)

	// 过滤amount
	if amountStr, ok := params["amount"]; ok {
		out.Stats = make([]*connlimit.HostConnStat, 0)
		amount, _ := strconv.Atoi(amountStr)
		for _, stat := range stats {
			if stat.Amount >= int32(amount) {
				out.Stats = append(out.Stats, stat)
			}
		}
	}
	out.StatsSize = len(out.Stats)

	if out.Stats == nil {
		out.Stats = make([]*connlimit.HostConnStat, 0)
	}
	_ = rsp.WriteAsJson(out)
}

// CloseConnections 关闭指定client ip的连接
func (h *HTTPServer) CloseConnections(req *restful.Request, rsp *restful.Response) {
	log.Info("[HTTP] Start doing close connections")
	var body []struct {
		Protocol string
		Host     string
		Port     int // 可以为0，为0意味着关闭host所有的连接
	}
	decoder := json.NewDecoder(req.Request.Body)
	if err := decoder.Decode(&body); err != nil {
		log.Errorf("[HTTP] close connection decode body err: %s", err.Error())
		_ = rsp.WriteError(http.StatusBadRequest, err)
		return
	}
	for _, entry := range body {
		if entry.Protocol == "" {
			log.Errorf("[HTTP] close connection missing protocol")
			_ = rsp.WriteErrorString(http.StatusBadRequest, "missing protocol")
			return
		}
		if entry.Host == "" {
			log.Errorf("[HTTP] close connection missing host")
			_ = rsp.WriteErrorString(http.StatusBadRequest, "missing host")
			return
		}
	}

	for _, entry := range body {
		listener := connlimit.GetLimitListener(entry.Protocol)
		if listener == nil {
			log.Warnf("[HTTP] not found listener for protocol(%s)", entry.Protocol)
			continue
		}
		if entry.Port != 0 {
			if conn := listener.GetHostConnection(entry.Host, entry.Port); conn != nil {
				log.Infof("[HTTP] address(%s:%d) to be closed", entry.Host, entry.Port)
				_ = conn.Close()
				continue
			}
		}

		log.Infof("[HTTP] host(%s) connections to be closed", entry.Host)
		activeConns := listener.GetHostActiveConns(entry.Host)
		for _, conn := range activeConns {
			if conn != nil {
				_ = conn.Close()
			}
		}
	}
}

// FreeOSMemory 增加一个释放系统内存的接口
func (h *HTTPServer) FreeOSMemory(_ *restful.Request, _ *restful.Response) {
	log.Info("[HTTP] start doing free os memory")
	// 防止并发释放
	start := time.Now()
	h.freeMemMu.Lock()
	debug.FreeOSMemory()
	h.freeMemMu.Unlock()
	log.Infof("[HTTP] finish doing free os memory, used time: %v", time.Since(start))
}

// CleanInstance 彻底清理flag=1的实例运维接口
// 支持一个个清理
func (h *HTTPServer) CleanInstance(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}

	instance := &api.Instance{}
	ctx, err := handler.Parse(instance)
	if err != nil {
		handler.WriteHeaderAndProto(api.NewResponseWithMsg(api.ParseException, err.Error()))
		return
	}

	handler.WriteHeaderAndProto(h.namingServer.CleanInstance(ctx, instance))
}

// GetLastHeartbeat 获取实例，上一次心跳的时间
func (h *HTTPServer) GetLastHeartbeat(req *restful.Request, rsp *restful.Response) {
	handler := &Handler{req, rsp}
	params := parseQueryParams(req)
	instance := &api.Instance{}
	if id, ok := params["id"]; ok && id != "" {
		instance.Id = utils.NewStringValue(id)
		ret := h.healthCheckServer.GetLastHeartbeat(instance)
		handler.WriteHeaderAndProto(ret)
		return
	}

	instance.Service = utils.NewStringValue(params["service"])
	instance.Namespace = utils.NewStringValue(params["namespace"])
	instance.VpcId = utils.NewStringValue(params["vpc_id"])
	instance.Host = utils.NewStringValue(params["host"])
	port, _ := strconv.Atoi(params["port"])
	instance.Port = utils.NewUInt32Value(uint32(port))

	ret := h.healthCheckServer.GetLastHeartbeat(instance)
	handler.WriteHeaderAndProto(ret)
}

// GetLogOutputLevel 获取日志输出级别
func (h *HTTPServer) GetLogOutputLevel(req *restful.Request, rsp *restful.Response) {
	scopes := commonlog.Scopes()
	out := make(map[string]string, len(scopes))
	for k, v := range scopes {
		out[k] = v.GetOutputLevel().Name()
	}

	_ = rsp.WriteAsJson(out)
}

// SetLogOutputLevel 设置日志输出级别
func (h *HTTPServer) SetLogOutputLevel(req *restful.Request, rsp *restful.Response) {
	var scopeLogLevel struct {
		Scope string `json:"scope"`
		Level string `json:"level"`
	}
	body, err := ioutil.ReadAll(req.Request.Body)
	if err != nil {
		_ = rsp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}
	if err := json.Unmarshal(body, &scopeLogLevel); err != nil {
		_ = rsp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}

	if err := commonlog.SetLogOutputLevel(scopeLogLevel.Scope, scopeLogLevel.Level); err != nil {
		_ = rsp.WriteErrorString(http.StatusBadRequest, err.Error())
		return
	}

	_ = rsp.WriteEntity("ok")
}
