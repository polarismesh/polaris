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
	"crypto/rand"
	"math/big"

	"github.com/pkg/errors"

	"github.com/polarismesh/polaris/common/model"
)

// transaction 事务; 不支持多协程并发操作，当前先支持单个协程串行操作
type transaction struct {
	tx     *BaseTx
	failed bool // 判断事务执行是否失败
	commit bool // 判断事务已经提交，如果已经提交，则Commit会立即返回
}

// Commit 提交事务，释放tx
func (t *transaction) Commit() error {
	if t.commit {
		return nil
	}

	t.commit = true
	if t.failed {
		return t.tx.Rollback()
	}

	return t.tx.Commit()
}

// LockBootstrap 启动锁，限制Server启动的并发数
func (t *transaction) LockBootstrap(key string, server string) error {
	countStr := "select count(*) from start_lock where lock_key = ?"
	var count int
	if err := t.tx.QueryRow(countStr, key).Scan(&count); err != nil {
		log.Errorf("[Store][database] lock bootstrap scan count err: %s", err.Error())
		t.failed = true
		return err
	}

	bid, err := rand.Int(rand.Reader, big.NewInt(1024))
	if err != nil {
		log.Errorf("[Store][database] rand int err: %s", err.Error())
		return err
	}

	log.Infof("[Store][database] get rand int: %d", bid.Int64())
	id := int(bid.Int64())%count + 1
	// innodb_lock_wait_timeout这个global变量表示锁超时的时间，cdb为7200秒
	log.Infof("[Store][database] update start lock_id: %d, lock_key: %s, lock server: %s", id, key, server)
	lockStr := "update start_lock set server = ? where lock_id = ? and lock_key = ?"
	if _, err := t.tx.Exec(lockStr, server, id, key); err != nil {
		log.Errorf("[Store][database] update start lock err: %s", err.Error())
		t.failed = true
		return err
	}

	return nil
}

// LockNamespace 排它锁，锁住指定命名空间
func (t *transaction) LockNamespace(name string) (*model.Namespace, error) {
	str := genNamespaceSelectSQL() + " where name = ? and flag != 1 for update"
	return t.getValidNamespace(str, name)
}

// RLockNamespace 共享锁，锁住命名空间
func (t *transaction) RLockNamespace(name string) (*model.Namespace, error) {
	str := genNamespaceSelectSQL() + " where name = ? and flag != 1 lock in share mode"
	return t.getValidNamespace(str, name)
}

// DeleteNamespace 删除命名空间，并且提交事务
func (t *transaction) DeleteNamespace(name string) error {
	if err := t.finish(); err != nil {
		return err
	}

	str := "update namespace set flag = 1, mtime = sysdate() where name = ?"
	if _, err := t.tx.Exec(str, name); err != nil {
		t.failed = true
	}

	return t.Commit()
}

// LockService 排它锁，锁住指定服务
func (t *transaction) LockService(name string, namespace string) (*model.Service, error) {
	str := genServiceSelectSQL() +
		" from service where name = ? and namespace = ? and flag !=1 for update"
	return t.getValidService(str, name, namespace)
}

// RLockService 共享锁，锁住指定服务
func (t *transaction) RLockService(name string, namespace string) (*model.Service, error) {
	str := genServiceSelectSQL() +
		" from service where name = ? and namespace = ? and flag !=1 lock in share mode"
	return t.getValidService(str, name, namespace)
}

// BatchRLockServices 批量锁住服务
func (t *transaction) BatchRLockServices(ids map[string]bool) (map[string]bool, error) {
	str := "select id, flag from service where id in ( "
	first := true
	args := make([]interface{}, 0, len(ids))
	for id := range ids {
		if first {
			str += "?"
			first = false
		} else {
			str += ", ?"
		}
		args = append(args, id)
	}
	str += ") and flag != 1 lock in share mode"
	log.Infof("[Store][database] RLock services: %+v", args)
	rows, err := t.tx.Query(str, args...)
	if err != nil {
		log.Errorf("[Store][database] batch RLock services err: %s", err.Error())
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	var flag int
	var id string
	for rows.Next() {
		if err := rows.Scan(&id, &flag); err != nil {
			log.Errorf("[Store][database] RLock services scan err: %s", err.Error())
			return nil, err
		}

		if flag == 0 {
			out[id] = true
		} else {
			out[id] = false
		}
	}
	if err := rows.Err(); err != nil {
		log.Errorf("[Store][database] RLock service rows next err: %s", err.Error())
		return nil, err
	}

	return out, nil
}

// DeleteService 删除服务，并且提交事务
func (t *transaction) DeleteService(name string, namespace string) error {
	if err := t.finish(); err != nil {
		return err
	}

	str := "update service set flag = 1, mtime = sysdate() where name = ? and namespace = ?"
	if _, err := t.tx.Exec(str, name, namespace); err != nil {
		log.Errorf("[Store][database] delete service err: %s", err.Error())
		t.failed = true
		return err
	}

	return nil
}

// DeleteAliasWithSourceID 根据源服务的ID，删除其所有的别名
func (t *transaction) DeleteAliasWithSourceID(sourceServiceID string) error {
	if err := t.finish(); err != nil {
		return err
	}

	str := `update service set flag = 1, mtime = sysdate() where reference = ?`
	if _, err := t.tx.Exec(str, sourceServiceID); err != nil {
		log.Errorf("[Store][database] delete service alias err: %s", err.Error())
		t.failed = false
		return err
	}

	return nil
}

// finish 判断事务是否已经提交
func (t *transaction) finish() error {
	if t.failed || t.commit {
		return errors.New("transaction has failed")
	}

	return nil
}

// getValidNamespace 获取有效的命名空间数据
func (t *transaction) getValidNamespace(sql string, name string) (*model.Namespace, error) {
	if err := t.finish(); err != nil {
		return nil, err
	}

	rows, err := t.tx.Query(sql, name)
	if err != nil {
		t.failed = true
		return nil, err
	}

	out, err := namespaceFetchRows(rows)
	if err != nil {
		t.failed = true
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out[0], nil
}

// getValidService 获取有效的服务数据
// 注意：该函数不会返回service_metadata
func (t *transaction) getValidService(sql string, name string, namespace string) (*model.Service, error) {
	if err := t.finish(); err != nil {
		return nil, err
	}

	rows, err := t.tx.Query(sql, name, namespace)
	if err != nil {
		t.failed = true
		return nil, err
	}

	out, err := fetchServiceRows(rows)
	if err != nil {
		t.failed = true
		return nil, err
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out[0], nil
}
