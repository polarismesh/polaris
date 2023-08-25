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
	bolt "go.etcd.io/bbolt"

	"github.com/polarismesh/polaris/store"
)

type Tx struct {
	delegateTx *bolt.Tx
}

func NewBoltTx(delegateTx *bolt.Tx) store.Tx {
	return &Tx{
		delegateTx: delegateTx,
	}
}

func (t *Tx) Commit() error {
	return t.delegateTx.Commit()
}

func (t *Tx) Rollback() error {
	return t.delegateTx.Rollback()
}

func (t *Tx) GetDelegateTx() interface{} {
	return t.delegateTx
}

func (t *Tx) CreateReadView() error {
	return nil
}
