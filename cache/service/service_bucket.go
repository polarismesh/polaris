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

package service

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"sync"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/common/model"
)

type serviceAliasBucket struct {
	lock sync.RWMutex
	// aliase namespace->service->alias_id
	alias map[string]map[string]map[string]*model.Service
}

func newServiceAliasBucket() *serviceAliasBucket {
	return &serviceAliasBucket{
		alias: make(map[string]map[string]map[string]*model.Service),
	}
}

func (s *serviceAliasBucket) cleanServiceAlias(aliasFor *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.alias[aliasFor.Namespace]; !ok {
		return
	}
	delete(s.alias[aliasFor.Namespace], aliasFor.Name)
}

func (s *serviceAliasBucket) addServiceAlias(alias, aliasFor *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.alias[aliasFor.Namespace]; !ok {
		s.alias[aliasFor.Namespace] = map[string]map[string]*model.Service{}
	}
	if _, ok := s.alias[aliasFor.Namespace][aliasFor.Name]; !ok {
		s.alias[aliasFor.Namespace][aliasFor.Name] = map[string]*model.Service{}
	}

	bucket := s.alias[aliasFor.Namespace][aliasFor.Name]
	bucket[alias.ID] = alias
}

func (s *serviceAliasBucket) delServiceAlias(alias, aliasFor *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.alias[aliasFor.Namespace]; !ok {
		return
	}
	if _, ok := s.alias[aliasFor.Namespace][aliasFor.Name]; !ok {
		return
	}

	bucket := s.alias[aliasFor.Namespace][aliasFor.Name]
	delete(bucket, alias.ID)
}

func (s *serviceAliasBucket) getServiceAliases(aliasFor *model.Service) []*model.Service {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]*model.Service, 0, 8)
	if _, ok := s.alias[aliasFor.Namespace]; !ok {
		return ret
	}
	if _, ok := s.alias[aliasFor.Namespace][aliasFor.Name]; !ok {
		return ret
	}

	bucket := s.alias[aliasFor.Namespace][aliasFor.Name]
	for i := range bucket {
		ret = append(ret, bucket[i])
	}
	return ret
}

type serviceNamespaceBucket struct {
	lock     sync.RWMutex
	revision string
	names    map[string]*serviceNameBucket
}

func newServiceNamespaceBucket() *serviceNamespaceBucket {
	return &serviceNamespaceBucket{
		names: map[string]*serviceNameBucket{},
	}
}

func (s *serviceNamespaceBucket) addService(svc *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.names[svc.Namespace]; !ok {
		s.names[svc.Namespace] = &serviceNameBucket{
			names: make(map[string]*model.Service),
		}
	}

	s.names[svc.Namespace].addService(svc)
}

func (s *serviceNamespaceBucket) removeService(svc *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.names[svc.Namespace]; !ok {
		s.names[svc.Namespace] = &serviceNameBucket{
			names: make(map[string]*model.Service),
		}
	}

	s.names[svc.Namespace].removeService(svc)
}

func (s *serviceNamespaceBucket) reloadRevision() {
	s.lock.Lock()
	defer s.lock.Unlock()

	revisions := make([]string, 0, len(s.names))
	for i := range s.names {
		s.names[i].reloadRevision()
		revisions = append(revisions, s.names[i].revisions)
	}

	sort.Strings(revisions)
	h := sha1.New()
	for i := range revisions {
		if _, err := h.Write([]byte(revisions[i])); err != nil {
			log.Error("[Cache][Service] rebuild service name all list revision", zap.Error(err))
			return
		}
	}

	s.revision = hex.EncodeToString(h.Sum(nil))
}

func (s *serviceNamespaceBucket) ListAllServices() (string, []*model.Service) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]*model.Service, 0, 32)
	for namespace := range s.names {
		_, val := s.ListServices(namespace)
		ret = append(ret, val...)
	}

	return s.revision, ret
}

func (s *serviceNamespaceBucket) ListServices(namespace string) (string, []*model.Service) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	b, ok := s.names[namespace]
	if !ok {
		return "", []*model.Service{}
	}

	return b.listServices()
}

type serviceNameBucket struct {
	lock      sync.RWMutex
	revisions string
	names     map[string]*model.Service
}

func (s *serviceNameBucket) addService(svc *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.names[svc.Name] = svc
}

func (s *serviceNameBucket) removeService(svc *model.Service) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.names, svc.Name)
}

func (s *serviceNameBucket) listServices() (string, []*model.Service) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ret := make([]*model.Service, 0, len(s.names))
	for i := range s.names {
		ret = append(ret, s.names[i])
	}

	return s.revisions, ret
}

func (s *serviceNameBucket) reloadRevision() {
	s.lock.Lock()
	defer s.lock.Unlock()

	revisions := make([]string, 0, len(s.names))
	for i := range s.names {
		revisions = append(revisions, s.names[i].Revision)
	}

	sort.Strings(revisions)

	h := sha1.New()
	for i := range revisions {
		if _, err := h.Write([]byte(revisions[i])); err != nil {
			log.Error("[Cache][Service] rebuild service name list revision", zap.Error(err))
			return
		}
	}

	s.revisions = hex.EncodeToString(h.Sum(nil))
}
