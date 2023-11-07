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

package discover

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/polarismesh/polaris/apiserver/nacosserver/core"
	"github.com/polarismesh/polaris/common/log"
	commontime "github.com/polarismesh/polaris/common/time"
)

func NewUDPPushCenter(store *core.NacosDataStorage) (core.PushCenter, error) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	baseCenter, err := core.NewBasePushCenter(store)
	if err != nil {
		return nil, err
	}
	pushCenter := &UdpPushCenter{
		BasePushCenter: baseCenter,
		udpLn:          ln,
		srcAddr:        ln.LocalAddr().(*net.UDPAddr),
	}
	go pushCenter.cleanZombieClient()
	return pushCenter, nil
}

type UdpPushCenter struct {
	*core.BasePushCenter
	lock    sync.RWMutex
	udpLn   *net.UDPConn
	srcAddr *net.UDPAddr
}

func (p *UdpPushCenter) AddSubscriber(s core.Subscriber) {
	notifier := newUDPNotifier(s, p.srcAddr)
	if ok := p.BasePushCenter.AddSubscriber(s, notifier); !ok {
		_ = notifier.Close()
		return
	}
	client := p.BasePushCenter.GetSubscriber(s)
	if client != nil {
		client.RefreshLastTime()
	}
}

func (p *UdpPushCenter) RemoveSubscriber(s core.Subscriber) {
	p.BasePushCenter.RemoveSubscriber(s)
}

func (p *UdpPushCenter) EnablePush(s core.Subscriber) bool {
	return p.Type() == s.Type
}

func (p *UdpPushCenter) Type() core.PushType {
	return core.UDPCPush
}

func (p *UdpPushCenter) cleanZombieClient() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		p.BasePushCenter.RemoveClientIf(func(s string, client *core.WatchClient) bool {
			if !client.IsZombie() {
				return false
			}
			sub := client.GetSubscribers()
			log.Info("[NACOS-V2][PushCenter] remove zombie udp subscriber", zap.Any("info", sub))
			return true
		})
	}
}

func newUDPNotifier(s core.Subscriber, srcAddr *net.UDPAddr) *UDPNotifier {
	connector := &UDPNotifier{
		subscriber: s,
		srcAddr:    srcAddr,
	}
	return connector
}

type UDPNotifier struct {
	lock        sync.Mutex
	subscriber  core.Subscriber
	conn        *net.UDPConn
	lastRefTime int64
	srcAddr     *net.UDPAddr
}

func (c *UDPNotifier) doConnect() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.conn != nil {
		return nil
	}

	conn, err := net.DialUDP("udp", c.srcAddr, &net.UDPAddr{
		IP:   net.IP(c.subscriber.AddrStr),
		Port: c.subscriber.Port,
	})
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *UDPNotifier) Notify(d *core.PushData) error {
	data := d.CompressUDPData
	if len(data) == 0 {
		body, err := json.Marshal(d.UDPData)
		if err != nil {
			return err
		}
		data = body
	}
	if len(data) == 0 {
		return nil
	}
	return c.send(data)
}

func (c *UDPNotifier) send(data []byte) error {
	if err := c.doConnect(); err != nil {
		return err
	}

	if _, err := c.conn.Write(data); err != nil {
		return err
	}
	return nil
}

func (c *UDPNotifier) IsZombie() bool {
	return commontime.CurrentMillisecond()-c.lastRefTime > 10*1000
}

func (c *UDPNotifier) Close() error {
	return nil
}

type UDPAckPacket struct {
	Type        string
	LastRefTime int64
	Data        string
}
