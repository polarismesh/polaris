package discoverLocal

import (
	"fmt"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/plugin"
	"time"
)

/**
 * @brief 注册服务发现统计插件
 */
func init() {
	d := &DiscoverStatisWorker{}
	plugin.RegisterPlugin(d.Name(), d)
}

/**
 * @brief 服务发现统计插件
 */
type DiscoverStatisWorker struct {
	interval time.Duration

	dcc chan *DiscoverCall
	dcs *DiscoverCallStatis
}

/**
 * @brief 获取插件名称
 */
func (d *DiscoverStatisWorker) Name() string {
	return "discoverLocal"
}

/**
 * @brief 初始化服务发现统计插件
 */
func (d *DiscoverStatisWorker) Initialize(conf *plugin.ConfigEntry) error {
	// 设置打印周期
	interval := conf.Option["interval"].(int)
	d.interval = time.Duration(interval) * time.Second

	outputPath := conf.Option["outputPath"].(string)

	// 初始化
	d.dcc = make(chan *DiscoverCall, 1024)
	d.dcs = &DiscoverCallStatis{
		statis: make(map[Service]time.Time),
		logger: newLogger(outputPath + "/" + "discovercall.log"),
	}

	go d.Run()

	return nil
}

/**
 * @brief 销毁服务发现统计插件
 */
func (d *DiscoverStatisWorker) Destroy() error {
	return nil
}

/**
 * @brief 上报请求
 */
func (d *DiscoverStatisWorker) AddDiscoverCall(service, namespace string, time time.Time) error {
	select {
	case d.dcc <- &DiscoverCall{
		service:   service,
		namespace: namespace,
		time:      time,
	}:
	default:
		log.Errorf("[DiscoverStatis] service: %s, namespace: %s is not captured", service, namespace)
		return fmt.Errorf("[DiscoverStatis] service: %s, namespace: %s is not captured", service, namespace)
	}
	return nil
}

/**
 * @brief 主流程
 */
func (d *DiscoverStatisWorker) Run() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.dcs.log()
		case dc := <-d.dcc:
			d.dcs.add(dc)
		}
	}
}
