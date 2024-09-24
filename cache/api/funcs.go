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

package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"hash"
	"time"
)

const mtimeLogIntervalSec = 120

// LogLastMtime 定时打印mtime更新结果
func LogLastMtime(lastMtimeLogged int64, lastMtime int64, prefix string) int64 {
	curTimeSec := time.Now().Unix()
	if lastMtimeLogged == 0 || curTimeSec-lastMtimeLogged >= mtimeLogIntervalSec {
		lastMtimeLogged = curTimeSec
		log.Infof("[Cache][%s] current lastMtime is %s", prefix, time.Unix(lastMtime, 0))
	}
	return lastMtimeLogged
}

func ComputeRevisionBySlice(h hash.Hash, slice []string) (string, error) {
	for _, revision := range slice {
		if _, err := h.Write([]byte(revision)); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CompositeComputeRevision 将多个 revision 合并计算为一个
func CompositeComputeRevision(revisions []string) (string, error) {
	if len(revisions) == 1 {
		return revisions[0], nil
	}

	h := sha1.New()
	for i := range revisions {
		if _, err := h.Write([]byte(revisions[i])); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

//

type (
	namespacePredicateCtxKey          struct{}
	servicePredicateCtxKey            struct{}
	routeRulePredicateCtxKey          struct{}
	ratelimitRulePredicateCtxKey      struct{}
	circuitbreakerRulePredicateCtxKey struct{}
	faultdetectRulePredicateCtxKey    struct{}
	laneRulePredicateCtxKey           struct{}
	configGroupPredicateCtxKey        struct{}
	userPredicateCtxKey               struct{}
	userGroupPredicateCtxKey          struct{}
	authPolicyPredicateCtxKey         struct{}
	authRolePredicateCtxKey           struct{}
)

func AppendNamespacePredicate(ctx context.Context, p NamespacePredicate) context.Context {
	var predicates []NamespacePredicate

	val := ctx.Value(namespacePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]NamespacePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, namespacePredicateCtxKey{}, predicates)
}

func LoadNamespacePredicates(ctx context.Context) []NamespacePredicate {
	var predicates []NamespacePredicate

	val := ctx.Value(namespacePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]NamespacePredicate)
	}
	return predicates
}

func AppendServicePredicate(ctx context.Context, p ServicePredicate) context.Context {
	var predicates []ServicePredicate

	val := ctx.Value(servicePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]ServicePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, servicePredicateCtxKey{}, predicates)
}

func LoadServicePredicates(ctx context.Context) []ServicePredicate {
	var predicates []ServicePredicate

	val := ctx.Value(servicePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]ServicePredicate)
	}
	return predicates
}

func AppendRouterRulePredicate(ctx context.Context, p RouteRulePredicate) context.Context {
	var predicates []RouteRulePredicate

	val := ctx.Value(routeRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]RouteRulePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, routeRulePredicateCtxKey{}, predicates)
}

func LoadRouterRulePredicates(ctx context.Context) []RouteRulePredicate {
	var predicates []RouteRulePredicate

	val := ctx.Value(routeRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]RouteRulePredicate)
	}
	return predicates
}

func AppendRatelimitRulePredicate(ctx context.Context, p RateLimitRulePredicate) context.Context {
	var predicates []RateLimitRulePredicate

	val := ctx.Value(ratelimitRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]RateLimitRulePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, ratelimitRulePredicateCtxKey{}, predicates)
}

func LoadRatelimitRulePredicates(ctx context.Context) []RateLimitRulePredicate {
	var predicates []RateLimitRulePredicate

	val := ctx.Value(ratelimitRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]RateLimitRulePredicate)
	}
	return predicates
}

func AppendCircuitBreakerRulePredicate(ctx context.Context, p CircuitBreakerPredicate) context.Context {
	var predicates []CircuitBreakerPredicate

	val := ctx.Value(circuitbreakerRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]CircuitBreakerPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, circuitbreakerRulePredicateCtxKey{}, predicates)
}

func LoadCircuitBreakerRulePredicates(ctx context.Context) []CircuitBreakerPredicate {
	var predicates []CircuitBreakerPredicate

	val := ctx.Value(circuitbreakerRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]CircuitBreakerPredicate)
	}
	return predicates
}

func AppendFaultDetectRulePredicate(ctx context.Context, p FaultDetectPredicate) context.Context {
	var predicates []FaultDetectPredicate

	val := ctx.Value(faultdetectRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]FaultDetectPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, faultdetectRulePredicateCtxKey{}, predicates)
}

func LoadFaultDetectRulePredicates(ctx context.Context) []FaultDetectPredicate {
	var predicates []FaultDetectPredicate

	val := ctx.Value(faultdetectRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]FaultDetectPredicate)
	}
	return predicates
}

func AppendLaneRulePredicate(ctx context.Context, p LanePredicate) context.Context {
	var predicates []LanePredicate

	val := ctx.Value(laneRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]LanePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, laneRulePredicateCtxKey{}, predicates)
}

func LoadLaneRulePredicates(ctx context.Context) []LanePredicate {
	var predicates []LanePredicate

	val := ctx.Value(laneRulePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]LanePredicate)
	}
	return predicates
}

func AppendConfigGroupPredicate(ctx context.Context, p ConfigGroupPredicate) context.Context {
	var predicates []ConfigGroupPredicate

	val := ctx.Value(configGroupPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]ConfigGroupPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, configGroupPredicateCtxKey{}, predicates)
}

func LoadConfigGroupPredicates(ctx context.Context) []ConfigGroupPredicate {
	var predicates []ConfigGroupPredicate

	val := ctx.Value(configGroupPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]ConfigGroupPredicate)
	}
	return predicates
}

func AppendUserPredicate(ctx context.Context, p UserPredicate) context.Context {
	var predicates []UserPredicate

	val := ctx.Value(userPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]UserPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, userPredicateCtxKey{}, predicates)
}

func LoadUserPredicates(ctx context.Context) []UserPredicate {
	var predicates []UserPredicate

	val := ctx.Value(userPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]UserPredicate)
	}
	return predicates
}

func AppendUserGroupPredicate(ctx context.Context, p UserGroupPredicate) context.Context {
	var predicates []UserGroupPredicate

	val := ctx.Value(userGroupPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]UserGroupPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, userGroupPredicateCtxKey{}, predicates)
}

func LoadUserGroupPredicates(ctx context.Context) []UserGroupPredicate {
	var predicates []UserGroupPredicate

	val := ctx.Value(userGroupPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]UserGroupPredicate)
	}
	return predicates
}

func AppendAuthPolicyPredicate(ctx context.Context, p AuthPolicyPredicate) context.Context {
	var predicates []AuthPolicyPredicate

	val := ctx.Value(authPolicyPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]AuthPolicyPredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, authPolicyPredicateCtxKey{}, predicates)
}

func LoadAuthPolicyPredicates(ctx context.Context) []AuthPolicyPredicate {
	var predicates []AuthPolicyPredicate

	val := ctx.Value(userGroupPredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]AuthPolicyPredicate)
	}
	return predicates
}

func AppendAuthRolePredicate(ctx context.Context, p AuthRolePredicate) context.Context {
	var predicates []AuthRolePredicate

	val := ctx.Value(authRolePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]AuthRolePredicate)
	}

	predicates = append(predicates, p)
	return context.WithValue(ctx, authRolePredicateCtxKey{}, predicates)
}

func LoadAuthRolePredicates(ctx context.Context) []AuthRolePredicate {
	var predicates []AuthRolePredicate

	val := ctx.Value(authRolePredicateCtxKey{})
	if val != nil {
		predicates, _ = val.([]AuthRolePredicate)
	}
	return predicates
}
