package discoverLocal

import (
	"bytes"
	"go.uber.org/zap"
	"time"
)

/**
 * @brief 服务发现统计
 */
type DiscoverCall struct {
	service   string
	namespace string
	time      time.Time
}

/**
 * @brief 服务
 */
type Service struct {
	name      string
	namespace string
}

/**
 * @brief 服务发现统计条目
 */
type DiscoverCallStatis struct {
	statis map[Service]time.Time

	logger *zap.Logger
}

/**
 * @brief 添加服务发现统计数据
 */
func (d *DiscoverCallStatis) add(dc *DiscoverCall) {
	service := Service{
		name:      dc.service,
		namespace: dc.namespace,
	}

	d.statis[service] = dc.time
}

/**
 * @brief 打印服务发现统计
 */
func (d *DiscoverCallStatis) log() {
	if len(d.statis) == 0 {
		return
	}

	var buffer bytes.Buffer
	for service, time := range d.statis {
		buffer.WriteString("service=")
		buffer.WriteString(service.name)
		buffer.WriteString(";")
		buffer.WriteString("namespace=")
		buffer.WriteString(service.namespace)
		buffer.WriteString(";")
		buffer.WriteString("visitTime=")
		buffer.WriteString(time.Format("2006-01-02 15:04:05"))
		buffer.WriteString("\n")
	}

	d.logger.Info(buffer.String())

	d.statis = make(map[Service]time.Time)
}
