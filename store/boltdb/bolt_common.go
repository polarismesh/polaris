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
	"go.uber.org/zap"

	"github.com/polarismesh/polaris/store"
)

type transactionFunc func(tx *bolt.Tx) ([]interface{}, error)

func DoTransactionIfNeed(sTx store.Tx, handler BoltHandler, handle transactionFunc) ([]interface{}, error) {
	var (
		err          error
		autoManageTx bool
	)
	autoManageTx = (sTx == nil)
	if sTx == nil {
		sTx, err = handler.StartTx()
		if err != nil {
			return nil, err
		}
	}
	tx := sTx.GetDelegateTx().(*bolt.Tx)
	defer func() {
		if autoManageTx {
			_ = tx.Rollback()
		}
	}()

	ret, err := handle(tx)

	if autoManageTx && err == nil {
		if err := tx.Commit(); err != nil {
			log.Error("do tx commit", zap.Error(err))
			return nil, err
		}
	}

	return ret, err
}
