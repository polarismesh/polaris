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

package connlimit

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/polarismesh/polaris-server/common/log"
)

const (
	// 最少连接数
	minHostConnLimit = 1
)

// 计数器
// limit connections for every ip
type counter struct {
	size       int32
	actives    map[string]*Conn // 活跃的连接
	mu         *sync.RWMutex
	lastAccess int64
}

// 新增计数器
func newCounter() *counter {
	return &counter{
		size:       1,
		actives:    make(map[string]*Conn),
		mu:         &sync.RWMutex{},
		lastAccess: time.Now().Unix(),
	}
}

/**
 * Listener 包装 net.Listener
 */
type Listener struct {
	net.Listener
	protocol             string             // 协议，主要用以日志记录与全局对象索引
	conns                sync.Map           // 保存 ip -> counter
	maxConnPerHost       int32              // 每个IP最多的连接数
	maxConnLimit         int32              // 当前listener最大的连接数限制
	whiteList            map[string]bool    // 白名单列表
	readTimeout          time.Duration      // 读超时
	connCount            int32              // 当前listener保持连接的个数
	purgeCounterInterval time.Duration      // 回收过期counter的
	purgeCounterExpire   int64              // counter过期的秒数
	purgeCancel          context.CancelFunc // 停止purge协程的ctx
}

// NewListener returns a new listener
// @param l 网络连接
// @param protocol 当前listener的七层协议，比如http，grpc等
func NewListener(l net.Listener, protocol string, config *Config) (net.Listener, error) {
	// 参数校验
	if protocol == "" {
		log.Errorf("[ConnLimit] listener is missing protocol")
		return nil, errors.New("listener is missing protocol")
	}
	if config == nil || !config.OpenConnLimit {
		log.Infof("[ConnLimit][%s] apiserver is not open conn limit", protocol)
		return l, nil
	}
	if config.PurgeCounterInterval == 0 || config.PurgeCounterExpire == 0 {
		log.Errorf("[ConnLimit][%s] purge params invalid", protocol)
		return nil, errors.New("purge params invalid")
	}

	hostConnLimit := int32(config.MaxConnPerHost)
	lisConnLimit := int32(config.MaxConnLimit)
	// 参数校验, perHost阈值不能小于1
	if hostConnLimit < minHostConnLimit {
		return nil, fmt.Errorf("invalid conn limit: %d, can't be smaller than %d", hostConnLimit, minHostConnLimit)
	}

	whites := strings.Split(config.WhiteList, ",")
	whiteList := make(map[string]bool, len(whites))
	for _, entry := range whites {
		if entry == "" {
			continue
		}

		whiteList[entry] = true
	}
	log.Infof("[ConnLimit] host conn limit white list: %+v", whites)

	lis := &Listener{
		Listener:             l,
		protocol:             protocol,
		maxConnPerHost:       hostConnLimit,
		maxConnLimit:         lisConnLimit,
		whiteList:            whiteList,
		readTimeout:          config.ReadTimeout,
		purgeCounterInterval: config.PurgeCounterInterval,
		purgeCounterExpire:   int64(config.PurgeCounterExpire / time.Second),
	}
	// 把listener放到全局变量中，方便外部访问
	if err := SetLimitListener(lis); err != nil {
		return nil, err
	}
	// 启动回收协程，定时回收过期counter
	ctx, cancel := context.WithCancel(context.Background())
	lis.purgeExpireCounter(ctx)
	lis.purgeCancel = cancel
	return lis, nil
}

// Accept 接收连接
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return l.accept(c), nil
}

// Close 关闭连接
func (l *Listener) Close() error {
	log.Infof("[Listener][%s] close the listen fd", l.protocol)
	l.purgeCancel()
	return l.Listener.Close()
}

// GetHostConnCount 查看对应ip的连接数
func (l *Listener) GetHostConnCount(host string) int32 {
	var connNum int32
	if value, ok := l.conns.Load(host); ok {
		c := value.(*counter)
		c.mu.RLock()
		connNum = c.size
		c.mu.RUnlock()
	}

	return connNum
}

// Range 遍历当前持有连接的host
func (l *Listener) Range(fn func(host string, count int32) bool) {
	l.conns.Range(func(key, value interface{}) bool {
		host := key.(string)
		return fn(host, l.GetHostConnCount(host))
	})
}

// GetListenerConnCount 查看当前监听server保持的连接数
func (l *Listener) GetListenerConnCount() int32 {
	return atomic.LoadInt32(&l.connCount)
}

// GetDistinctHostCount 获取当前缓存的host的个数
func (l *Listener) GetDistinctHostCount() int32 {
	var count int32
	l.conns.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetHostActiveConns 获取指定host的活跃的连接
func (l *Listener) GetHostActiveConns(host string) map[string]*Conn {
	obj, ok := l.conns.Load(host)
	if !ok {
		return nil
	}

	ct := obj.(*counter)
	ct.mu.RLock()
	out := make(map[string]*Conn, len(ct.actives))
	for address, conn := range ct.actives {
		out[address] = conn
	}
	ct.mu.RUnlock()

	return out
}

// GetHostConnStats 获取客户端连接的stat信息
func (l *Listener) GetHostConnStats(host string) []*HostConnStat {
	loadStat := func(h string, ct *counter) *HostConnStat {
		ct.mu.RLock()
		stat := &HostConnStat{
			Host:       h,
			Amount:     ct.size,
			LastAccess: time.Unix(ct.lastAccess, 0),
			Actives:    make([]string, 0, len(ct.actives)),
		}
		for client := range ct.actives {
			stat.Actives = append(stat.Actives, client)
		}
		ct.mu.RUnlock()
		return stat
	}

	var out []*HostConnStat
	// 只获取一个，推荐每次只获取一个
	if host != "" {
		if obj, ok := l.conns.Load(host); ok {
			out = append(out, loadStat(host, obj.(*counter)))
			return out
		}
		return nil
	}

	// 全量扫描，比较耗时
	l.conns.Range(func(key, value interface{}) bool {
		out = append(out, loadStat(key.(string), value.(*counter)))
		return true
	})
	return out
}

// GetHostConnection 获取指定host和port的连接
func (l *Listener) GetHostConnection(host string, port int) *Conn {
	obj, ok := l.conns.Load(host)
	if !ok {
		return nil
	}

	ct := obj.(*counter)
	target := fmt.Sprintf("%s:%d", host, port)
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	for address, conn := range ct.actives {
		if address == target {
			return conn
		}
	}

	return nil
}

// 封装一层，让关键函数acquire的更具备可测试性（不需要mock net.Conn）
func (l *Listener) accept(conn net.Conn) net.Conn {
	address := conn.RemoteAddr().String()
	// addr解析失败, 不做限制
	ipPort := strings.Split(address, ":")
	if len(ipPort) != 2 || ipPort[0] == "" {
		return conn
	}
	return l.acquire(conn, address, ipPort[0])
}

// 包裹一下conn
// 增加ip的连接计数，如果发现ip连接达到上限，则关闭
// conn 	原始连接
// address 	客户端地址
// host 	处理后的客户端IP地址
func (l *Listener) acquire(conn net.Conn, address string, host string) *Conn {
	limiterConn := &Conn{
		Conn:     conn,
		closed:   false,
		address:  address,
		host:     host,
		listener: l,
	}

	log.Debugf("acquire conn for: %s", address)
	if ok := l.incConnCount(); !ok {
		log.Errorf("[ConnLimit][%s] host(%s) reach apiserver conn limit(%d)", l.protocol, host, l.maxConnLimit)
		limiterConn.closed = true
		_ = limiterConn.Conn.Close()
		return limiterConn
	}

	value, ok := l.conns.Load(host)
	// 首次访问, 置1返回ok
	if !ok {
		ctr := newCounter()
		ctr.actives[address] = limiterConn
		l.conns.Store(host, ctr)
		return limiterConn
	}

	c := value.(*counter)
	c.mu.Lock() // release是并发的，因此需要加锁
	// 如果连接数已经超过阈值, 则返回失败, 使用方要调用release减少计数
	// 如果在白名单中，则直接忽略host连接限制
	if c.size >= l.maxConnPerHost && !l.ignoreHostConnLimit(host) {
		c.mu.Unlock()
		l.descConnCount() // 前面已经增加了计数，因此这里失败，必须减少计数
		log.Errorf("[ConnLimit][%s] host(%s) reach host conn limit(%d)", l.protocol, host, l.maxConnPerHost)
		limiterConn.closed = true
		_ = limiterConn.Conn.Close()
		return limiterConn
	}

	// 单个IP的连接，还有冗余，则增加计数
	c.size++
	c.actives[address] = limiterConn
	c.lastAccess = time.Now().Unix()
	// map里面存储的是指针，可以不用store，这里直接对指针的内存操作
	// l.conns.Store(host, c)
	c.mu.Unlock()
	return limiterConn
}

// 减少连接计数
func (l *Listener) release(conn *Conn) {
	log.Debugf("release conn for: %s", conn.host)
	l.descConnCount()

	if value, ok := l.conns.Load(conn.host); ok {
		c := value.(*counter)
		c.mu.Lock()
		c.size--
		// map里面存储的是指针，可以不用store，这里直接对指针的内存操作
		// l.conns.Store(host, c)
		delete(c.actives, conn.address)
		c.mu.Unlock()
	}

}

// 增加监听server的连接计数
// 这里使用了原子变量来增加计数，先判断是否超过最大限制
// 如果超过了，则立即返回false，否则计数+1
// 在计数+1的过程中，即使有Desc释放过程，也不影响
func (l *Listener) incConnCount() bool {
	if l.maxConnLimit <= 0 {
		return true
	}
	if count := atomic.LoadInt32(&l.connCount); count >= l.maxConnLimit {
		return false
	}

	atomic.AddInt32(&l.connCount, 1)
	return true
}

// 释放监听server的连接计数
func (l *Listener) descConnCount() {
	if l.maxConnLimit <= 0 {
		return
	}

	atomic.AddInt32(&l.connCount, -1)
}

// 判断host是否在白名单中
// 如果host在白名单中，则忽略host连接限制
func (l *Listener) ignoreHostConnLimit(host string) bool {
	_, ok := l.whiteList[host]
	return ok
}

// 回收长时间没有访问的IP
// 定时扫描
func (l *Listener) purgeExpireCounter(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(l.purgeCounterInterval)
		defer ticker.Stop()
		log.Infof("[Listener][%s] start doing purge expire counter", l.protocol)
		for {
			select {
			case <-ticker.C:
				l.purgeExpireCounterHandler()
			case <-ctx.Done():
				log.Infof("[Listener][%s] purge expire counter exit", l.protocol)
				return
			}
		}
	}()
}

// 回收过期counter执行函数
func (l *Listener) purgeExpireCounterHandler() {
	start := time.Now()
	scanCount := 0
	purgeCount := 0
	l.conns.Range(func(key, value interface{}) bool {
		scanCount++
		ct := value.(*counter)
		ct.mu.RLock()
		if ct.size == 0 && time.Now().Unix()-ct.lastAccess > l.purgeCounterExpire {
			// log.Infof("[Listener][%s] purge expire counter: %s", l.protocol, key.(string))
			l.conns.Delete(key)
			purgeCount++
		}
		ct.mu.RUnlock()
		return true
	})

	spendTime := time.Since(start)
	log.Infof("[Listener][%s] purge expire counter total(%d), use time: %+v, scan total(%d), scan qps: %.2f",
		l.protocol, purgeCount, spendTime, scanCount, float64(scanCount)/spendTime.Seconds())
}
