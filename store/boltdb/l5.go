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
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/common/model"
)

type l5Store struct {
	handler BoltHandler
}

// GetL5Extend 获取扩展数据
func (l *l5Store) GetL5Extend(serviceID string) (map[string]interface{}, error) {
	return nil, nil
}

// SetL5Extend 设置meta里保存的扩展数据，并返回剩余的meta
func (l *l5Store) SetL5Extend(serviceID string, meta map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

// InitL5Data 初始化L5数据
func (l *l5Store) InitL5Data() error {
	return l.handler.Execute(true, func(tx *bolt.Tx) error {
		var err error
		var tblBucket *bolt.Bucket
		tblBucket, err = tx.CreateBucketIfNotExists([]byte(tblNameL5))
		if err != nil {
			return err
		}
		rowBucket := tblBucket.Bucket([]byte(rowSidKey))
		if rowBucket != nil {
			// 数据已存在，不做处理
			return nil
		}
		rowBucket, err = tblBucket.CreateBucket([]byte(rowSidKey))
		if err != nil {
			return err
		}
		return updateL5SidTable(rowBucket, 3000001, 1, 0)
	})
}

const (
	tblNameL5      = "l5"
	rowSidKey      = "sidSequence"
	colModuleId    = "module_id"
	colInterfaceId = "interface_id"
	colRangeNum    = "range_num"
)

func updateL5SidTable(rowBucket *bolt.Bucket, mid uint64, iid uint64, rnum uint64) error {
	var err error
	if err = rowBucket.Put([]byte(colModuleId), encodeUintBuffer(mid, typeUint32)); err != nil {
		return err
	}
	if err = rowBucket.Put([]byte(colInterfaceId), encodeUintBuffer(iid, typeUint32)); err != nil {
		return err
	}
	if err = rowBucket.Put([]byte(colRangeNum), encodeUintBuffer(rnum, typeUint32)); err != nil {
		return err
	}
	return nil
}

// GenNextL5Sid 获取module
func (l *l5Store) GenNextL5Sid(layoutID uint32) (string, error) {
	var pmid *uint64
	var piid *uint64
	var prnum *uint64
	err := l.handler.Execute(true, func(tx *bolt.Tx) error {
		tblBucket := tx.Bucket([]byte(tblNameL5))
		if tblBucket == nil {
			return fmt.Errorf("[BlobStore] table bucket %s not exists", tblNameL5)
		}
		rowBucket := tblBucket.Bucket([]byte(rowSidKey))
		if rowBucket == nil {
			return fmt.Errorf("[BlobStore] row bucket %s not exists", rowSidKey)
		}
		midBytes := rowBucket.Get([]byte(colModuleId))
		mid, err := decodeUintBuffer(colModuleId, midBytes, typeUint32)
		if err != nil {
			return err
		}
		iidBytes := rowBucket.Get([]byte(colInterfaceId))
		iid, err := decodeUintBuffer(colInterfaceId, iidBytes, typeUint32)
		if err != nil {
			return err
		}
		rnumBytes := rowBucket.Get([]byte(colRangeNum))
		rnum, err := decodeUintBuffer(colRangeNum, rnumBytes, typeUint32)
		if err != nil {
			return err
		}
		rnum++
		if rnum >= 65536 {
			rnum = 0
			iid++
		}
		if iid >= 4096 {
			iid = 1
			mid++
		}
		err = updateL5SidTable(rowBucket, mid, iid, rnum)
		if err != nil {
			return err
		}
		pmid = &mid
		piid = &iid
		prnum = &rnum
		return nil
	})
	if err != nil {
		return "", err
	}
	modID := uint32(*pmid)<<6 + layoutID
	cmdID := uint32(*piid)<<16 + uint32(*prnum)
	return fmt.Sprintf("%d:%d", modID, cmdID), nil
}

// GetMoreL5Extend 获取增量数据
func (l *l5Store) GetMoreL5Extend(mtime time.Time) (map[string]map[string]interface{}, error) {
	return nil, nil
}

// GetMoreL5Routes 获取Route增量数据
func (l *l5Store) GetMoreL5Routes(flow uint32) ([]*model.Route, error) {
	return nil, nil
}

// GetMoreL5Policies 获取Policy增量数据
func (l *l5Store) GetMoreL5Policies(flow uint32) ([]*model.Policy, error) {
	return nil, nil
}

// GetMoreL5Sections 获取Section增量数据
func (l *l5Store) GetMoreL5Sections(flow uint32) ([]*model.Section, error) {
	return nil, nil
}

// GetMoreL5IPConfigs 获取IP Config增量数据
func (l *l5Store) GetMoreL5IPConfigs(flow uint32) ([]*model.IPConfig, error) {
	return nil, nil
}
