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
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func Test_toolStore_GetUnixSecond(t *testing.T) {
	t.Run("正常场景", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"UNIX_TIMESTAMP(SYSDATE())"})
		rows.AddRow(1)
		mock.ExpectQuery(nowSql).WillDelayFor(2 * time.Second).WillReturnRows(rows)

		tr := &toolStore{
			db: &BaseDB{DB: db},
		}
		got, err := tr.GetUnixSecond(0)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), int64(got))
	})

	t.Run("SQL执行时间超过MaxWait", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"UNIX_TIMESTAMP(SYSDATE())"})
		rows.AddRow(100)
		mock.ExpectQuery(nowSql).WillDelayFor(2 * time.Second).WillReturnRows(rows)

		tr := &toolStore{
			db: &BaseDB{DB: db},
		}
		got, err := tr.GetUnixSecond(time.Second)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), int64(got))
	})

	t.Run("SQL查询出错", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		mock.ExpectQuery(nowSql).WillReturnError(errors.New("mock error"))

		tr := &toolStore{
			db: &BaseDB{DB: db},
		}
		got, err := tr.GetUnixSecond(time.Second)
		assert.Error(t, err)
		assert.Equal(t, int64(0), int64(got))
	})

	t.Run("SQL返回的不是int", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		rows := sqlmock.NewRows([]string{"UNIX_TIMESTAMP(SYSDATE())"})
		rows.AddRow("100qer")
		mock.ExpectQuery(nowSql).WillDelayFor(0).WillReturnRows(rows)

		tr := &toolStore{
			db: &BaseDB{DB: db},
		}
		got, err := tr.GetUnixSecond(time.Second)
		assert.Error(t, err)
		assert.Equal(t, int64(0), int64(got))
	})
}
