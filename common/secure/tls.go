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

package secure

import (
	"crypto/tls"
)

// TLSInfo tls 配置信息
type TLSInfo struct {
	// CertFile 服务端证书文件
	CertFile string
	// KeyFile CertFile 的密钥 key 文件
	KeyFile string
	// ClientCertFile 客户端证书文件 只有当 ClientCertAuth 设置为 true 时生效
	ClientCertFile string
	// ClientKeyFile ClientCertFile 的密钥 key 文件
	ClientKeyFile string

	// TrustedCAFile CA证书文件
	TrustedCAFile string
	// ClientCertAuth 是否启用客户端证书
	ClientCertAuth bool
	// CRLFile 客户端证书吊销列表
	CRLFile string

	// InsecureSkipVerify tls 的一个配置
	// 客户端是否验证证书和服务器主机名
	InsecureSkipVerify bool
	// SkipClientSANVerify tls 的一个配置项
	// 跳过客户端SAN值验证
	SkipClientSANVerify bool
	// ServerName 客户端发送的 Server Name Indication 扩展的值
	// 它在服务器端和客户端都可用
	ServerName string

	// HandshakeFailure 当连接握手失败时可以选择调用 连接将在之后立即关闭
	HandshakeFailure func(*tls.Conn, error)

	// CipherSuites tls 的一个配置
	// 支持的密码套件列表。如果为空，Go 默认会自动填充它。请注意，密码套件按给定顺序排列优先级。
	CipherSuites []uint16

	// AllowedCN 必须由客户提供的 CN
	AllowedCN string

	// AllowedHostname 必须与 TLS 匹配的 IP 地址或主机名
	// 是由客户端提供的证书
	AllowedHostname string
}

// IsEmpty 检查 tls 配置信息是否为空 当证书和密钥同时存在时才不为空
func (t *TLSInfo) IsEmpty() bool {
	if t == nil {
		return true
	}
	if t.CertFile != "" && t.KeyFile != "" {
		return false
	}
	return true
}
