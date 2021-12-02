package local

import "fmt"

// MetricData metric 结构体
type MetricData struct {
	Name       string
	Data       float64
	Labels     map[string]string
	DeleteFlag bool
}

type metricDesc struct {
	Name       string
	Help       string
	MetricType string
	LabelNames []string
}

const (
	// metric name
	MetricForClientRqTotal         string = "client_rq_total"
	MetricForClientRqFailure       string = "client_rq_failure"
	MetricForClientRqTimeout       string = "client_rq_timeout"
	MetricForClientRqIntervalCount string = "client_rq_interval_count"
	MetricForClientRqTimeoutAvg    string = "client_rq_timeout_avg"
	MetricForClientRqTimeoutMax    string = "client_rq_timeout_max"
	MetricForClientRqTimeoutP99    string = "client_rq_timeout_p99"

	// metric label
	LabelForPolarisServerInstance string = "polaris_server_instance"
	LabelForApi                   string = "api"
	LabelForProtocol              string = "protocol"
	LabelForErrCode               string = "err_code"

	// metric type
	TypeForCounterVec   string = "counter_vec"
	TypeForGaugeVec     string = "gauge_vec"
	TypeForHistogramVec string = "histogram_vec"
)

var (
	// metricDescList Metrics Description Defines the list
	metricDescList []metricDesc = []metricDesc{
		{
			Name:       MetricForClientRqTotal,
			Help:       "total number of client requests",
			MetricType: TypeForCounterVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqFailure,
			Help:       "number of client request failures",
			MetricType: TypeForCounterVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeout,
			Help:       "time consumed per interface call",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqIntervalCount,
			Help:       "total number of client request in current interval",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutMax,
			Help:       "max latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutAvg,
			Help:       "average latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
		{
			Name:       MetricForClientRqTimeoutP99,
			Help:       "P99 latency of client requests",
			MetricType: TypeForGaugeVec,
			LabelNames: []string{
				LabelForPolarisServerInstance,
				LabelForApi,
				LabelForProtocol,
				LabelForErrCode,
			},
		},
	}
)

// buildMetricLabels build metric label from APICall or APICallStatisItem
func buildMetricLabels(call *APICall, item *APICallStatisItem) map[string]string {
	if call != nil {
		return map[string]string{
			LabelForErrCode:  fmt.Sprintf("%d", call.code),
			LabelForApi:      call.api,
			LabelForProtocol: call.protocol,
		}
	} else {
		return map[string]string{
			LabelForErrCode:  fmt.Sprintf("%d", item.code),
			LabelForApi:      item.api,
			LabelForProtocol: item.protocol,
		}
	}
}
