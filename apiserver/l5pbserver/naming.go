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

package l5pbserver

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/api/l5"
	"github.com/polarismesh/polaris/common/model"
)

type l5Code uint32

const (
	l5Success l5Code = iota
	l5ResponseFailed
	l5UnmarshalPacketFailed
	l5SyncByAgentCmdFailed
	l5RegisterByNameCmdFailed
	l5MarshalPacketFailed
	l5PacketCmdFailed
)

// handleConnection 连接处理的协程函数
// 每个客户端会新建一个协程
// 使用方法 go handleConnection(conn)
func (l *L5pbserver) handleConnection(conn net.Conn) {
	defer conn.Close()

	req := &cl5Request{
		conn:       conn,
		clientAddr: conn.RemoteAddr().String(),
	}

	// 先读取头部数据，然后再根据packetLength读取body数据
	header := make([]byte, headSize)
	bufRead := bufio.NewReader(conn)
	for {
		if _, err := io.ReadFull(bufRead, header); err != nil {
			// end of the reader
			if err == io.EOF {
				return
			}
			log.Errorf("[Cl5] read header from conn(%s) err: %s", req.clientAddr, err.Error())
			return
		}
		packetLength := checkRequest(header)
		if packetLength <= headSize || packetLength > maxSize { // 包大小检查
			log.Errorf("[Cl5] read header from conn(%s) found body length(%d) invalid",
				req.clientAddr, packetLength)
			return
		}
		body := make([]byte, packetLength-headSize)
		if _, err := io.ReadFull(bufRead, body); err != nil {
			log.Errorf("[Cl5] read body from conn(%s) err: %s", req.clientAddr, err.Error())
			return
		}
		if code := l.handleRequest(req, body); code != l5Success {
			log.Error("[CL5] catch error code", zap.Uint32("code", uint32(code)),
				zap.String("client", req.clientAddr))
			return
		}
	}
}

// handleRequest cl5请求的handle入口
func (l *L5pbserver) handleRequest(req *cl5Request, requestData []byte) l5Code {
	req.start = time.Now() // 从解包开始计算开始时间
	cl5Pkg := &l5.Cl5Pkg{}
	err := proto.Unmarshal(requestData, cl5Pkg)
	if err != nil {
		log.Errorf("[Cl5] client(%s) Unmarshal requestData error: %v",
			req.clientAddr, err)
		return l5UnmarshalPacketFailed
	}

	req.cmd = cl5Pkg.GetCmd()
	l.PreProcess(req)
	switch cl5Pkg.GetCmd() {
	case int32(l5.CL5_CMD_CL5_REGISTER_BY_NAME_CMD):
		req.code = l.handleRegisterByNameCmd(req.conn, cl5Pkg)
	case int32(l5.CL5_CMD_CL5_SYNC_BY_AGENT_CMD):
		req.code = l.handleSyncByAgentCmd(req.conn, cl5Pkg)
	default:
		log.Errorf("receive invalid cmd[%d] from [%d]", cl5Pkg.GetCmd(), cl5Pkg.GetIp())
		req.code = l5PacketCmdFailed
	}
	l.PostProcess(req)
	return req.code
}

// handleSyncByAgentCmd 根据SID列表获取路由信息
func (l *L5pbserver) handleSyncByAgentCmd(conn net.Conn, iPkg *l5.Cl5Pkg) l5Code {
	ctx := context.Background()
	ctx = context.WithValue(ctx, model.Cl5ServerCluster{}, l.clusterName)
	ctx = context.WithValue(ctx, model.Cl5ServerProtocol{}, l.GetProtocol())
	syncByAgentAck, err := l.namingServer.SyncByAgentCmd(ctx, iPkg.GetSyncByAgentCmd())
	if err != nil {
		log.Errorf("%v", err)
		return l5SyncByAgentCmdFailed
	}

	oPkg := &l5.Cl5Pkg{
		Seqno:             proto.Int32(iPkg.GetSeqno()),
		Cmd:               proto.Int32(int32(l5.CL5_CMD_CL5_SYNC_BY_AGENT_ACK_CMD)),
		Result:            proto.Int32(success),
		Ip:                proto.Int32(iPkg.GetIp()),
		SyncByAgentAckCmd: syncByAgentAck,
	}

	log.Infof("handle sync by agent cmd, sid count(%d), callee count(%d), ip count(%d)",
		len(iPkg.GetSyncByAgentCmd().GetOptList().GetOpt()),
		len(oPkg.GetSyncByAgentAckCmd().GetServList().GetServ()),
		len(oPkg.GetSyncByAgentAckCmd().GetIpcList().GetIpc()))
	return response(conn, oPkg)
}

// handleRegisterByNameCmd 根据服务名列表寻找对应的SID列表
func (l *L5pbserver) handleRegisterByNameCmd(conn net.Conn, iPkg *l5.Cl5Pkg) l5Code {
	registerByNameAck, err := l.namingServer.RegisterByNameCmd(iPkg.GetRegisterByNameCmd())
	if err != nil {
		log.Errorf("%v", err)
		return l5RegisterByNameCmdFailed
	}

	oPkg := &l5.Cl5Pkg{
		Seqno:                proto.Int32(iPkg.GetSeqno()),
		Cmd:                  proto.Int32(int32(l5.CL5_CMD_CL5_REGISTER_BY_NAME_ACK_CMD)),
		Result:               proto.Int32(success),
		Ip:                   proto.Int32(iPkg.GetIp()),
		RegisterByNameAckCmd: registerByNameAck,
	}

	return response(conn, oPkg)
}

// checkRequest 请求包完整性检查
func checkRequest(buffer []byte) int {
	var length uint32
	isLittle := isLittleEndian()
	if isLittle {
		length = binary.LittleEndian.Uint32(buffer[sohSize:headSize])
	} else {
		length = binary.BigEndian.Uint32(buffer[sohSize:headSize])
	}

	return int(length)
}

func response(conn net.Conn, pkg *l5.Cl5Pkg) l5Code {
	responseData, err := proto.Marshal(pkg)
	if err != nil {
		log.Errorf("Marshal responseData error: %v", err)
		return l5MarshalPacketFailed
	}

	sohData := make([]byte, 2)
	lengthData := make([]byte, 4)
	var soh uint16 = 1
	length := uint32(binary.Size(responseData) + headSize)

	isLittle := isLittleEndian()
	if isLittle {
		binary.LittleEndian.PutUint16(sohData, soh)
		binary.LittleEndian.PutUint32(lengthData, length)
	} else {
		binary.BigEndian.PutUint16(sohData, soh)
		binary.BigEndian.PutUint32(lengthData, length)
	}

	sohData = append(sohData, lengthData...)
	sohData = append(sohData, responseData...)

	if _, err = conn.Write(sohData); err != nil {
		log.Errorf("conn write error: %v", err)
		return l5ResponseFailed
	}
	return l5Success
}

// isLittleEndian 判断系统是大端/小端存储
func isLittleEndian() bool {
	a := int16(0x1234)
	b := int8(a)

	return 0x34 == b
}
