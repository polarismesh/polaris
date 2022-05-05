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

package boltdb

import (
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
	api "github.com/polarismesh/polaris-server/common/api/v1"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/common/utils"
)

const (
	tblClient string = "client"

	ClientFieldHost     string = "Host"
	ClientFieldType     string = "Type"
	ClientFieldVersion  string = "Version"
	ClientFieldLocation string = "Location"
	ClientFieldId       string = "Id"
	ClientFieldStat     string = "Stat"
	ClientFieldCtime    string = "Ctime"
	ClientFieldMtime    string = "Mtime"
)

type clientObject struct {
	Host       string
	Type       string
	Version    string
	Location   map[string]string
	Id         string
	Ctime      time.Time
	Mtime      time.Time
	StatArrStr string
	Valid      bool
}

type clientStore struct {
	handler BoltHandler
}

// BatchAddClients insert the client info
func (cs *clientStore) BatchAddClients(clients []*model.Client) error {
	err := cs.handler.Execute(true, func(tx *bolt.Tx) error {
		for i := range clients {
			client := clients[i]
			saveVal, err := convertToClientObject(client)
			if err != nil {
				return err
			}

			if err := saveValue(tx, tblClient, saveVal.Id, saveVal); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// BatchDeleteClients delete the client info
func (cs *clientStore) BatchDeleteClients(ids []string) error {
	return cs.handler.DeleteValues(tblClient, ids, true)
}

// GetMoreClients 根据mtime获取增量clients，返回所有store的变更信息
func (cs *clientStore) GetMoreClients(mtime time.Time, firstUpdate bool) (map[string]*model.Client, error) {

	fields := []string{ClientFieldMtime}

	ret, err := cs.handler.LoadValuesByFilter(tblClient, fields, &clientObject{}, func(m map[string]interface{}) bool {
		if firstUpdate {
			return true
		}

		return m[ClientFieldMtime].(time.Time).After(mtime)
	})

	if err != nil {
		return nil, err
	}

	clients := make([]*model.Client, 0, len(ret))

	for _, v := range ret {
		client, err := convertToModelClient(v.(*clientObject))

		if err != nil {
			return nil, err
		}

		clients = append(clients, client)
	}

	return nil, nil
}

func convertToClientObject(client *model.Client) (*clientObject, error) {
	stat := client.Proto().Stat
	data, err := json.Marshal(stat)
	if err != nil {
		return nil, err
	}

	tn := time.Now()

	return &clientObject{
		Host:    client.Proto().Host.Value,
		Type:    client.Proto().Type.String(),
		Version: client.Proto().Version.Value,
		Location: map[string]string{
			"region": client.Proto().GetLocation().GetRegion().GetValue(),
			"zone":   client.Proto().GetLocation().GetZone().GetValue(),
			"campus": client.Proto().GetLocation().GetCampus().GetValue(),
		},
		Id:         client.Proto().Id.Value,
		Ctime:      tn,
		Mtime:      tn,
		StatArrStr: string(data),
		Valid:      true,
	}, nil
}

func convertToModelClient(client *clientObject) (*model.Client, error) {
	stat := make([]*api.StatInfo, 0, 4)
	err := json.Unmarshal([]byte(client.StatArrStr), &stat)
	if err != nil {
		return nil, err
	}

	c := &api.Client{
		Id:      utils.NewStringValue(client.Id),
		Host:    utils.NewStringValue(client.Host),
		Type:    api.Client_ClientType(api.Client_ClientType_value[client.Type]),
		Version: utils.NewStringValue(client.Version),
		Ctime:   utils.NewStringValue(client.Ctime.Format("2006-01-02 15:04:05")),
		Mtime:   utils.NewStringValue(client.Mtime.Format("2006-01-02 15:04:05")),
		Location: &api.Location{
			Region: utils.NewStringValue(client.Location["region"]),
			Zone:   utils.NewStringValue(client.Location["zone"]),
			Campus: utils.NewStringValue(client.Location["campus"]),
		},
		Stat: stat,
	}

	return model.NewClient(c), nil
}
