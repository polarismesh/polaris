package local

import (
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sync"
)

// PrometheusStatis
type PrometheusStatis struct {
	registry                     *prometheus.Registry
	metricVecCaches              map[string]interface{}
	polarisPrometheusHttpHandler *PolarisPrometheusHttpHandler
}

// NewPrometheusStatis 初始化 PrometheusStatis
func NewPrometheusStatis() (*PrometheusStatis, error) {
	statis := &PrometheusStatis{}
	statis.metricVecCaches = make(map[string]interface{})
	statis.registry = prometheus.NewRegistry()

	err := statis.registerMetrics()
	if err != nil {
		return nil, err
	}

	handler := &PolarisPrometheusHttpHandler{}
	handler.lock = &sync.RWMutex{}
	handler.promeHttpHandler = promhttp.HandlerFor(statis.GetRegistry(), promhttp.HandlerOpts{})
	statis.polarisPrometheusHttpHandler = handler

	return statis, nil
}

// registerMetrics Registers the interface invocation-related observability metrics
func (statis *PrometheusStatis) registerMetrics() error {
	for _, desc := range metricDescList {
		var collector prometheus.Collector
		switch desc.MetricType {
		case TypeForGaugeVec:
			collector = prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: desc.Name,
				Help: desc.Help,
			}, desc.LabelNames)
		default:
			continue
		}

		err := statis.registry.Register(collector)
		if err != nil {
			log.Errorf("[APICall] register pormetheus collector error, %v", err)
			return err
		}
		statis.metricVecCaches[desc.Name] = collector
	}
	return nil
}

func (a *APICallStatis) collectMetricData(staticsSlice []*APICallStatisItem) {
	if len(staticsSlice) == 0 {
		return
	}

	// 每一个接口，共 metricsNumber 个指标，下面的指标的个数调整时，这里的 metricsNumber 也需要调整
	statInfos := make([]*MetricData, 0, len(staticsSlice)*metricsNumber)

	for _, item := range staticsSlice {

		var maxTime, avgTime, rqTime, rqCount float64
		var deleteFlag bool

		if item.count == 0 && item.zeroDuration > maxZeroDuration {
			deleteFlag = true
		} else {
			maxTime = float64(item.maxTime) / 1e6
			rqTime = float64(item.accTime) / 1e6
			rqCount = float64(item.count)
			if item.count > 0 {
				avgTime = float64(item.accTime) / float64(item.count) / 1e6
			}
			deleteFlag = false
		}

		statInfos = append(statInfos, &MetricData{
			Name:       MetricForClientRqTimeoutMax,
			Data:       maxTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqTimeoutAvg,
			Data:       avgTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqTimeout,
			Data:       rqTime,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		}, &MetricData{
			Name:       MetricForClientRqIntervalCount,
			Data:       rqCount,
			Labels:     buildMetricLabels(nil, item),
			DeleteFlag: deleteFlag,
		},
		)
	}

	// 将指标收集到 prometheus
	a.prometheusStatis.collectMetricData(statInfos)
}

// collectMetricData
func (statis *PrometheusStatis) collectMetricData(statInfos []*MetricData) {

	if len(statInfos) == 0 {
		return
	}

	// 收集到 prometheus 时加写锁，防止 polaris-monitor 读取到不一致的数据
	statis.polarisPrometheusHttpHandler.lock.Lock()
	defer statis.polarisPrometheusHttpHandler.lock.Unlock()

	// 清理掉当前的存量数据

	for _, statInfo := range statInfos {

		if len(statInfo.Labels) == 0 {
			statInfo.Labels = make(map[string]string)
		}

		// Set a public label information: Polaris node information
		statInfo.Labels[LabelForPolarisServerInstance] = utils.LocalHost
		metricVec := statis.metricVecCaches[statInfo.Name]

		if metricVec == nil {
			continue
		}

		switch metric := metricVec.(type) {
		case *prometheus.GaugeVec:
			if statInfo.DeleteFlag {
				metric.Delete(statInfo.Labels)
			} else {
				metric.With(statInfo.Labels).Set(statInfo.Data)
			}
		}
	}

}

// GetRegistry return prometheus.Registry instance
func (statis *PrometheusStatis) GetRegistry() *prometheus.Registry {
	return statis.registry
}

// PolarisPrometheusHttpHandler prometheus 处理 handler
type PolarisPrometheusHttpHandler struct {
	promeHttpHandler http.Handler
	lock             *sync.RWMutex
}

// ServeHTTP 提供 prometheus http 服务
func (p *PolarisPrometheusHttpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	p.promeHttpHandler.ServeHTTP(writer, request)
}

// GetHttpHandler 获取 handler
func (statis *PrometheusStatis) GetHttpHandler() http.Handler {
	return statis.polarisPrometheusHttpHandler
}
