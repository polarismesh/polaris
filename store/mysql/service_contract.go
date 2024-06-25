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
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

type serviceContractStore struct {
	master *BaseDB
	slave  *BaseDB
}

// CreateServiceContract 创建服务契约
func (s *serviceContractStore) CreateServiceContract(contract *model.ServiceContract) error {
	addSql := "INSERT INTO service_contract(`id`,`type`, `namespace`, `service`, `protocol`,`version`, " +
		" `revision`, `flag`, `content`, `ctime`, `mtime`" +
		") VALUES (?,?,?,?,?,?,?,0,?,sysdate(),sysdate())"

	_, err := s.master.Exec(addSql, []interface{}{
		contract.ID,
		contract.Type,
		contract.Namespace,
		contract.Service,
		contract.Protocol,
		contract.Version,
		contract.Revision,
		contract.Content,
	}...)
	return store.Error(err)
}

// UpdateServiceContract 更新服务契约信息
func (s *serviceContractStore) UpdateServiceContract(contract *model.ServiceContract) error {
	updateSql := "UPDATE service_contract SET content = ?, revision = ?, mtime = sysdate() WHERE id = ?"
	_, err := s.master.Exec(updateSql, contract.Content, contract.Revision, contract.ID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteServiceContract 删除服务契约 删除该版本的全部数据
func (s *serviceContractStore) DeleteServiceContract(contract *model.ServiceContract) error {
	return s.master.processWithTransaction("DeleteServiceContract", func(tx *BaseTx) error {
		deleteSql := "UPDATE service_contract SET flag = 1, mtime = sysdate() WHERE id = ?"
		if _, err := tx.Exec(deleteSql, []interface{}{
			contract.ID,
		}...); err != nil {
			log.Errorf("[Store][database] all delete service contract err: %s", err.Error())
			return err
		}

		deleteDetailSql := "UPDATE service_contract_detail SET flag = 1 WHERE contract_id = ?"
		if _, err := tx.Exec(deleteDetailSql, []interface{}{
			contract.ID,
		}...); err != nil {
			log.Errorf("[Store][database] all delete service contract err: %s", err.Error())
			return err
		}

		return tx.Commit()
	})
}

// GetServiceContract 通过ID查询服务契约数据
func (s *serviceContractStore) GetServiceContract(id string) (*model.EnrichServiceContract, error) {
	querySql := "SELECT id, type, namespace, service, protocol, version, revision, flag, content, " +
		" UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM service_contract WHERE flag = 0 AND id = ?"

	args := []interface{}{id}

	list := make([]*model.EnrichServiceContract, 0)
	err := s.slave.processWithTransaction("GetServiceContract", func(tx *BaseTx) error {
		rows, err := tx.Query(querySql, args...)
		if err != nil {
			log.Error("[Store][Contract] list contract ", zap.String("query sql", querySql), zap.Any("args", args))
			return err
		}
		err = transferEnrichServiceContract(rows, func(contract *model.EnrichServiceContract) {
			list = append(list, contract)
		})
		if err != nil {
			log.Errorf("[Store][Contract] fetch contract rows scan err: %s", err.Error())
			return err
		}

		for i := range list {
			contract := list[i]
			queryDetailSql := "SELECT id, contract_id, namespace, service, protocol, version, type, method, path, content, revision, " +
				" UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), IFNULL(source, 1) " +
				" FROM service_contract_detail " +
				" WHERE contract_id = ?"

			descriptors, err := s.loadContractInterfaces(tx, queryDetailSql, []interface{}{contract.ID})
			if err != nil {
				log.Error("[Store][Contract] load service_contract link interfaces",
					zap.String("contract_id", contract.ID), zap.Error(err))
				return err
			}
			contract.Interfaces = descriptors
		}
		return nil
	})
	if err != nil {
		return nil, store.Error(err)
	}

	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

func transferEnrichServiceContract(rows *sql.Rows, consumer func(contract *model.EnrichServiceContract)) error {
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var flag, ctime, mtime int64
		contract := model.EnrichServiceContract{
			ServiceContract: &model.ServiceContract{},
		}
		if scanErr := rows.Scan(&contract.ID, &contract.Type, &contract.Namespace, &contract.Service,
			&contract.Protocol, &contract.Version, &contract.Revision, &flag,
			&contract.Content, &ctime, &mtime); scanErr != nil {
			log.Errorf("[Store][Contract] fetch contract rows scan err: %s", scanErr.Error())
			return scanErr
		}

		contract.Valid = flag == 0
		contract.CreateTime = time.Unix(0, ctime)
		contract.ModifyTime = time.Unix(0, mtime)
		consumer(&contract)
	}
	return nil
}

// AddServiceContractInterfaces 创建服务契约API接口
func (s *serviceContractStore) AddServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.master.processWithTransaction("AddServiceContractInterfaces", func(tx *BaseTx) error {
		updateRevision := "UPDATE service_contract SET revision = ?, mtime = sysdate() WHERE id = ?"
		if _, err := tx.Exec(updateRevision, contract.Revision, contract.ID); err != nil {
			log.Errorf("[Store][database] update service contract revision err: %s", err.Error())
			return err
		}

		// 新增批量数据
		for _, item := range contract.Interfaces {
			addSql := "REPLACE INTO service_contract_detail (id, contract_id, namespace, service, protocol, " +
				" version, method, path, type, content, revision, flag, ctime, mtime, source) " +
				" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, sysdate(), sysdate(), ?)"
			if _, err := tx.Exec(addSql, []interface{}{
				item.ID,
				contract.ID,
				item.Namespace,
				item.Service,
				item.Protocol,
				item.Version,
				item.Method,
				item.Path,
				item.Type,
				item.Content,
				item.Revision,
				0,
				int(item.Source),
			}...); err != nil {
				log.Errorf("[Store][database] add service contract detail err: %s", err.Error())
				return err
			}
		}
		return tx.Commit()
	})
}

// AppendServiceContractInterfaces 追加服务契约API接口
func (s *serviceContractStore) AppendServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.master.processWithTransaction("AppendServiceContractDetail", func(tx *BaseTx) error {
		updateRevision := "UPDATE service_contract SET revision = ?, mtime = sysdate() WHERE id = ?"
		if _, err := tx.Exec(updateRevision, contract.Revision, contract.ID); err != nil {
			log.Errorf("[Store][database] update service contract revision err: %s", err.Error())
			return err
		}
		for _, item := range contract.Interfaces {
			addSql := "REPLACE INTO service_contract_detail(`id`,`contract_id`, `namespace`, `service`, " +
				" `protocol`, `version`, `method`, `path`, `type` ,`content`,`revision`" +
				",`flag`,`ctime`, `mtime`,`source`" +
				") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,sysdate(),sysdate(),?)"

			if _, err := tx.Exec(addSql, []interface{}{
				item.ID,
				contract.ID,
				item.Namespace,
				item.Service,
				item.Protocol,
				item.Version,
				item.Method,
				item.Path,
				item.Type,
				item.Content,
				item.Revision,
				0,
				int(item.Source),
			}...); err != nil {
				log.Errorf("[Store][database] append service contract detail err: %s", err.Error())
				return err
			}
		}
		return tx.Commit()
	})
}

// DeleteServiceContractInterfaces 删除服务契约API接口
func (s *serviceContractStore) DeleteServiceContractInterfaces(contract *model.EnrichServiceContract) error {
	return s.master.processWithTransaction("DeleteServiceContractInterfaces", func(tx *BaseTx) error {
		updateRevision := "UPDATE service_contract SET revision = ?, mtime = sysdate() WHERE id = ?"
		if _, err := tx.Exec(updateRevision, contract.Revision, contract.ID); err != nil {
			log.Errorf("[Store][database] update service contract revision err: %s", err.Error())
			return err
		}
		for _, item := range contract.Interfaces {
			addSql := "UPDATE service_contract_detail SET flag = 1 WHERE contract_id = ? AND method = ? AND path = ? AND type = ?"

			if _, err := tx.Exec(addSql, []interface{}{
				item.ContractID,
				item.Method,
				item.Path,
				item.Type,
			}...); err != nil {
				log.Errorf("[Store][database] delete service contract detail err: %s", err.Error())
				return err
			}
		}
		return tx.Commit()
	})
}

func (s *serviceContractStore) GetServiceContracts(ctx context.Context, filter map[string]string,
	offset, limit uint32) (uint32, []*model.EnrichServiceContract, error) {

	if _, ok := filter["order_field"]; !ok {
		filter["order_field"] = "mtime"
	}
	if _, ok := filter["order_type"]; !ok {
		filter["order_type"] = "DESC"
	}

	querySql := `
 SELECT id, type, namespace, service, protocol
	 , version, revision, flag, content
	 , UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
 FROM service_contract
 WHERE flag = 0 
 `

	countSql := "SELECT COUNT(*) FROM service_contract WHERE flag = 0 "

	brief := filter[briefSearch] == "true"
	args := make([]interface{}, 0, len(filter))
	conditions := make([]string, 0, len(filter))
	for k, v := range filter {
		if k == "order_field" || k == "order_type" {
			continue
		}
		if utils.IsWildName(v) {
			conditions = append(conditions, k+" LIKE ? ")
			args = append(args, utils.ParseWildNameForSql(v))
		} else {
			conditions = append(conditions, k+" = ? ")
			args = append(args, v)
		}
	}

	if len(conditions) > 0 {
		countSql += " AND " + strings.Join(conditions, " AND ")
		querySql += " AND " + strings.Join(conditions, " AND ")
	}
	querySql += fmt.Sprintf(" ORDER BY %s %s LIMIT ?, ? ", filter["order_field"], filter["order_type"])

	var count int64
	var list = make([]*model.EnrichServiceContract, 0, limit)

	err := s.master.processWithTransaction("GetServiceContracts", func(tx *BaseTx) error {
		row := tx.QueryRow(countSql, args...)
		if err := row.Scan(&count); err != nil {
			log.Error("[Store][Contract] count service_contracts", zap.String("count", countSql), zap.Any("args", args), zap.Error(err))
			return err
		}

		args = append(args, offset, limit)
		rows, err := tx.Query(querySql, args...)
		if err != nil {
			log.Error("[Store][Contract] list service_contracts", zap.String("query", querySql), zap.Any("args", args), zap.Error(err))
			return err
		}
		defer func() {
			_ = rows.Close()
		}()

		contractIds := make([]interface{}, 0, 32)
		err = transferEnrichServiceContract(rows, func(contract *model.EnrichServiceContract) {
			list = append(list, contract)
			contractIds = append(contractIds, contract.ID)
		})
		if err != nil {
			log.Errorf("[Store][Contract] fetch contract rows scan err: %s", err.Error())
			return err
		}

		if !brief && len(contractIds) > 0 {
			// 加载 interfaces 列表
			queryDetailSql := fmt.Sprintf("SELECT id, contract_id, namespace, service, protocol, version, "+
				" type, method, path, content, revision, "+
				" UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime), IFNULL(source, 1) "+
				" FROM service_contract_detail "+
				" WHERE contract_id IN (%s)", placeholders(len(contractIds)))

			contractDetailMap := map[string][]*model.InterfaceDescriptor{}
			interfaces, err := s.loadContractInterfaces(tx, queryDetailSql, contractIds)
			if err != nil {
				return err
			}
			for i := range interfaces {
				descriptor := interfaces[i]
				if _, ok := contractDetailMap[descriptor.ContractID]; !ok {
					contractDetailMap[descriptor.ContractID] = make([]*model.InterfaceDescriptor, 0, 4)
				}
				contractDetailMap[descriptor.ContractID] = append(contractDetailMap[descriptor.ContractID], descriptor)
			}

			for _, item := range list {
				methods := contractDetailMap[item.ID]
				item.Interfaces = methods
				item.Format()
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, store.Error(err)
	}
	return uint32(count), list, nil
}

func (s *serviceContractStore) GetInterfaceDescriptors(ctx context.Context, filter map[string]string,
	offset, limit uint32) (uint32, []*model.InterfaceDescriptor, error) {

	countSql := "SELECT COUNT(*) FROM service_contract_detail sd WHERE flag = 0 "

	// 加载 interfaces 列表
	querySql := `
 SELECT id, contract_id, namespace, service, protocol, version
	 , type, method, path, content, revision
	 , UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
	 , IFNULL(source, 1)
 FROM service_contract_detail
 WHERE flag = 0
 `

	args := make([]interface{}, 0, len(filter))
	conditions := make([]string, 0, len(filter))

	for k, v := range filter {
		if k == "order_field" || k == "order_type" {
			continue
		}
		conditions = append(conditions, k+" = ? ")
		args = append(args, v)
	}

	var count uint32
	var list []*model.InterfaceDescriptor

	if len(conditions) > 0 {
		countSql += " AND " + strings.Join(conditions, " AND ")
		querySql += " AND " + strings.Join(conditions, " AND ")
	}
	querySql += fmt.Sprintf(" ORDER BY %s %s LIMIT ?, ? ", filter["order_field"], filter["order_type"])

	err := s.slave.processWithTransaction("GetInterfaceDescriptors", func(tx *BaseTx) error {
		row := tx.QueryRow(countSql, args...)
		err := row.Scan(&count)
		if err != nil {
			log.Error("[Store][Contract] count service_interfaces", zap.String("countSql", countSql), zap.Any("args", args), zap.Error(err))
			return err
		}

		args = append(args, offset, limit)
		list, err = s.loadContractInterfaces(tx, querySql, args)
		return err
	})
	if err != nil {
		return 0, nil, store.Error(err)
	}
	return count, list, nil
}

func (s *serviceContractStore) loadContractInterfaces(tx *BaseTx, query string, args []interface{}) ([]*model.InterfaceDescriptor, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		log.Error("[Store][Contract] load service_contract interface list", zap.String("sql", query), zap.Error(err))
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	var list []*model.InterfaceDescriptor
	for rows.Next() {
		var flag, ctime, mtime, source int64
		detailItem := &model.InterfaceDescriptor{}
		if scanErr := rows.Scan(
			&detailItem.ID, &detailItem.ContractID, &detailItem.Namespace, &detailItem.Service, &detailItem.Protocol,
			&detailItem.Version, &detailItem.Type, &detailItem.Method,
			&detailItem.Path, &detailItem.Content, &detailItem.Revision,
			&ctime, &mtime, &source,
		); scanErr != nil {
			log.Error("[Store][Contract] load service_contract interface rows scan", zap.Error(scanErr))
			return nil, err
		}

		detailItem.Valid = flag == 0
		detailItem.CreateTime = time.Unix(ctime, 0)
		detailItem.ModifyTime = time.Unix(mtime, 0)
		switch source {
		case 2:
			detailItem.Source = service_manage.InterfaceDescriptor_Client
		default:
			detailItem.Source = service_manage.InterfaceDescriptor_Manual
		}

		list = append(list, detailItem)
	}
	return list, nil
}

// ListVersions .
func (s *serviceContractStore) ListVersions(ctx context.Context, service, namespace string) ([]*model.ServiceContract, error) {
	list := make([]*model.ServiceContract, 0, 4)

	querySql := `
 SELECT id, type, namespace, service, protocol
	 , version, revision, flag, content
	 , UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime)
 FROM service_contract
 WHERE flag = 0 AND namespace = ? AND service = ?
 `

	rows, err := s.slave.Query(querySql, namespace, service)
	if err != nil {
		log.Error("[Store][Contract] list version service_contract", zap.String("namespace", namespace),
			zap.String("service", service), zap.Error(err))
		return nil, store.Error(err)
	}

	err = transferEnrichServiceContract(rows, func(contract *model.EnrichServiceContract) {
		list = append(list, contract.ServiceContract)
	})
	if err != nil {
		log.Errorf("[Store][Contract] fetch contract rows scan err: %s", err.Error())
		return nil, err
	}
	return list, nil
}

// GetMoreServiceContracts .
func (s *serviceContractStore) GetMoreServiceContracts(firstUpdate bool, mtime time.Time) ([]*model.EnrichServiceContract, error) {
	querySql := "SELECT id, type, namespace, service, protocol, version, revision, flag, content, " +
		" UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(mtime) FROM service_contract WHERE mtime >= ? "
	if firstUpdate {
		mtime = time.Unix(0, 1)
		querySql += " AND flag = 0 "
	}

	tx, err := s.slave.Begin()
	if err != nil {
		log.Error("[Store][Contract] list contract for cache when begin tx", zap.Error(err))
		return nil, store.Error(err)
	}
	defer func() {
		_ = tx.Commit()
	}()

	rows, err := tx.Query(querySql, mtime)
	if err != nil {
		log.Error("[Store][Contract] list contract for cache when query", zap.Error(err))
		return nil, store.Error(err)
	}
	defer func() {
		_ = rows.Close()
	}()

	list := make([]*model.EnrichServiceContract, 0)
	for rows.Next() {
		var flag, ctime, mtime int64
		contract := &model.ServiceContract{}
		if scanErr := rows.Scan(&contract.ID, &contract.Type, &contract.Namespace, &contract.Service,
			&contract.Protocol, &contract.Version, &contract.Revision, &flag,
			&contract.Content, &ctime, &mtime); scanErr != nil {
			log.Error("[Store][Contract] fetch contract rows scan err: %s", zap.Error(err))
			return nil, store.Error(err)
		}

		contract.Valid = flag == 0
		contract.CreateTime = time.Unix(ctime, 0)
		contract.ModifyTime = time.Unix(mtime, 0)

		list = append(list, &model.EnrichServiceContract{
			ServiceContract: contract,
		})
	}

	contractDetailMap := map[string][]*model.InterfaceDescriptor{}
	if len(list) > 0 {
		queryDetailSql := "SELECT sd.id, sd.contract_id, sd.type, sd.method, sd.path, sd.content, sd.revision, " +
			" UNIX_TIMESTAMP(sd.ctime), UNIX_TIMESTAMP(sd.mtime), IFNULL(sd.source, 1) " +
			" FROM service_contract_detail sd  LEFT JOIN service_contract sc ON sd.contract_id = sc.id " +
			" WHERE sc.mtime >= ?"
		detailRows, err := tx.Query(queryDetailSql, mtime)
		if err != nil {
			log.Error("[Store][Contract] list contract detail", zap.String("query sql", queryDetailSql), zap.Error(err))
			return nil, store.Error(err)
		}
		defer func() {
			_ = detailRows.Close()
		}()
		for detailRows.Next() {
			var flag, ctime, mtime, source int64
			detailItem := &model.InterfaceDescriptor{}
			if scanErr := detailRows.Scan(
				&detailItem.ID, &detailItem.ContractID, &detailItem.Type, &detailItem.Method,
				&detailItem.Path, &detailItem.Content, &detailItem.Revision,
				&ctime, &mtime, &source,
			); scanErr != nil {
				log.Error("[Store][Contract] fetch contract detail rows scan", zap.Error(scanErr))
				return nil, store.Error(scanErr)
			}

			detailItem.Valid = flag == 0
			detailItem.CreateTime = time.Unix(ctime, 0)
			detailItem.ModifyTime = time.Unix(mtime, 0)
			switch source {
			case 2:
				detailItem.Source = service_manage.InterfaceDescriptor_Client
			default:
				detailItem.Source = service_manage.InterfaceDescriptor_Manual
			}

			if _, ok := contractDetailMap[detailItem.ContractID]; !ok {
				contractDetailMap[detailItem.ContractID] = make([]*model.InterfaceDescriptor, 0, 4)
			}
			contractDetailMap[detailItem.ContractID] = append(contractDetailMap[detailItem.ContractID], detailItem)
		}

		for _, item := range list {
			methods := contractDetailMap[item.ID]
			item.Interfaces = methods
			item.Format()
		}
	}
	return list, nil
}
