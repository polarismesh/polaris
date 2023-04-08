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

package heartbeatp2p

import (
	"context"
	"fmt"

	commonhash "github.com/polarismesh/polaris/common/hash"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Peer
type Peer struct {
	Local  bool
	ID     string
	Host   string
	Port   uint32
	Index  int32
	Conn   *grpc.ClientConn
	Client CheckerPeerServiceClient
	Putter CheckerPeerService_PutRecordsClient
	Delter CheckerPeerService_DelRecordsClient
	Cache  BeatRecordCache
}

func (p *Peer) Serve() error {
	if p.Local {
		p.Cache = newLocalBeatRecordCache(64, commonhash.Fnv32)
	} else {
		opts := []grpc.DialOption{
			grpc.WithBlock(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("%s:%d", p.Host, p.Port), opts)
		if err != nil {
			return err
		}
		p.Conn = conn
		p.Client = NewCheckerPeerServiceClient(p.Conn)
		putter, err := p.Client.PutRecords(context.Background())
		if err != nil {
			return err
		}
		p.Putter = putter
		delter, err := p.Client.DelRecords(context.Background())
		if err != nil {
			return err
		}
		p.Delter = delter
		p.Cache = newRemoteBeatRecordCache(func(req *GetRecordsRequest) *GetRecordsResponse {
			resp, err := p.Client.GetRecords(context.Background(), req)
			if err != nil {
				return nil
			}
			return resp
		}, func(req *PutRecordsRequest) {
			if err := p.Putter.Send(req); err != nil {

			}
		}, func(req *DelRecordsRequest) {
			if err := p.Delter.Send(req); err != nil {

			}
		})
	}
	return nil
}

func (p *Peer) Close() error {
	if p.Conn != nil {
		return p.Conn.Close()
	}
	return nil
}
