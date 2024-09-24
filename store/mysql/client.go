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

package sqldb

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/store"
)

type clientStore struct {
	master *BaseDB
	slave  *BaseDB // 缓存相关的读取，请求到slave
}

// CreateClient insert the client info
func (cs *clientStore) CreateClient(client *model.Client) error {
	clientID := client.Proto().GetId().GetValue()
	if len(clientID) == 0 {
		log.Errorf("[Store][database] add business missing id")
		return fmt.Errorf("add Business missing some params, id %s, name %s", clientID,
			client.Proto().GetHost().GetValue())
	}
	err := RetryTransaction("createClient", func() error {
		return cs.createClient(client)
	})
	return store.Error(err)
}

// UpdateClient update the client info
func (cs *clientStore) UpdateClient(client *model.Client) error {
	err := RetryTransaction("updateClient", func() error {
		return cs.updateClient(client)
	})
	if err == nil {
		return nil
	}

	serr := store.Error(err)
	if store.Code(serr) == store.DuplicateEntryErr {
		serr = store.NewStatusError(store.DataConflictErr, err.Error())
	}
	return serr
}

// deleteClient delete the client info
func deleteClient(tx *BaseTx, clientID string) error {
	if clientID == "" {
		return errors.New("delete client missing client id")
	}

	str := "update client set flag = 1, mtime = sysdate() where `id` = ?"
	_, err := tx.Exec(str, clientID)
	return store.Error(err)
}

// BatchAddClients 增加多个实例
func (cs *clientStore) BatchAddClients(clients []*model.Client) error {
	err := RetryTransaction("batchAddClients", func() error {
		return cs.batchAddClients(clients)
	})
	if err == nil {
		return nil
	}
	return store.Error(err)
}

// BatchDeleteClients 批量删除实例，flag=1
func (cs *clientStore) BatchDeleteClients(ids []string) error {
	err := RetryTransaction("batchDeleteClients", func() error {
		return cs.batchDeleteClients(ids)
	})
	if err == nil {
		return nil
	}
	return store.Error(err)
}

// GetMoreClients 根据mtime获取增量clients，返回所有store的变更信息
func (cs *clientStore) GetMoreClients(mtime time.Time, firstUpdate bool) (map[string]*model.Client, error) {
	str := `select client.id, client.host, client.type, IFNULL(client.version,""), IFNULL(client.region, ""),
		 IFNULL(client.zone, ""), IFNULL(client.campus, ""), client.flag,  IFNULL(client_stat.target, ""), 
		 IFNULL(client_stat.port, 0), IFNULL(client_stat.protocol, ""), IFNULL(client_stat.path, ""), 
		 UNIX_TIMESTAMP(client.ctime), UNIX_TIMESTAMP(client.mtime)
		 from client left join client_stat on client.id = client_stat.client_id `
	str += " where client.mtime >= FROM_UNIXTIME(?)"
	if firstUpdate {
		str += " and flag = 0"
	}
	rows, err := cs.slave.Query(str, timeToTimestamp(mtime))
	if err != nil {
		log.Errorf("[Store][database] get more client query err: %s", err.Error())
		return nil, err
	}

	out := make(map[string]*model.Client)
	err = callFetchClientRows(rows, func(entry *model.ClientStore) (b bool, e error) {
		outClient, ok := out[entry.ID]
		if !ok {
			out[entry.ID] = model.Store2Client(entry)
		} else {
			statInfo := model.Store2ClientStat(&entry.Stat)
			outClient.Proto().Stat = append(outClient.Proto().Stat, statInfo)
		}
		return true, nil
	})
	if err != nil {
		log.Errorf("[Store][database] call fetch client rows err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

func (cs *clientStore) batchAddClients(clients []*model.Client) error {
	tx, err := cs.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] batch add clients tx begin err: %s", err.Error())
		return err
	}
	defer func() { _ = tx.Rollback() }()

	ids := make([]string, 0, len(clients))
	var client2StatInfos = make(map[string][]*apiservice.StatInfo)
	builder := strings.Builder{}
	for idx, entry := range clients {
		if idx > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("?")
		ids = append(ids, entry.Proto().GetId().GetValue())
		var statInfos []*apiservice.StatInfo
		if len(entry.Proto().GetStat()) > 0 {
			statInfos = append(statInfos, entry.Proto().GetStat()...)
			client2StatInfos[entry.Proto().GetId().GetValue()] = statInfos
		}
	}
	if err = batchCleanClientStats(tx, ids); nil != err {
		log.Errorf("[Store][database] batch clean client stat err: %s", err.Error())
		return err
	}
	if err = batchAddClientMain(tx, clients); nil != err {
		log.Errorf("[Store][database] batch add clients err: %s", err.Error())
		return err
	}
	if err = batchAddClientStat(tx, client2StatInfos); nil != err {
		log.Errorf("[Store][database] batch add clientStats err: %s", err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] batch add clients commit tx err: %s", err.Error())
		return err
	}
	return nil
}

func (cs *clientStore) batchDeleteClients(ids []string) error {
	tx, err := cs.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] batch delete clients tx begin err: %s", err.Error())
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err = batchCleanClientStats(tx, ids); nil != err {
		log.Errorf("[Store][database] batch clean client stat err: %s", err.Error())
		return err
	}
	if err = batchDeleteClientsMain(tx, ids); nil != err {
		log.Errorf("[Store][database] batch delete clients err: %s", err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] batch delete clients commit tx err: %s", err.Error())
		return err
	}
	return nil
}

func batchDeleteClientsMain(tx *BaseTx, ids []string) error {
	args := make([]interface{}, 0, len(ids))
	for i := range ids {
		args = append(args, ids[i])
	}

	return BatchOperation("batch-delete-clients", args, func(objects []interface{}) error {
		if len(objects) == 0 {
			return nil
		}
		str := `update client set flag = 1, mtime = sysdate() where id in ( ` + PlaceholdersN(len(objects)) + `)`
		_, err := tx.Exec(str, objects...)
		return store.Error(err)
	})
}

func batchCleanClientStats(tx *BaseTx, ids []string) error {
	args := make([]interface{}, 0, len(ids))
	for i := range ids {
		args = append(args, ids[i])
	}

	return BatchOperation("batch-delete-client-stats", args, func(objects []interface{}) error {
		if len(objects) == 0 {
			return nil
		}
		str := `delete from client_stat where client_id in (` + PlaceholdersN(len(objects)) + `)`
		_, err := tx.Exec(str, objects...)
		return store.Error(err)
	})
}

func (cs *clientStore) GetClientStat(clientID string) ([]*model.ClientStatStore, error) {
	str := "select `target`, `port`, `protocol`, `path` from client_stat where client.id = ?"
	rows, err := cs.master.Query(str, clientID)
	if err != nil {
		log.Errorf("[Store][database] query client stat err: %s", err.Error())
		return nil, err
	}
	defer rows.Close()

	var clientStatStores []*model.ClientStatStore
	for rows.Next() {
		clientStatStore := &model.ClientStatStore{}
		if err := rows.Scan(&clientStatStore.Target,
			&clientStatStore.Port, &clientStatStore.Protocol, &clientStatStore.Path); err != nil {
			log.Errorf("[Store][database] get client meta rows scan err: %s", err.Error())
			return nil, err
		}
		clientStatStores = append(clientStatStores, clientStatStore)
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] get client meta rows next err: %s", err.Error())
		return nil, err
	}

	return clientStatStores, nil
}

// callFetchClientRows 带回调的fetch client
func callFetchClientRows(rows *sql.Rows, callback func(entry *model.ClientStore) (bool, error)) error {
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var item model.ClientStore
	progress := 0
	for rows.Next() {
		progress++
		if progress%100000 == 0 {
			log.Infof("[Store][database] client fetch rows progress: %d", progress)
		}
		err := rows.Scan(&item.ID, &item.Host, &item.Type, &item.Version, &item.Region, &item.Zone,
			&item.Campus, &item.Flag, &item.Stat.Target, &item.Stat.Port, &item.Stat.Protocol,
			&item.Stat.Path, &item.CreateTime, &item.ModifyTime)
		if err != nil {
			log.Errorf("[Store][database] fetch client rows err: %s", err.Error())
			return err
		}
		ok, err := callback(&item)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] client rows catch err: %s", err.Error())
		return err
	}

	return nil
}

func (cs *clientStore) createClient(client *model.Client) error {
	tx, err := cs.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] create client tx begin err: %s", err.Error())
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// clean the old items before add
	if err := deleteClient(tx, client.Proto().GetId().GetValue()); err != nil {
		return err
	}
	if err := addClientMain(tx, client); err != nil {
		log.Errorf("[Store][database] add client main err: %s", err.Error())
		return err
	}
	if err := addClientStat(tx, client); err != nil {
		log.Errorf("[Store][database] add client stat err: %s", err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] create client commit tx err: %s", err.Error())
		return err
	}
	return nil
}

func (cs *clientStore) updateClient(client *model.Client) error {
	tx, err := cs.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] update client tx begin err: %s", err.Error())
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := updateClientMain(tx, client); err != nil {
		log.Errorf("[Store][database] update client main err: %s", err.Error())
		return err
	}

	if err := updateClientStat(tx, client); err != nil {
		log.Errorf("[Store][database] update client stat err: %s", err.Error())
		return err
	}
	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update client commit tx err: %s", err.Error())
		return err
	}
	return nil
}

func addClientMain(tx *BaseTx, client *model.Client) error {
	str := `insert into client(id, host, type, version, region, zone, campus, flag, ctime, mtime)
			 values(?, ?, ?, ?, ?, ?, ?, 0, sysdate(), sysdate())`
	_, err := tx.Exec(str,
		client.Proto().GetId().GetValue(),
		client.Proto().GetHost().GetValue(),
		client.Proto().GetType().String(),
		client.Proto().GetVersion().GetValue(),
		client.Proto().GetLocation().GetRegion().GetValue(),
		client.Proto().GetLocation().GetZone().GetValue(),
		client.Proto().GetLocation().GetCampus().GetValue(),
	)
	return err
}

func batchAddClientMain(tx *BaseTx, clients []*model.Client) error {
	str := `replace into client(id, host, type, version, region, zone, campus, flag, ctime, mtime)
		 values`
	first := true
	args := make([]interface{}, 0)
	for _, client := range clients {
		if !first {
			str += ","
		}
		str += "(?, ?, ?, ?, ?, ?, ?, 0, sysdate(), sysdate())"
		first = false

		args = append(args, client.Proto().GetId().GetValue(),
			client.Proto().GetHost().GetValue(),
			client.Proto().GetType().String())
		args = append(args, client.Proto().GetVersion().GetValue(),
			client.Proto().GetLocation().GetRegion().GetValue(),
			client.Proto().GetLocation().GetZone().GetValue(),
			client.Proto().GetLocation().GetCampus().GetValue())
	}
	_, err := tx.Exec(str, args...)
	return err
}

func batchAddClientStat(tx *BaseTx, client2Stats map[string][]*apiservice.StatInfo) error {
	if len(client2Stats) == 0 {
		return nil
	}
	str := `insert into client_stat(client_id, target, port, protocol, path)
			 values`
	first := true
	args := make([]interface{}, 0)
	for clientId, stats := range client2Stats {
		for _, entry := range stats {
			if !first {
				str += ","
			}
			str += "(?, ?, ?, ?, ?)"
			first = false
			args = append(args,
				clientId,
				entry.GetTarget().GetValue(),
				entry.GetPort().GetValue(),
				entry.GetProtocol().GetValue(),
				entry.GetPath().GetValue())
		}
	}
	_, err := tx.Exec(str, args...)
	return err
}

func addClientStat(tx *BaseTx, client *model.Client) error {
	stats := client.Proto().GetStat()
	if len(stats) == 0 {
		return nil
	}
	str := `insert into client_stat(client_id, target, port, protocol, path)
			 values`
	first := true
	args := make([]interface{}, 0)
	for _, entry := range stats {
		if !first {
			str += ","
		}
		str += "(?, ?, ?, ?, ?)"
		first = false
		args = append(args,
			client.Proto().GetId().GetValue(),
			entry.GetTarget().GetValue(),
			entry.GetPort().GetValue(),
			entry.GetProtocol().GetValue(),
			entry.GetPath().GetValue())
	}
	_, err := tx.Exec(str, args...)
	return err
}

func updateClientMain(tx *BaseTx, client *model.Client) error {
	str := `update client set host = ?,
	 type = ?, version = ?, region = ?, zone = ?, campus = ?, mtime = sysdate() where id = ?`

	_, err := tx.Exec(str,
		client.Proto().GetHost().GetValue(),
		client.Proto().GetType().String(),
		client.Proto().GetVersion().GetValue(),
		client.Proto().GetLocation().GetRegion().GetValue(),
		client.Proto().GetLocation().GetZone().GetValue(),
		client.Proto().GetLocation().GetCampus().GetValue(),
		client.Proto().GetId().GetValue(),
	)

	return err
}

// updateClientStat 更新client的stat表
func updateClientStat(tx *BaseTx, client *model.Client) error {
	deleteStr := "delete from client_stat where cliend_id = ?"
	if _, err := tx.Exec(deleteStr, client.Proto().GetId().GetValue()); err != nil {
		return err
	}
	return addClientStat(tx, client)
}
