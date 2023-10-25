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

package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	nacosmodel "github.com/polarismesh/polaris/apiserver/nacosserver/model"
	"github.com/polarismesh/polaris/common/eventhub"
	"github.com/polarismesh/polaris/common/log"
	"github.com/polarismesh/polaris/common/model"
	commontime "github.com/polarismesh/polaris/common/time"
)

type PushType string

const (
	NoopPush     PushType = "noop"
	UDPCPush     PushType = "udp"
	GRPCPush     PushType = "grpc"
	AssemblyPush PushType = "assembly"
)

type PushCenter interface {
	AddSubscriber(s Subscriber)
	RemoveSubscriber(s Subscriber)
	EnablePush(s Subscriber) bool
	Type() PushType
}

type PushData struct {
	Service          *model.Service
	ServiceInfo      *nacosmodel.ServiceInfo
	UDPData          interface{}
	CompressUDPData  []byte
	GRPCData         interface{}
	CompressGRPCData []byte
}

func WarpGRPCPushData(p *PushData) {
	data := map[string]interface{}{
		"type": "dom",
		"data": map[string]interface{}{
			"dom":             p.Service.Name,
			"cacheMillis":     p.ServiceInfo.CacheMillis,
			"lastRefTime":     p.ServiceInfo.LastRefTime,
			"checksum":        p.ServiceInfo.Checksum,
			"useSpecifiedURL": false,
			"hosts":           p.ServiceInfo.Hosts,
			"metadata":        p.Service.Meta,
		},
		"lastRefTime": time.Now().Nanosecond(),
	}
	p.GRPCData = data
	//nolint:errchkjson
	body, _ := json.Marshal(data)
	p.CompressGRPCData = CompressIfNecessary(body)
}

func WarpUDPPushData(p *PushData) {
	data := map[string]interface{}{
		"type": "dom",
		"data": map[string]interface{}{
			"dom":             p.Service.Name,
			"cacheMillis":     p.ServiceInfo.CacheMillis,
			"lastRefTime":     p.ServiceInfo.LastRefTime,
			"checksum":        p.ServiceInfo.Checksum,
			"useSpecifiedURL": false,
			"hosts":           p.ServiceInfo.Hosts,
			"metadata":        p.Service.Meta,
		},
		"lastRefTime": time.Now().Nanosecond(),
	}
	p.UDPData = data
	//nolint:errchkjson
	body, _ := json.Marshal(data)
	p.CompressUDPData = CompressIfNecessary(body)
}

const (
	maxDataSizeUncompress = 1024
)

func CompressIfNecessary(data []byte) []byte {
	if len(data) <= maxDataSizeUncompress {
		return data
	}

	var ret bytes.Buffer
	writer := gzip.NewWriter(&ret)
	_, err := writer.Write(data)
	if err != nil {
		return data
	}
	return ret.Bytes()
}

type Subscriber struct {
	Key         string
	ConnID      string
	AddrStr     string
	Agent       string
	App         string
	Ip          string
	Port        int
	NamespaceId string
	Group       string
	Service     string
	Cluster     string
	Type        PushType
}

type (
	Notifier interface {
		io.Closer
		Notify(d *PushData) error
		IsZombie() bool
	}

	WatchClient struct {
		subscriber         Subscriber
		notifier           Notifier
		lastRefreshTimeRef atomic.Int64
		lastCheclksum      string
	}
)

func (w *WatchClient) RefreshLastTime() {
	w.lastRefreshTimeRef.Store(commontime.CurrentMillisecond())
}

func (w *WatchClient) IsZombie() bool {
	return w.notifier.IsZombie()
}

func (w *WatchClient) GetSubscriber() Subscriber {
	return w.subscriber
}

type BasePushCenter struct {
	lock sync.RWMutex

	store *NacosDataStorage

	clients map[string]*WatchClient
	// notifiers namespace -> service -> notifiers
	notifiers map[string]map[nacosmodel.ServiceKey]map[string]*WatchClient

	watchCtx *eventhub.SubscribtionContext
}

func NewBasePushCenter(store *NacosDataStorage) (*BasePushCenter, error) {
	pc := &BasePushCenter{
		store:     store,
		clients:   map[string]*WatchClient{},
		notifiers: map[string]map[nacosmodel.ServiceKey]map[string]*WatchClient{},
	}
	subCtx, err := eventhub.Subscribe(nacosmodel.NacosServicesChangeEventTopic, pc)
	if err != nil {
		return nil, err
	}
	pc.watchCtx = subCtx
	return pc, nil
}

// RemoveClientIf .
func (pc *BasePushCenter) RemoveClientIf(test func(string, *WatchClient) bool) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	for i := range pc.clients {
		client := pc.clients[i]
		if test(i, client) {
			pc.removeSubscriber0(client.subscriber)
		}
	}
}

// PreProcess do preprocess logic for event
func (pc *BasePushCenter) PreProcess(_ context.Context, any any) any {
	return any
}

// OnEvent event process logic
func (pc *BasePushCenter) OnEvent(ctx context.Context, any2 any) error {
	event, ok := any2.(*nacosmodel.NacosServicesChangeEvent)
	if !ok {
		log.Error("[NACOS-CORE][PushCenter] receive event type not NacosServicesChangeEvent")
		return nil
	}
	for i := range event.Services {
		svc := event.Services[i]
		svcName := nacosmodel.GetServiceName(svc.Name)
		groupName := nacosmodel.GetGroupName(svc.Name)
		filterCtx := &FilterContext{
			Service:    ToNacosService(pc.store.Cache(), svc.Namespace, svcName, groupName),
			EnableOnly: true,
		}
		svcInfo := pc.store.ListInstances(filterCtx, NoopSelectInstances)
		pushData := &PushData{
			Service: &model.Service{
				ID:        svc.ServiceID,
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Meta:      svc.ExtendData,
			},
			ServiceInfo: svcInfo,
		}
		log.Info("[NACOS-CORE][PushCenter] notify subscriber data", zap.Any("pushData", pushData))
		// WarpGRPCPushData(pushData) // 目前根本不会使用这个数据
		WarpUDPPushData(pushData)
		svcKey := nacosmodel.ServiceKey{Namespace: svc.Namespace, Group: groupName, Name: svcName}
		pc.NotifyClients(svcKey, func(client *WatchClient) {
			if err := client.notifier.Notify(pushData); err != nil {
				log.Error("[NACOS-CORE][PushCenter] notify client fail", zap.String("ip", client.subscriber.Ip),
					zap.String("conn-id", client.subscriber.ConnID),
					zap.String("type", string(client.subscriber.Type)), zap.Error(err))
			}
		})
	}
	return nil
}

func (pc *BasePushCenter) GetSubscriber(s Subscriber) *WatchClient {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	id := s.Key
	val := pc.clients[id]
	return val
}

func (pc *BasePushCenter) AddSubscriber(s Subscriber, notifier Notifier) bool {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	id := s.Key
	if _, ok := pc.clients[id]; ok {
		return false
	}

	key := nacosmodel.ServiceKey{
		Namespace: s.NamespaceId,
		Group:     s.Group,
		Name:      s.Service,
	}

	client := &WatchClient{
		subscriber: s,
		notifier:   notifier,
	}
	pc.clients[id] = client

	if _, ok := pc.notifiers[s.NamespaceId]; !ok {
		pc.notifiers[s.NamespaceId] = map[nacosmodel.ServiceKey]map[string]*WatchClient{}
	}
	if _, ok := pc.notifiers[s.NamespaceId][key]; !ok {
		pc.notifiers[s.NamespaceId][key] = map[string]*WatchClient{}
	}
	_, ok := pc.notifiers[s.NamespaceId][key][id]
	if !ok {
		pc.notifiers[s.NamespaceId][key][id] = client
	}
	return true
}

func (pc *BasePushCenter) RemoveSubscriber(s Subscriber) {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	pc.removeSubscriber0(s)
}

func (pc *BasePushCenter) removeSubscriber0(s Subscriber) {
	id := s.Key
	if _, ok := pc.clients[id]; !ok {
		return
	}

	key := nacosmodel.ServiceKey{
		Namespace: s.NamespaceId,
		Group:     s.Group,
		Name:      s.Service,
	}

	if _, ok := pc.notifiers[s.NamespaceId]; ok {
		if _, ok = pc.notifiers[s.NamespaceId][key]; ok {
			if _, ok = pc.notifiers[s.NamespaceId][key][id]; ok {
				notifiers := pc.notifiers[s.NamespaceId][key]
				delete(notifiers, id)
				pc.notifiers[s.NamespaceId][key] = notifiers
			}
		}
	}

	if notifier, ok := pc.clients[id]; ok {
		_ = notifier.notifier.Close()
	}
	delete(pc.clients, id)
}

func (pc *BasePushCenter) NotifyClients(key nacosmodel.ServiceKey, notify func(client *WatchClient)) {
	pc.lock.RLock()
	defer pc.lock.RUnlock()

	ns, ok := pc.notifiers[key.Namespace]
	if !ok {
		return
	}
	clients, ok := ns[key]
	if !ok {
		return
	}

	log.Info("[NACOS-CORE][PushCenter] receive nacos services change", zap.String("namespace", key.Namespace),
		zap.String("group", key.Group), zap.String("service", key.Name), zap.Int("client-count", len(clients)))
	for i := range clients {
		notify(clients[i])
	}
}
