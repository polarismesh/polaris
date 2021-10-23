package xdsserverv3

import (
	"github.com/polarismesh/polaris-server/apiserver"
)

/**
 * @brief 自注册到API服务器插槽
 */
func init() {
	_ = apiserver.Register("xdsserverv3", &XDSServer{})
}
