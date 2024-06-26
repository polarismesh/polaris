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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/polarismesh/polaris/common/model"
	"github.com/polarismesh/polaris/common/utils"
	"github.com/polarismesh/polaris/store"
)

// serviceStore 实现了ServiceStore
type serviceStore struct {
	master *BaseDB
	slave  *BaseDB
}

// AddService 增加服务
func (ss *serviceStore) AddService(s *model.Service) error {
	if s.ID == "" || s.Name == "" || s.Namespace == "" {
		return store.NewStatusError(store.EmptyParamsErr, fmt.Sprintf(
			"add service missing some params, id is %s, name is %s, namespace is %s", s.ID, s.Name, s.Namespace))
	}

	err := RetryTransaction("addService", func() error {
		return ss.addService(s)
	})
	return store.Error(err)
}

// addService add service
func (ss *serviceStore) addService(s *model.Service) error {
	tx, err := ss.master.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 先清理无效数据
	if err := cleanService(tx, s.Name, s.Namespace); err != nil {
		return err
	}

	// 填充main表
	if err := addServiceMain(tx, s); err != nil {
		log.Errorf("[Store][database] add service table err: %s", err.Error())
		return err
	}

	// 填充metadata表
	if err := addServiceMeta(tx, s.ID, s.Meta); err != nil {
		log.Errorf("[Store][database] add service meta table err: %s", err.Error())
		return err
	}

	// 填充owner_service_map表
	if err := addOwnerServiceMap(tx, s.Name, s.Namespace, s.Owner); err != nil {
		log.Errorf("[Store][database] add owner_service_map table err: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add service tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// DeleteService 删除服务
func (ss *serviceStore) DeleteService(id, serviceName, namespaceName string) error {
	err := RetryTransaction("deleteService", func() error {
		return ss.deleteService(id, serviceName, namespaceName)
	})
	return store.Error(err)
}

// deleteService 删除服务的内部函数
func (ss *serviceStore) deleteService(id, serviceName, namespaceName string) error {
	tx, err := ss.master.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 删除服务
	if err := deleteServiceByID(tx, id); err != nil {
		log.Errorf("[Store][database] delete service(%s) err : %s", id, err.Error())
		return err
	}

	// 删除负责人、服务映射表对应记录
	if err := deleteOwnerServiceMap(tx, serviceName, namespaceName); err != nil {
		log.Errorf("[Store][database] delete owner_service_map(%s) err : %s", id, err.Error())
		return err
	}

	if err := deleteServiceMetadata(tx, serviceName, namespaceName); err != nil {
		log.Errorf("[Store][database] delete service_metadata(%s) err : %s", id, err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] add service tx commit err: %s", err.Error())
		return err
	}

	return nil
}

// deleteServiceByID 删除服务或服务别名
func deleteServiceByID(tx *BaseTx, id string) error {
	log.Infof("[Store][database] delete service id(%s)", id)
	str := "update service set flag = 1, mtime = sysdate() where id = ?"
	if _, err := tx.Exec(str, id); err != nil {
		return err
	}

	return nil
}

// DeleteServiceAlias 删除服务别名
func (ss *serviceStore) DeleteServiceAlias(name string, namespace string) error {
	return ss.master.processWithTransaction("deleteServiceAlias", func(tx *BaseTx) error {
		str := "update service set flag = 1, mtime = sysdate() where name = ? and namespace = ?"
		if _, err := tx.Exec(str, name, namespace); err != nil {
			log.Errorf("[Store][database] delete service alias err: %s", err.Error())
			return store.Error(err)
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] batch delete service alias commit tx err: %s", err.Error())
			return err
		}
		return nil
	})
}

// UpdateServiceAlias 更新服务别名
func (ss *serviceStore) UpdateServiceAlias(alias *model.Service, needUpdateOwner bool) error {
	if alias.ID == "" ||
		alias.Name == "" ||
		alias.Namespace == "" ||
		alias.Revision == "" ||
		alias.Reference == "" ||
		(needUpdateOwner && alias.Owner == "") {
		return store.NewStatusError(store.EmptyParamsErr, "update Service Alias missing some params")
	}
	if err := ss.updateServiceAlias(alias, needUpdateOwner); err != nil {
		log.Errorf("[Store][ServiceAlias] update service alias err: %s", err.Error())
		return store.Error(err)
	}

	return nil
}

// updateServiceAlias update service alias
func (ss *serviceStore) updateServiceAlias(alias *model.Service, needUpdateOwner bool) error {
	tx, err := ss.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] update service alias tx begin err: %s", err.Error())
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	updateStmt := `
		update 
			service 
		set 
			name = ?, namespace = ?, reference = ?, comment = ?, token = ?, revision = ?, owner = ?, mtime = sysdate()
		where 
			id = ? and (select flag from (select flag from service where id = ?) as alias) = 0`

	result, err := tx.Exec(updateStmt, alias.Name, alias.Namespace, alias.Reference, alias.Comment, alias.Token,
		alias.Revision, alias.Owner, alias.ID, alias.Reference)
	if err != nil {
		log.Errorf("[Store][ServiceAlias] update service alias exec err: %s", err.Error())
		return err
	}

	// 更新owner_service_map表
	if needUpdateOwner {
		if err := updateOwnerServiceMap(tx, alias.Name, alias.Namespace, alias.Owner); err != nil {
			log.Errorf("[Store][database] update owner_service_map table err: %s", err.Error())
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update service alias tx commit err: %s", err.Error())
		return err
	}

	if err := checkServiceAffectedRows(result, 1); err != nil {
		if store.Code(err) == store.AffectedRowsNotMatch {
			return store.NewStatusError(store.NotFoundService, "not found service")
		}
	}
	return nil
}

// checkServiceAffectedRows 检查服务数据库处理返回的行数
func checkServiceAffectedRows(result sql.Result, count int64) error {
	n, err := result.RowsAffected()
	if err != nil {
		log.Errorf("[Store][ServiceAlias] get rows affected err: %s", err.Error())
		return err
	}

	if n == count {
		return nil
	}
	log.Errorf("[Store][ServiceAlias] get rows affected result(%d) is not match expect(%d)", n, count)
	return store.NewStatusError(store.AffectedRowsNotMatch, "affected rows not match")
}

// UpdateService 更新完整的服务信息
func (ss *serviceStore) UpdateService(service *model.Service, needUpdateOwner bool) error {
	if service.ID == "" ||
		service.Name == "" ||
		service.Namespace == "" ||
		service.Revision == "" {
		return store.NewStatusError(store.EmptyParamsErr, "Update Service missing some params")
	}

	err := RetryTransaction("updateService", func() error {
		return ss.updateService(service, needUpdateOwner)
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

// updateService update service
func (ss *serviceStore) updateService(service *model.Service, needUpdateOwner bool) error {
	tx, err := ss.master.Begin()
	if err != nil {
		log.Errorf("[Store][database] update service tx begin err: %s", err.Error())
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 更新main表
	if err := updateServiceMain(tx, service); err != nil {
		log.Errorf("[Store][database] update service main table err: %s", err.Error())
		return err
	}

	// 更新meta表
	if err := updateServiceMeta(tx, service.ID, service.Meta); err != nil {
		log.Errorf("[Store][database] update service meta table err: %s", err.Error())
		return err
	}

	// 更新owner_service_map表
	if needUpdateOwner {
		if err := updateOwnerServiceMap(tx, service.Name, service.Namespace, service.Owner); err != nil {
			log.Errorf("[Store][database] update owner_service_map table err: %s", err.Error())
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Errorf("[Store][database] update service tx commit err: %s", err.Error())
		return err
	}
	return nil
}

// UpdateServiceToken 更新服务token
func (ss *serviceStore) UpdateServiceToken(id string, token string, revision string) error {
	return ss.master.processWithTransaction("updateServiceToken", func(tx *BaseTx) error {
		str := `update service set token = ?, revision = ?, mtime = sysdate() where id = ?`
		_, err := tx.Exec(str, token, revision, id)
		if err != nil {
			log.Errorf("[Store][database] update service(%s) token err: %s", id, err.Error())
			return store.Error(err)
		}

		if err := tx.Commit(); err != nil {
			log.Errorf("[Store][database] update service token tx commit err: %s", err.Error())
			return err
		}

		return nil
	})
}

// GetService 获取服务详情，只返回有效的数据
func (ss *serviceStore) GetService(name string, namespace string) (*model.Service, error) {
	service, err := ss.getService(name, namespace)
	if err != nil {
		return nil, fmt.Errorf("getService err: %v", err)
	}

	if service != nil && !service.Valid {
		return nil, nil
	}

	return service, nil
}

// GetSourceServiceToken 获取只获取服务token
// 返回服务ID，服务token
func (ss *serviceStore) GetSourceServiceToken(name string, namespace string) (*model.Service, error) {
	str := `select id, token, IFNULL(platform_id, "") from service
			where name = ? and namespace = ? and flag = 0 
			and (reference is null or reference = '')`
	var out model.Service
	err := ss.master.QueryRow(str, name, namespace).Scan(&out.ID, &out.Token, &out.PlatformID)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, err
	default:
		out.Name = name
		out.Namespace = namespace
		return &out, nil
	}
}

// GetServiceByID 根据服务ID查询服务详情
func (ss *serviceStore) GetServiceByID(id string) (*model.Service, error) {
	service, err := ss.getServiceByID(id)
	if err != nil {
		return nil, err
	}
	if service != nil && !service.Valid {
		return nil, nil
	}

	return service, nil
}

// GetServices 根据相关条件查询对应服务及数目，不包括别名
func (ss *serviceStore) GetServices(serviceFilters, serviceMetas map[string]string, instanceFilters *store.InstanceArgs,
	offset, limit uint32) (uint32, []*model.Service, error) {
	// 只查询flag=0的服务列表
	serviceFilters["service.flag"] = "0"

	out, err := ss.getServices(serviceFilters, serviceMetas, instanceFilters, offset, limit)
	if err != nil {
		return 0, nil, err
	}

	num, err := ss.getServicesCount(serviceFilters, serviceMetas, instanceFilters)
	if err != nil {
		return 0, nil, err
	}
	return num, out, err
}

// GetServicesCount 获取所有服务总数
func (ss *serviceStore) GetServicesCount() (uint32, error) {
	countStr := "select count(*) from service where flag = 0"
	return queryEntryCount(ss.master, countStr, nil)
}

// GetMoreServices 根据modify_time获取增量数据
func (ss *serviceStore) GetMoreServices(mtime time.Time, firstUpdate, disableBusiness, needMeta bool) (
	map[string]*model.Service, error) {
	if needMeta {
		services, err := getMoreServiceWithMeta(ss.slave.Query, mtime, firstUpdate, disableBusiness)
		if err != nil {
			log.Errorf("[Store][database] get more service+meta err: %s", err.Error())
			return nil, err
		}
		return services, nil
	}
	services, err := getMoreServiceMain(ss.slave.Query, mtime, firstUpdate, disableBusiness)
	if err != nil {
		log.Errorf("[Store][database] get more service main err: %s", err.Error())
		return nil, err
	}
	return services, nil
}

// GetSystemServices 获取系统服务
func (ss *serviceStore) GetSystemServices() ([]*model.Service, error) {
	str := genServiceSelectSQL()
	str += " from service where flag = 0 and namespace = ?"
	rows, err := ss.master.Query(str, SystemNamespace)
	if err != nil {
		log.Errorf("[Store][database] get system service query err: %s", err.Error())
		return nil, err
	}

	out, err := ss.fetchRowServices(rows)
	if err != nil {
		log.Errorf("[Store][database] get row services err: %s", err)
		return nil, err
	}

	return out, nil
}

// GetServiceAliases 获取服务别名列表
func (ss *serviceStore) GetServiceAliases(filter map[string]string, offset uint32, limit uint32) (uint32,
	[]*model.ServiceAlias, error) {

	whereFilter := serviceAliasFilter2Where(filter)
	count, err := ss.getServiceAliasesCount(whereFilter)
	if err != nil {
		log.Errorf("[Store][database] get service aliases count err: %s", err.Error())
		return 0, nil, err
	}

	items, err := ss.getServiceAliasesInfo(whereFilter, offset, limit)
	if err != nil {
		log.Errorf("[Store][database] get service aliases info err: %s", err.Error())
		return 0, nil, err
	}

	return count, items, nil
}

// getServiceAliasesInfo 获取服务别名的详细信息
func (ss *serviceStore) getServiceAliasesInfo(filter map[string]string, offset uint32,
	limit uint32) ([]*model.ServiceAlias, error) {
	// limit为0，则直接返回
	if limit == 0 {
		return make([]*model.ServiceAlias, 0), nil
	}

	baseStr := `
		select 
			alias.id, alias.name, alias.namespace, UNIX_TIMESTAMP(alias.ctime), UNIX_TIMESTAMP(alias.mtime), 
			alias.comment, source.id as sourceID, source.name as sourceName, source.namespace, alias.owner 
		from 
			service as alias inner join service as source 
			on alias.reference = source.id and alias.flag != 1 `
	order := &Order{"alias.mtime", "desc"}

	queryStmt, args := genServiceAliasWhereSQLAndArgs(baseStr, filter, order, offset, limit)
	rows, err := ss.master.Query(queryStmt, args...)
	if err != nil {
		log.Errorf("[Store][database] get service aliases query(%s) err: %s", queryStmt, err.Error())
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*model.ServiceAlias
	var ctime, mtime int64
	for rows.Next() {
		var entry model.ServiceAlias
		err := rows.Scan(
			&entry.ID, &entry.Alias, &entry.AliasNamespace, &ctime, &mtime, &entry.Comment,
			&entry.ServiceID, &entry.Service, &entry.Namespace, &entry.Owner)
		if err != nil {
			log.Errorf("[Store][database] get service alias rows scan err: %s", err.Error())
			return nil, err
		}

		entry.CreateTime = time.Unix(ctime, 0)
		entry.ModifyTime = time.Unix(mtime, 0)
		out = append(out, &entry)
	}

	return out, nil
}

// getServiceAliasesCount 获取别名总数
func (ss *serviceStore) getServiceAliasesCount(filter map[string]string) (uint32, error) {
	baseStr := `
		select 
			count(*) 
		from 
			service as alias inner join service as source 
			on alias.reference = source.id and alias.flag != 1 `
	str, args := genServiceAliasWhereSQLAndArgs(baseStr, filter, nil, 0, 1)
	return queryEntryCount(ss.master, str, args)
}

// getServices 根据相关条件查询对应服务，不包括别名
func (ss *serviceStore) getServices(sFilters, sMetas map[string]string, iFilters *store.InstanceArgs,
	offset, limit uint32) ([]*model.Service, error) {
	// 不查询任意内容，直接返回空数组
	if limit == 0 {
		return make([]*model.Service, 0), nil
	}

	// 构造SQL语句
	var args []interface{}
	whereStr := " from service where (reference is null or reference = '') "
	if len(sMetas) > 0 {
		subStr, subArgs := filterMetadata(sMetas)
		whereStr += " and service.id in " + subStr
		args = append(args, subArgs...)
	}
	if iFilters != nil {
		subStr, subArgs := filterInstance(iFilters)
		whereStr += " and service.id in " + subStr
		args = append(args, subArgs...)
	}
	str := genServiceSelectSQL() + whereStr

	filterStr, filterArgs := genServiceFilterSQL(sFilters)
	if filterStr != "" {
		str += " and " + filterStr
		args = append(args, filterArgs...)
	}

	order := &Order{"service.mtime", "desc"}
	page := &Page{offset, limit}
	opStr, opArgs := genOrderAndPage(order, page)

	str += opStr
	args = append(args, opArgs...)
	rows, err := ss.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] get services by filter query(%s) err: %s", str, err.Error())
		return nil, err
	}

	out, err := ss.fetchRowServices(rows)
	if err != nil {
		log.Errorf("[Store][database] get row services err: %s", err)
		return nil, err
	}

	return out, nil
}

// getServicesCount 根据相关条件查询对应服务数目，不包括别名
func (ss *serviceStore) getServicesCount(
	sFilters, sMetas map[string]string, iFilters *store.InstanceArgs) (uint32, error) {
	str := `select count(*) from service  where (reference is null or reference = '')`
	var args []interface{}
	if len(sMetas) > 0 {
		subStr, subArgs := filterMetadata(sMetas)
		str += " and service.id in " + subStr
		args = append(args, subArgs...)
	}
	if iFilters != nil {
		subStr, subArgs := filterInstance(iFilters)
		str += " and service.id in " + subStr
		args = append(args, subArgs...)
	}

	filterStr, filterArgs := genServiceFilterSQL(sFilters)
	if filterStr != "" {
		str += " and " + filterStr
		args = append(args, filterArgs...)
	}
	return queryEntryCount(ss.master, str, args)
}

// fetchRowServices 根据rows，获取到services，并且批量获取对应的metadata
func (ss *serviceStore) fetchRowServices(rows *sql.Rows) ([]*model.Service, error) {
	services, err := fetchServiceRows(rows)
	if err != nil {
		return nil, err
	}

	data := make([]interface{}, 0, len(services))
	for idx := range services {
		// 只获取valid为true的metadata
		if services[idx].Valid {
			data = append(data, services[idx])
		}
	}

	err = BatchQuery("get-service-metadata", data, func(objects []interface{}) error {
		rows, batchErr := batchQueryServiceMeta(ss.master.Query, objects)
		if batchErr != nil {
			return batchErr
		}
		metas := make(map[string]map[string]string)
		batchErr = callFetchServiceMetaRows(rows, func(id, key, value string) (b bool, e error) {
			if _, ok := metas[id]; !ok {
				metas[id] = make(map[string]string)
			}
			metas[id][key] = value
			return true, nil
		})
		if batchErr != nil {
			return batchErr
		}
		for id, meta := range metas {
			for _, entry := range objects {
				if entry.(*model.Service).ID == id {
					entry.(*model.Service).Meta = meta
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("[Store][database] get service metadata err: %s", err.Error())
		return nil, err
	}

	return services, nil
}

// getServiceMeta 获取metadata数据
func (ss *serviceStore) getServiceMeta(id string) (map[string]string, error) {
	if id == "" {
		return nil, nil
	}

	// 从metadata表中获取数据
	metaStr := "select `mkey`, `mvalue` from service_metadata where id = ?"
	rows, err := ss.master.Query(metaStr, id)
	if err != nil {
		log.Errorf("[Store][database] get service metadata query err: %s", err.Error())
		return nil, err
	}
	return fetchServiceMeta(rows)
}

// getService 获取service内部函数
func (ss *serviceStore) getService(name string, namespace string) (*model.Service, error) {
	if name == "" || namespace == "" {
		return nil, fmt.Errorf("missing params, name: %s, namespace: %s", name, namespace)
	}

	out, err := ss.getServiceMain(name, namespace)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}

	meta, err := ss.getServiceMeta(out.ID)
	if err != nil {
		return nil, err
	}

	out.Meta = meta
	return out, nil
}

// getServiceMain 获取服务表的信息，不包括metadata
func (ss *serviceStore) getServiceMain(name string, namespace string) (*model.Service, error) {
	str := genServiceSelectSQL() + " from service where name = ? and namespace = ?"
	rows, err := ss.master.Query(str, name, namespace)
	if err != nil {
		log.Errorf("[Store][database] get service err: %s", err.Error())
		return nil, err
	}

	out, err := fetchServiceRows(rows)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out[0], nil
}

// getServiceByID 根据服务ID获取服务详情的内部函数
func (ss *serviceStore) getServiceByID(serviceID string) (*model.Service, error) {
	str := genServiceSelectSQL() + " from service where service.id = ?"
	rows, err := ss.master.Query(str, serviceID)
	if err != nil {
		log.Errorf("[Store][database] get service by id query err: %s", err.Error())
		return nil, err
	}

	out, err := fetchServiceRows(rows)
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	meta, err := ss.getServiceMeta(out[0].ID)
	if err != nil {
		return nil, err
	}
	out[0].Meta = meta

	return out[0], nil
}

// cleanService 清理无效数据，flag=1的数据，只需要删除service即可
func cleanService(tx *BaseTx, name string, namespace string) error {
	log.Infof("[Store][database] clean service(%s, %s)", name, namespace)
	str := "delete from service where name = ? and namespace = ? and flag = 1"
	_, err := tx.Exec(str, name, namespace)
	if err != nil {
		log.Errorf("[Store][database] clean service(%s, %s) err: %s", name, namespace, err.Error())
		return err
	}

	return nil
}

// getMoreServiceWithMeta获取增量服务,包括元数据
func getMoreServiceWithMeta(queryHandler QueryHandler, mtime time.Time, firstUpdate, disableBusiness bool) (
	map[string]*model.Service, error) {
	// 首次拉取
	if firstUpdate {
		// 获取全量服务
		services, err := getMoreServiceMain(queryHandler, mtime, firstUpdate, disableBusiness)
		if err != nil {
			log.Errorf("[Store][database] get more service main err: %s", err.Error())
			return nil, err
		}
		// 获取全量服务元数据
		str := "select id, mkey, mvalue from service_metadata"
		rows, err := queryHandler(str)
		if err != nil {
			log.Errorf("[Store][database] acquire services meta query err: %s", err.Error())
			return nil, err
		}
		if err := fetchMoreServiceMeta(services, rows); err != nil {
			return nil, err
		}
		return services, nil
	}

	// 非首次拉取
	var args []interface{}
	args = append(args, timeToTimestamp(mtime))
	str := genServiceSelectSQL() + `, IFNULL(service_metadata.id, ""), IFNULL(mkey, ""), IFNULL(mvalue, "") ` +
		`from service left join service_metadata on service.id = service_metadata.id where service.mtime >= FROM_UNIXTIME(?)`
	if disableBusiness {
		str += " and service.namespace = ?"
		args = append(args, SystemNamespace)
	}
	rows, err := queryHandler(str, args...)
	if err != nil {
		log.Errorf("[Store][database] get more services with meta query err: %s", err.Error())
		return nil, err
	}
	return fetchServiceWithMetaRows(rows)
}

// fetchServiceWithMetaRows 获取service+metadata rows里面的数据
func fetchServiceWithMetaRows(rows *sql.Rows) (map[string]*model.Service, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	out := make(map[string]*model.Service)
	var id, mKey, mValue string
	var flag int
	progress := 0
	for rows.Next() {
		progress++
		if progress%100000 == 0 {
			log.Infof("[Store][database] services+meta row next progress: %d", progress)
		}

		var item model.Service
		var exportTo string
		if err := rows.Scan(&item.ID, &item.Name, &item.Namespace, &item.Business, &item.Comment,
			&item.Token, &item.Revision, &item.Owner, &flag, &item.Ctime, &item.Mtime, &item.Ports,
			&item.Department, &item.CmdbMod1, &item.CmdbMod2, &item.CmdbMod3,
			&item.Reference, &item.ReferFilter, &item.PlatformID, &exportTo, &id, &mKey, &mValue); err != nil {
			log.Errorf("[Store][database] fetch service+meta rows scan err: %s", err.Error())
			return nil, err
		}
		item.CreateTime = time.Unix(item.Ctime, 0)
		item.ModifyTime = time.Unix(item.Mtime, 0)
		item.ExportTo = map[string]struct{}{}
		_ = json.Unmarshal([]byte(exportTo), &item.ExportTo)
		item.Valid = true
		if flag == 1 {
			item.Valid = false
		}

		if _, ok := out[item.ID]; !ok {
			out[item.ID] = &item
		}
		// 服务存在meta
		if id != "" {
			if out[item.ID].Meta == nil {
				out[item.ID].Meta = make(map[string]string)
			}
			out[item.ID].Meta[mKey] = mValue
		}
	}

	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch service+meta rows next err: %s", err.Error())
		return nil, err
	}
	return out, nil
}

// getMoreServiceMain get more service main
func getMoreServiceMain(queryHandler QueryHandler, mtime time.Time,
	firstUpdate, disableBusiness bool) (map[string]*model.Service, error) {
	var args []interface{}
	args = append(args, timeToTimestamp(mtime))
	str := genServiceSelectSQL() + " from service where service.mtime >= FROM_UNIXTIME(?)"
	if disableBusiness {
		str += " and service.namespace = ?"
		args = append(args, SystemNamespace)
	}
	if firstUpdate {
		str += " and flag != 1"
	}
	rows, err := queryHandler(str, args...)
	if err != nil {
		log.Errorf("[Store][database] get more services query err: %s", err.Error())
		return nil, err
	}
	out := make(map[string]*model.Service)
	err = callFetchServiceRows(rows, func(entry *model.Service) (b bool, e error) {
		out[entry.ID] = entry
		return true, nil
	})
	if err != nil {
		log.Errorf("[Store][database] call fetch service rows err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// batchQueryServiceMeta 批量查询service meta的封装
func batchQueryServiceMeta(handler QueryHandler, services []interface{}) (*sql.Rows, error) {
	if len(services) == 0 {
		return nil, nil
	}

	str := "select `id`, `mkey`, `mvalue` from service_metadata where id in("
	first := true
	args := make([]interface{}, 0, len(services))
	for _, ele := range services {
		if first {
			str += "?"
			first = false
		} else {
			str += ",?"
		}
		args = append(args, ele.(*model.Service).ID)
	}
	str += ")"

	rows, err := handler(str, args...)
	if err != nil {
		log.Errorf("[Store][database] batch query service meta err: %s", err.Error())
		return nil, err
	}

	return rows, nil
}

// fetchMoreServiceMeta fetch more service meta
func fetchMoreServiceMeta(services map[string]*model.Service, rows *sql.Rows) error {
	err := callFetchServiceMetaRows(rows, func(id, key, value string) (b bool, e error) {
		service, ok := services[id]
		if !ok {
			return true, nil
		}
		if service.Meta == nil {
			service.Meta = make(map[string]string)
		}
		service.Meta[key] = value
		return true, nil
	})
	if err != nil {
		log.Errorf("[Store][database] call fetch service meta rows err: %s", err.Error())
		return err
	}

	return nil
}

// callFetchServiceMetaRows 获取metadata的回调
func callFetchServiceMetaRows(rows *sql.Rows, handler func(id, key, value string) (bool, error)) error {
	if rows == nil {
		return nil
	}
	defer rows.Close()

	var id, key, value string
	progress := 0
	for rows.Next() {
		progress++
		if progress%100000 == 0 {
			log.Infof("[Store][database] fetch service meta rows progress: %d", progress)
		}
		if err := rows.Scan(&id, &key, &value); err != nil {
			log.Errorf("[Store][database] multi get service metadata scan err: %s", err.Error())
			return err
		}
		ok, err := handler(id, key, value)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

// addServiceMain 增加service主表数据
func addServiceMain(tx *BaseTx, s *model.Service) error {
	// 先把主表填充
	insertStmt := `
		insert into service
			(id, name, namespace, ports, business, department, cmdb_mod1, cmdb_mod2,
			cmdb_mod3, comment, token, reference,  platform_id, revision, owner, export_to, ctime, mtime)
		values
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, sysdate(), sysdate())`

	_, err := tx.Exec(insertStmt, s.ID, s.Name, s.Namespace, s.Ports, s.Business, s.Department,
		s.CmdbMod1, s.CmdbMod2, s.CmdbMod3, s.Comment, s.Token,
		s.Reference, s.PlatformID, s.Revision, s.Owner, utils.MustJson(s.ExportTo))
	return err
}

// addServiceMeta 增加服务metadata
func addServiceMeta(tx *BaseTx, id string, meta map[string]string) error {
	if len(meta) == 0 {
		return nil
	}
	str := "insert into service_metadata(id, `mkey`, `mvalue`, `ctime`, `mtime`) values "
	cnt := 0
	args := make([]interface{}, 0, len(meta)*3)
	for key, value := range meta {
		cnt++
		if cnt == len(meta) {
			str += "(?, ?, ?, sysdate(), sysdate())"
		} else {
			str += "(?, ?, ?, sysdate(), sysdate()),"
		}

		args = append(args, id)
		args = append(args, key)
		args = append(args, value)
	}

	// log.Infof("str: %s, args: %+v", str, args)
	_, err := tx.Exec(str, args...)
	return err
}

// updateServiceMain 更新service主表
func updateServiceMain(tx *BaseTx, service *model.Service) error {
	str := `update service set name = ?, namespace = ?, ports = ?, business = ?,
	department = ?, cmdb_mod1 = ?, cmdb_mod2 = ?, cmdb_mod3 = ?, comment = ?, token = ?, platform_id = ?,
	revision = ?, owner = ?, mtime = sysdate(), export_to = ? where id = ?`

	_, err := tx.Exec(str, service.Name, service.Namespace, service.Ports, service.Business,
		service.Department, service.CmdbMod1, service.CmdbMod2, service.CmdbMod3,
		service.Comment, service.Token, service.PlatformID, service.Revision, service.Owner,
		utils.MustJson(service.ExportTo), service.ID)
	return err
}

// updateServiceMeta 更新service meta表
func updateServiceMeta(tx *BaseTx, id string, meta map[string]string) error {
	// 只有metadata为nil的时候，则不用处理。
	// 如果metadata不为nil，但是len(metadata) == 0，则代表删除metadata
	if meta == nil {
		return nil
	}

	str := "delete from service_metadata where id = ?"
	if _, err := tx.Exec(str, id); err != nil {
		return err
	}

	return addServiceMeta(tx, id, meta)
}

// fetchServiceMeta 从rows里面获取metadata数据
func fetchServiceMeta(rows *sql.Rows) (map[string]string, error) {
	if rows == nil {
		return nil, nil
	}
	defer rows.Close()

	out := make(map[string]string)
	var key, value string
	for rows.Next() {

		if err := rows.Scan(&key, &value); err != nil {
			log.Errorf("[Store][database] fetch service meta err: %s", err.Error())
			return nil, err
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] service metadata rows err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// genServiceSelectSQL 生成service查询语句
func genServiceSelectSQL() string {
	return `select service.id, name, namespace, IFNULL(business, ""), IFNULL(comment, ""),
			token, service.revision, owner, service.flag, 
			UNIX_TIMESTAMP(service.ctime), UNIX_TIMESTAMP(service.mtime),
			IFNULL(ports, ""), IFNULL(department, ""), IFNULL(cmdb_mod1, ""), IFNULL(cmdb_mod2, ""), 
			IFNULL(cmdb_mod3, ""), IFNULL(reference, ""), IFNULL(refer_filter, ""), IFNULL(platform_id, ""),
			IFNULL(export_to, "{}") `
}

// callFetchServiceRows call fetch service rows
func callFetchServiceRows(rows *sql.Rows, callback func(entry *model.Service) (bool, error)) error {
	if rows == nil {
		return nil
	}
	defer rows.Close()

	var ctime, mtime int64
	var flag int
	progress := 0
	for rows.Next() {
		progress++
		if progress%100000 == 0 {
			log.Infof("[Store][database] services row next progress: %d", progress)
		}

		var item model.Service
		var exportTo string
		err := rows.Scan(
			&item.ID, &item.Name, &item.Namespace, &item.Business, &item.Comment,
			&item.Token, &item.Revision, &item.Owner, &flag, &ctime, &mtime, &item.Ports,
			&item.Department, &item.CmdbMod1, &item.CmdbMod2, &item.CmdbMod3,
			&item.Reference, &item.ReferFilter, &item.PlatformID, &exportTo)

		if err != nil {
			log.Errorf("[Store][database] fetch service rows scan err: %s", err.Error())
			return err
		}

		item.CreateTime = time.Unix(ctime, 0)
		item.ModifyTime = time.Unix(mtime, 0)
		item.ExportTo = map[string]struct{}{}
		_ = json.Unmarshal([]byte(exportTo), &item.ExportTo)
		item.Valid = true
		if flag == 1 {
			item.Valid = false
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
		log.Errorf("[Store][database] fetch service rows next err: %s", err.Error())
		return err
	}
	return nil
}

// fetchServiceRows 获取service rows里面的数据
func fetchServiceRows(rows *sql.Rows) ([]*model.Service, error) {
	var out []*model.Service
	err := callFetchServiceRows(rows, func(entry *model.Service) (b bool, e error) {
		out = append(out, entry)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// filterInstance 查找service，根据instance属性过滤
// 生成子查询语句
func filterInstance(filters *store.InstanceArgs) (string, []interface{}) {
	var args []interface{}
	str := "(select service_id from instance where instance.flag != 1 and host in (" +
		PlaceholdersN(len(filters.Hosts)) + ")"
	if len(filters.Ports) > 0 {
		str += " and port in (" + PlaceholdersN(len(filters.Ports)) + ")"
	}
	str += " group by service_id)"
	for _, host := range filters.Hosts {
		args = append(args, host)
	}
	for _, port := range filters.Ports {
		args = append(args, port)
	}
	return str, args
}

// filterMetadata 查找service，根据metadata属性过滤
// 生成子查询语句
// 多个metadata，取交集（and）
func filterMetadata(metas map[string]string) (string, []interface{}) {
	str := "(select id from service_metadata where mkey = ? and mvalue = ?)"
	args := make([]interface{}, 0, 2)
	for key, value := range metas {
		args = append(args, key)
		args = append(args, value)
	}

	return str, args
}

// GetServicesBatch 查询多个服务的id
func (ss *serviceStore) GetServicesBatch(services []*model.Service) ([]*model.Service, error) {
	if len(services) == 0 {
		return nil, nil
	}
	str := `select id, name, namespace,owner from service where flag = 0 and (name, namespace) in (`
	args := make([]interface{}, 0, len(services)*2)
	for key, value := range services {
		str += "(" + PlaceholdersN(2) + ")"
		if key != len(services)-1 {
			str += ","
		}
		args = append(args, value.Name)
		args = append(args, value.Namespace)
	}
	str += `)`

	rows, err := ss.master.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] query services batch err: %s", err.Error())
		return nil, err
	}

	res := make([]*model.Service, 0, len(services))
	var namespace, name, id, owner string
	for rows.Next() {
		err := rows.Scan(&id, &name, &namespace, &owner)
		if err != nil {
			log.Errorf("[Store][database] fetch services batch scan err: %s", err.Error())
			return nil, err
		}
		res = append(res, &model.Service{
			ID:        id,
			Name:      name,
			Namespace: namespace,
			Owner:     owner,
		})
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] fetch services batch next err: %s", err.Error())
		return nil, err
	}
	return res, nil
}

// addOwnerServiceMap 填充owner_service_map表
func addOwnerServiceMap(tx *BaseTx, service, namespace, owner string) error {
	addSql := "insert into owner_service_map(id,owner,service,namespace) values"

	// 根据; ,进行分割
	owners := strings.FieldsFunc(owner, func(r rune) bool {
		return r == ';' || r == ','
	})
	args := make([]interface{}, 0)

	if len(owners) >= 1 {
		for i := 0; i < len(owners); i++ {
			addSql += "(?,?,?,?),"
			args = append(args, utils.NewUUID(), owners[i], service, namespace)
		}
		if len(args) != 0 {
			addSql = strings.TrimSuffix(addSql, ",")
			if _, err := tx.Exec(addSql, args...); err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteOwnerServiceMap 删除owner_service_map表中对应记录
func deleteOwnerServiceMap(tx *BaseTx, service, namespace string) error {
	log.Infof("[Store][database] delete service(%s) namespace(%s)", service, namespace)
	delSql := "delete from owner_service_map where service=? and namespace=?"
	if _, err := tx.Exec(delSql, service, namespace); err != nil {
		return err
	}

	return nil
}

// deleteServiceMetadata 删除 service_metadata 中的残留数据
func deleteServiceMetadata(tx *BaseTx, service, namespace string) error {
	log.Infof("[Store][database] delete service(%s) namespace(%s)", service, namespace)
	delSql := "delete from service_metadata where id IN (select id from service where name = ? and namespace = ?)"
	if _, err := tx.Exec(delSql, service, namespace); err != nil {
		return err
	}

	return nil
}

// updateOwnerServiceMap owner_service_map表，先删除，后填充
func updateOwnerServiceMap(tx *BaseTx, service, namespace, owner string) error {
	// 删除
	if err := deleteOwnerServiceMap(tx, service, namespace); err != nil {
		return err
	}
	// 填充
	if err := addOwnerServiceMap(tx, service, namespace, owner); err != nil {
		return err
	}

	return nil
}
