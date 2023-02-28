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

package main

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	apifault "github.com/polarismesh/specification/source/go/api/v1/fault_tolerance"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	EnvDbAddr = "env_db_addr"
	EnvDbName = "env_db_name"
	EnvDbUser = "env_db_user"
	EnvDbPwd  = "env_db_pwd"
)

var (
	dbAddr string
	dbName string
	dbUser string
	dbPwd  string
)

func init() {
	flag.StringVar(&dbAddr, "db_addr", "", "Input database address")
	flag.StringVar(&dbName, "db_name", "", "Input database name")
	flag.StringVar(&dbUser, "db_user", "", "Input database user")
	flag.StringVar(&dbPwd, "db_pwd", "", "Input database password")
}

const (
	queryCbSql = `select rule.id, rule.version, rule.name, rule.namespace, service.name, IFNULL(rule.business, ""),
			IFNULL(rule.comment, ""), IFNULL(rule.department, ""),
			rule.inbounds, rule.outbounds, rule.owner, rule.revision,
			unix_timestamp(rule.ctime), unix_timestamp(rule.mtime) 
			from circuitbreaker_rule as rule, circuitbreaker_rule_relation as relation, service 
			where service.id = relation.service_id and relation.rule_id = rule.id 
			and relation.rule_version = rule.version 
			and relation.flag = 0 and service.flag = 0 and rule.flag = 0`
	insertCircuitBreakerRuleSql = `insert into circuitbreaker_rule_v2(
			id, name, namespace, enable, revision, description, level, src_service, src_namespace, 
			dst_service, dst_namespace, dst_method, config, ctime, mtime, etime)
			values(?,?,?,?,?,?,?,?,?,?,?,?,?, sysdate(),sysdate(), %s)`
)

type CircuitBreakerData struct {
	ID         string
	Version    string
	Name       string
	Namespace  string
	Service    string
	Business   string
	Department string
	Comment    string
	Inbounds   string
	Outbounds  string
	Token      string
	Owner      string
	Revision   string
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

type NewCircuitBreakerData struct {
	ID           string
	Name         string
	Namespace    string
	Description  string
	Level        int
	SrcService   string
	SrcNamespace string
	DstService   string
	DstNamespace string
	DstMethod    string
	Rule         string
	Revision     string
	Enable       bool
	Valid        bool
	CreateTime   time.Time
	ModifyTime   time.Time
	EnableTime   time.Time
}

func queryOldRule(db *sql.DB) ([]*CircuitBreakerData, error) {
	rows, err := db.Query(queryCbSql)
	if nil != err {
		return nil, err
	}
	defer rows.Close()
	var out []*CircuitBreakerData
	for rows.Next() {
		var breaker CircuitBreakerData
		var ctime, mtime int64
		err := rows.Scan(&breaker.ID, &breaker.Version, &breaker.Name, &breaker.Namespace,
			&breaker.Service, &breaker.Business, &breaker.Comment, &breaker.Department,
			&breaker.Inbounds, &breaker.Outbounds, &breaker.Owner, &breaker.Revision, &ctime, &mtime)
		if err != nil {
			log.Printf("fetch brief circuitbreaker rule scan err: %s", err.Error())
			break
		}
		breaker.CreateTime = time.Unix(ctime, 0)
		breaker.ModifyTime = time.Unix(mtime, 0)
		log.Printf("old circuitbreaker rule query is %v", breaker)
		out = append(out, &breaker)
	}

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		log.Printf("get tag circuitbreaker err: %s", err.Error())
		return nil, err
	default:
		return out, nil
	}
}

// Time2String Convert time.Time to string time
func Time2String(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func toCircuitBreakerRule(svc string,
	oldRuleProto *apifault.CircuitBreaker, subRule *apifault.CbRule, inbound bool) *apifault.CircuitBreakerRule {
	cbRule := &apifault.CircuitBreakerRule{}
	ruleMatcher := &apifault.RuleMatcher{}
	if inbound {
		ruleMatcher.Destination = &apifault.RuleMatcher_DestinationService{
			Service:   svc,
			Namespace: oldRuleProto.GetNamespace().GetValue(),
		}
		ruleMatcher.Source = &apifault.RuleMatcher_SourceService{}
		if len(subRule.Sources) > 0 {
			ruleMatcher.Source.Namespace = subRule.Sources[0].GetNamespace().GetValue()
			ruleMatcher.Source.Service = subRule.Sources[0].GetService().GetValue()
		}
		if len(ruleMatcher.Source.Namespace) == 0 {
			ruleMatcher.Source.Namespace = "*"
		}
		if len(ruleMatcher.Source.Service) == 0 {
			ruleMatcher.Source.Service = "*"
		}
	} else {
		ruleMatcher.Source = &apifault.RuleMatcher_SourceService{
			Service:   svc,
			Namespace: oldRuleProto.GetNamespace().GetValue(),
		}
		ruleMatcher.Destination = &apifault.RuleMatcher_DestinationService{}
		if len(subRule.Destinations) > 0 {
			ruleMatcher.Destination.Service = subRule.Destinations[0].GetService().GetValue()
			ruleMatcher.Destination.Namespace = subRule.Destinations[0].GetNamespace().GetValue()
		}
		if len(ruleMatcher.Destination.Namespace) == 0 {
			ruleMatcher.Destination.Namespace = "*"
		}
		if len(ruleMatcher.Destination.Service) == 0 {
			ruleMatcher.Destination.Service = "*"
		}
	}
	cbRule.RuleMatcher = ruleMatcher
	if len(subRule.Destinations) == 0 {
		return nil
	}
	dstSet := subRule.Destinations[0]
	if nil != dstSet.GetMethod() {
		ruleMatcher.Destination.Method = &apimodel.MatchString{
			Type:  apimodel.MatchString_MatchStringType(dstSet.GetMethod().GetType()),
			Value: dstSet.GetMethod().GetValue(),
		}
	}
	var triggers []*apifault.TriggerCondition
	policy := dstSet.GetPolicy()
	cbRule.MaxEjectionPercent = policy.GetMaxEjectionPercent().GetValue()
	if policy.GetConsecutive().GetEnable().GetValue() {
		triggers = append(triggers, &apifault.TriggerCondition{
			TriggerType: apifault.TriggerCondition_CONSECUTIVE_ERROR,
			ErrorCount:  policy.GetConsecutive().GetConsecutiveErrorToOpen().GetValue(),
		})
	}
	if policy.GetErrorRate().GetEnable().GetValue() {
		triggers = append(triggers, &apifault.TriggerCondition{
			TriggerType:    apifault.TriggerCondition_ERROR_RATE,
			ErrorPercent:   policy.GetErrorRate().GetErrorRateToOpen().GetValue(),
			MinimumRequest: policy.GetErrorRate().GetRequestVolumeThreshold().GetValue(),
			Interval:       60,
		})
	}
	recoverRule := &apifault.RecoverCondition{
		SleepWindow:        uint32(dstSet.GetRecover().GetSleepWindow().GetSeconds()),
		ConsecutiveSuccess: dstSet.GetRecover().GetMaxRetryAfterHalfOpen().GetValue(),
	}
	if recoverRule.ConsecutiveSuccess == 0 {
		recoverRule.ConsecutiveSuccess = 3
	}
	cbRule.RecoverCondition = recoverRule
	if dstSet.GetRecover().GetOutlierDetectWhen() != apifault.RecoverConfig_NEVER {
		cbRule.FaultDetectConfig = &apifault.FaultDetectConfig{Enable: true}
	} else {
		cbRule.FaultDetectConfig = &apifault.FaultDetectConfig{Enable: false}
	}
	return cbRule
}

func oldCbRuleToNewCbRule(cbRule *CircuitBreakerData) ([]*NewCircuitBreakerData, error) {
	oldRuleProto := &apifault.CircuitBreaker{
		Id:         wrapperspb.String(cbRule.ID),
		Version:    wrapperspb.String(cbRule.Version),
		Name:       wrapperspb.String(cbRule.Name),
		Namespace:  wrapperspb.String(cbRule.Namespace),
		Owners:     wrapperspb.String(cbRule.Owner),
		Comment:    wrapperspb.String(cbRule.Comment),
		Ctime:      wrapperspb.String(Time2String(cbRule.CreateTime)),
		Mtime:      wrapperspb.String(Time2String(cbRule.ModifyTime)),
		Revision:   wrapperspb.String(cbRule.Revision),
		Business:   wrapperspb.String(cbRule.Business),
		Department: wrapperspb.String(cbRule.Department),
	}
	if cbRule.Inbounds != "" {
		var inBounds []*apifault.CbRule
		if err := json.Unmarshal([]byte(cbRule.Inbounds), &inBounds); err != nil {
			return nil, err
		}
		oldRuleProto.Inbounds = inBounds
	}
	if cbRule.Outbounds != "" {
		var outBounds []*apifault.CbRule
		if err := json.Unmarshal([]byte(cbRule.Outbounds), &outBounds); err != nil {
			return nil, err
		}
		oldRuleProto.Outbounds = outBounds
	}
	var ret []*NewCircuitBreakerData
	if len(cbRule.Inbounds) > 0 {
		for i, inboundRule := range oldRuleProto.Inbounds {
			newRuleProto := toCircuitBreakerRule(cbRule.Service, oldRuleProto, inboundRule, true)
			if nil == newRuleProto {
				continue
			}
			newRuleStr, err := json.Marshal(newRuleProto)
			if nil != err {
				log.Printf("fail to marshal proto %+v, err: %v", newRuleProto, err)
				return nil, err
			}
			newRuleData := &NewCircuitBreakerData{
				ID:           fmt.Sprintf("%s-%s-%d", oldRuleProto.GetId().GetValue(), "inbound", i),
				Name:         fmt.Sprintf("%s-%s-%d", oldRuleProto.GetName().GetValue(), "inbound", i),
				Namespace:    oldRuleProto.GetNamespace().GetValue(),
				Description:  oldRuleProto.GetComment().GetValue(),
				Level:        int(apifault.Level_INSTANCE),
				SrcService:   newRuleProto.GetRuleMatcher().GetSource().GetService(),
				SrcNamespace: newRuleProto.GetRuleMatcher().GetSource().GetNamespace(),
				DstService:   newRuleProto.GetRuleMatcher().GetDestination().GetService(),
				DstNamespace: newRuleProto.GetRuleMatcher().GetDestination().GetNamespace(),
				DstMethod:    newRuleProto.GetRuleMatcher().GetDestination().GetMethod().GetValue().GetValue(),
				Rule:         string(newRuleStr),
				Revision:     NewUUID(),
				Enable:       true,
			}
			log.Printf("converted new inbound circuitbreaker rule query is %v", *newRuleData)
			ret = append(ret, newRuleData)
		}
	}
	if len(cbRule.Outbounds) > 0 {
		for i, outboundRule := range oldRuleProto.Outbounds {
			newRuleProto := toCircuitBreakerRule(cbRule.Service, oldRuleProto, outboundRule, false)
			if nil == newRuleProto {
				continue
			}
			newRuleStr, err := json.Marshal(newRuleProto)
			if nil != err {
				log.Printf("fail to marshal proto %+v, err: %v", newRuleProto, err)
				os.Exit(1)
			}
			newRuleData := &NewCircuitBreakerData{
				ID:           fmt.Sprintf("%s-%s-%d", oldRuleProto.GetId().GetValue(), "outbound", i),
				Name:         fmt.Sprintf("%s-%s-%d", oldRuleProto.GetName().GetValue(), "outbound", i),
				Namespace:    oldRuleProto.GetNamespace().GetValue(),
				Description:  oldRuleProto.GetComment().GetValue(),
				Level:        int(apifault.Level_INSTANCE),
				SrcService:   newRuleProto.GetRuleMatcher().GetSource().GetService(),
				SrcNamespace: newRuleProto.GetRuleMatcher().GetSource().GetNamespace(),
				DstService:   newRuleProto.GetRuleMatcher().GetDestination().GetService(),
				DstNamespace: newRuleProto.GetRuleMatcher().GetDestination().GetNamespace(),
				DstMethod:    newRuleProto.GetRuleMatcher().GetDestination().GetMethod().GetValue().GetValue(),
				Rule:         string(newRuleStr),
				Revision:     NewUUID(),
				Enable:       true,
			}
			log.Printf("converted new outbound circuitbreaker rule query is %v", *newRuleData)
			ret = append(ret, newRuleData)
		}
	}
	return ret, nil
}

// NewUUID 返回一个随机的UUID
func NewUUID() string {
	uuidBytes := uuid.New()
	return hex.EncodeToString(uuidBytes[:])
}

func main() {
	flag.Parse()
	if len(dbAddr) == 0 {
		dbAddr = os.Getenv(EnvDbAddr)
	}
	if len(dbName) == 0 {
		dbName = os.Getenv(EnvDbName)
	}
	if len(dbUser) == 0 {
		dbUser = os.Getenv(EnvDbUser)
	}
	if len(dbPwd) == 0 {
		dbPwd = os.Getenv(EnvDbPwd)
	}
	if len(dbAddr) == 0 || len(dbName) == 0 || len(dbUser) == 0 || len(dbPwd) == 0 {
		log.Printf("invalid arguments: dbAddr %s, dbName %s, dbUser %s, dbPwd %s", dbAddr, dbName, dbUser, dbPwd)
		os.Exit(1)
	}
	dns := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPwd, dbAddr, dbName)
	log.Printf("start to connection database %s", dns)
	db, err := sql.Open("mysql", dns)
	if err != nil {
		log.Printf("sql open err: %s", err.Error())
		os.Exit(1)
	}
	defer db.Close()
	oldRuleDatas, err := queryOldRule(db)
	if err != nil {
		log.Printf("get old rule err: %s", err.Error())
		os.Exit(1)
	}
	log.Printf("selected old circuitbreaker rule count %d", len(oldRuleDatas))

	var newRules []*NewCircuitBreakerData
	for _, oldRuleData := range oldRuleDatas {
		subNewRules, err := oldCbRuleToNewCbRule(oldRuleData)
		if err != nil {
			log.Printf("convert old rule to new rule err: %s", err.Error())
			os.Exit(1)
		}
		newRules = append(newRules, subNewRules...)
	}
	log.Printf("converted new circuitbreaker rule count %d", len(newRules))

	for _, newRule := range newRules {
		log.Printf("start to insert rule %s into database, name %s", newRule.ID, newRule.Name)
		err = createCircuitBreakerRule(db, newRule)
		if nil != err {
			log.Printf("fail to process rule %s, name %s, err: %v", newRule.ID, newRule.Name, err)
		}
	}
}

func processWithTransaction(db *sql.DB, handle func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[Store][database] begin tx err: %s", err.Error())
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()
	return handle(tx)
}

func buildEtimeStr(enable bool) string {
	etimeStr := "sysdate()"
	if !enable {
		etimeStr = "STR_TO_DATE('1980-01-01 00:00:01', '%Y-%m-%d %H:%i:%s')"
	}
	return etimeStr
}

func createCircuitBreakerRule(db *sql.DB, cbRule *NewCircuitBreakerData) error {
	return processWithTransaction(db, func(tx *sql.Tx) error {
		etimeStr := buildEtimeStr(cbRule.Enable)
		str := fmt.Sprintf(insertCircuitBreakerRuleSql, etimeStr)
		if _, err := tx.Exec(str, cbRule.ID, cbRule.Name, cbRule.Namespace, cbRule.Enable, cbRule.Revision,
			cbRule.Description, cbRule.Level, cbRule.SrcService, cbRule.SrcNamespace, cbRule.DstService,
			cbRule.DstNamespace, cbRule.DstMethod, cbRule.Rule); err != nil {
			log.Printf("[Store][database] fail to insert cbRule exec sql, err: %s", err.Error())
			return err
		}
		if err := tx.Commit(); err != nil {
			log.Printf("[Store][database] fail to insert cbRule commit tx, rule(%+v) commit tx err: %s",
				cbRule, err.Error())
			return err
		}
		return nil
	})
}
