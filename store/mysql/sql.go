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
	"fmt"
	"strconv"
	"strings"

	"github.com/polarismesh/polaris/common/utils"
)

const (

	// OwnerAttribute
	OwnerAttribute string = "owner"

	// And
	And string = " and"
)

// Order 排序结构体
type Order struct {
	Filed    string
	Sequence string
}

// Page 分页结构体
type Page struct {
	Offset uint32
	Limit  uint32
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// genFilterSQL 根据filter生成where相关的语句
func genFilterSQL(filter map[string]string) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	args := make([]interface{}, 0, len(filter))
	var str string
	firstIndex := true
	for key, value := range filter {
		if !firstIndex {
			str += And
		}
		firstIndex = false
		// 这个查询组装，先这样完成，后续优化filter TODO
		if key == OwnerAttribute || key == "alias."+OwnerAttribute || key == "business" {
			str += fmt.Sprintf(" %s like ?", key)
			value = "%" + value + "%"
		} else if key == "name" && utils.IsWildName(value) {
			str += " name like ?"
			value = utils.ParseWildNameForSql(value)
		} else if key == "id" {
			if utils.IsWildName(value) {
				str += " instance.id like ?"
				value = utils.ParseWildNameForSql(value)
			} else {
				str += " instance.id = ?"
			}
		} else if key == "host" {
			hosts := strings.Split(value, ",")
			str += " host in (" + PlaceholdersN(len(hosts)) + ")"
			for _, host := range hosts {
				args = append(args, host)
			}
		} else if key == "managed" {
			str += " managed = ?"
			managed, _ := strconv.ParseBool(value)
			args = append(args, boolToInt(managed))
			continue
		} else if key == "namespace" && utils.IsWildName(value) {
			str += " namespace like ?"
			value = utils.ParseWildNameForSql(value)
		} else {
			str += " " + key + "=?"
		}
		if key != "host" {
			args = append(args, value)
		}
	}

	return str, args
}

// genServiceFilterSQL 根据service filter生成where相关的语句
func genServiceFilterSQL(filter map[string]string) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	args := make([]interface{}, 0, len(filter))
	var str string
	firstIndex := true
	for key, value := range filter {
		if !firstIndex {
			str += And
		}
		firstIndex = false

		if key == OwnerAttribute {
			str += " (service.name, service.namespace) in (select service,namespace from owner_service_map where owner=?)"
		} else if key == "alias."+OwnerAttribute {
			str += " (alias.name, alias.namespace) in (select service,namespace from owner_service_map where owner=?)"
		} else if key == "business" {
			str += fmt.Sprintf(" %s like ?", key)
			value = "%" + value + "%"
		} else if key == "name" && utils.IsPrefixWildName(value) {
			str += " name like ?"
			value = "%" + value[0:len(value)-1] + "%"
		} else {
			str += " " + key + "=?"
		}

		args = append(args, value)
	}

	return str, args
}

// genRuleFilterSQL 根据规则的filter生成where相关的语句
func genRuleFilterSQL(tableName string, filter map[string]string) (string, []interface{}) {
	if len(filter) == 0 {
		return "", nil
	}

	args := make([]interface{}, 0, len(filter))
	var str string
	firstIndex := true
	for key, value := range filter {
		if tableName != "" {
			key = tableName + "." + key
		}
		if !firstIndex {
			str += And
		}
		if key == OwnerAttribute || key == (tableName+"."+OwnerAttribute) {
			str += fmt.Sprintf(" %s like ? ", key)
			value = "%" + value + "%"
		} else {
			str += " " + key + " = ? "
		}
		args = append(args, value)
		firstIndex = false
	}
	return str, args
}

// genOrderAndPage 生成order和page相关语句
func genOrderAndPage(order *Order, page *Page) (string, []interface{}) {
	var str string
	var args []interface{}
	if order != nil {
		str += " order by " + order.Filed + " " + order.Sequence
	}
	if page != nil {
		str += " limit ?, ?"
		args = append(args, page.Offset, page.Limit)
	}

	return str, args
}

// genWhereSQLAndArgs 生成service和instance查询数据的where语句和对应参数
func genWhereSQLAndArgs(str string, filter, metaFilter map[string]string, order *Order, offset uint32, limit uint32) (
	string, []interface{}) {
	baseStr := str
	var args []interface{}
	filterStr, filterArgs := genFilterSQL(filter)
	var conjunction string = " where "
	if filterStr != "" {
		baseStr += " where " + filterStr
		conjunction = " and "
	}
	args = append(args, filterArgs...)
	var metaStr string
	var metaArgs []interface{}
	if len(metaFilter) > 0 {
		metaStr, metaArgs = genInstanceMetadataArgs(metaFilter)
		args = append(args, metaArgs...)
		baseStr += conjunction + metaStr
	}
	page := &Page{offset, limit}
	opStr, opArgs := genOrderAndPage(order, page)

	return baseStr + opStr, append(args, opArgs...)
}

func genInstanceMetadataArgs(metaFilter map[string]string) (string, []interface{}) {
	str := `instance.id in (select id from instance_metadata where mkey = ? and mvalue = ?)`
	args := make([]interface{}, 0, 2)
	for k, v := range metaFilter {
		args = append(args, k)
		args = append(args, v)
	}
	return str, args
}

// genServiceAliasWhereSQLAndArgs 生成service alias查询数据的where语句和对应参数
func genServiceAliasWhereSQLAndArgs(str string, filter map[string]string, order *Order, offset uint32, limit uint32) (
	string, []interface{}) {
	baseStr := str
	filterStr, filterArgs := genServiceFilterSQL(filter)
	if filterStr != "" {
		baseStr += " where "
	}
	page := &Page{offset, limit}
	opStr, opArgs := genOrderAndPage(order, page)

	return baseStr + filterStr + opStr, append(filterArgs, opArgs...)
}

// genNamespaceWhereSQLAndArgs 生成namespace查询数据的where语句和对应参数
func genNamespaceWhereSQLAndArgs(str string, filter map[string][]string, order *Order, offset, limit int) (
	string, []interface{}) {
	num := 0
	for _, value := range filter {
		num += len(value)
	}
	args := make([]interface{}, 0, num+2)

	if num > 0 {
		str += "where"
		firstIndex := true

		for index, value := range filter {
			if !firstIndex {
				str += And
			}
			str += " ("

			firstItem := true
			for _, item := range value {
				if !firstItem {
					str += " or "
				}
				if index == OwnerAttribute {
					str += "owner like ?"
					item = "%" + item + "%"
				} else {
					if index == NameAttribute && utils.IsWildName(item) {
						str += "name like ?"
						item = utils.ParseWildNameForSql(item)
					} else {
						str += index + "=?"
					}
				}
				args = append(args, item)
				firstItem = false
			}
			firstIndex = false
			str += ")"
		}
	}

	if order != nil {
		str += " order by " + order.Filed + " " + order.Sequence
	}

	str += " limit ?, ?"
	args = append(args, offset, limit)

	return str, args
}

// filterMetadataWithTable 根据metadata属性过滤
// 生成子查询语句
// 多个metadata，取交集（and）
func filterMetadataWithTable(table string, metas map[string]string) (string, []interface{}) {
	str := "(select id from " + table + " where mkey = ? and mvalue = ?)"
	args := make([]interface{}, 0, 2)
	for key, value := range metas {
		args = append(args, key)
		args = append(args, value)
	}

	return str, args
}

// PlaceholdersN 构造多个占位符
func PlaceholdersN(size int) string {
	if size <= 0 {
		return ""
	}
	str := strings.Repeat("?,", size)
	return str[0 : len(str)-1]
}
