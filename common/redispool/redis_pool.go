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

package redispool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/polarismesh/polaris-server/common/log"
)

const (
	// Get get method define
	Get = iota
	// Set set method define
	Set
	// Del del method define
	Del
)

const (
	// keyPrefix the prefix for hb key
	keyPrefix = "hb_"
	// eventSep the separator to split event
	eventSep = ":"
)

func toRedisKey(instanceId string) string {
	return fmt.Sprintf("%s%s", keyPrefix, instanceId)
}

type Event struct {
	EventType int
	Id        string
}

// Task ckv任务请求结构体
type Task struct {
	taskType int
	id       string
	value    string
	respCh   chan *Resp
}

/**
 * Resp ckv任务结果
 */
type Resp struct {
	Value  string
	Err    error
	Exists bool
}

// MetaData ckv连接池元数据
type MetaData struct {
	insConnNum  int
	password    string
	maxIdle     int
	idleTimeout int
	address     string
}

// Instance ckv节点结构体
type Instance struct {
	index     uint32 // 节点在连接池中的序号
	addr      string
	redisPool *redis.Pool
	ch        []chan *Task
	stopCtx   chan struct{}
}

// Config redis pool configuration
type Config struct {
	KvAddr         string   `json:"kvAddr"`
	KvPasswd       string   `json:"kvPasswd"`
	SlotNum        int      `json:"slotNum"`
	MaxIdle        int      `json:"maxIdle"`
	IdleTimeout    Duration `json:"idleTimeout"`
	ConnectTimeout Duration `json:"connectTimeout"`
	MsgTimeout     Duration `json:"msgTimeout"`
	Concurrency    int      `json:"concurrency"`
}

// DefaultConfig redis pool configuration with default values
func DefaultConfig() *Config {
	return &Config{
		SlotNum:        30,
		MaxIdle:        200,
		IdleTimeout:    Duration(120 * time.Second),
		ConnectTimeout: Duration(500 * time.Millisecond),
		MsgTimeout:     Duration(200 * time.Millisecond),
		Concurrency:    200,
	}
}

// Duration duration alias
type Duration time.Duration

// MarshalJSON marshal duration to json
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON unmarshal json text to struct
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// Validate validate config params
func (c *Config) Validate() error {
	if len(c.KvAddr) == 0 {
		return errors.New("kvAddr is empty")
	}
	if len(c.KvPasswd) == 0 {
		return errors.New("KvPasswd is empty")
	}
	if c.SlotNum <= 0 {
		return errors.New("slotNum is empty")
	}
	if c.MaxIdle <= 0 {
		return errors.New("maxIdle is empty")
	}
	if c.IdleTimeout == 0 {
		return errors.New("idleTimeout is empty")
	}
	if c.ConnectTimeout == 0 {
		return errors.New("connectTimeout is empty")
	}
	if c.MsgTimeout == 0 {
		return errors.New("msgTimeout is empty")
	}
	if c.Concurrency <= 0 {
		return errors.New("concurrency is empty")
	}
	return nil
}

// Pool ckv连接池结构体
type Pool struct {
	config         *Config
	ctx            context.Context
	ch             []chan *Task
	redisPool      *redis.Pool
	redisDead      uint32
	recoverTimeSec int64
}

// NewPool init a redis connection pool instance
func NewPool(ctx context.Context, config *Config) *Pool {
	redisPool := &redis.Pool{
		MaxIdle:     config.MaxIdle,
		MaxActive:   config.Concurrency,
		IdleTimeout: time.Duration(config.IdleTimeout),
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", config.KvAddr, redis.DialPassword(config.KvPasswd),
				redis.DialConnectTimeout(time.Duration(config.ConnectTimeout)),
				redis.DialReadTimeout(time.Duration(config.MsgTimeout)),
				redis.DialWriteTimeout(time.Duration(config.MsgTimeout)))
			if err != nil {
				log.Infof("ERROR: fail init redis: %s", err.Error())
				return nil, err
			}
			return conn, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	pool := &Pool{
		config:         config,
		ctx:            ctx,
		redisPool:      redisPool,
		recoverTimeSec: time.Now().Unix(),
		ch:             make([]chan *Task, 0, config.Concurrency),
	}
	for i := 0; i < config.Concurrency; i++ {
		pool.ch = append(pool.ch, make(chan *Task, 100))
	}
	return pool
}

func hashValue(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

// Get 使用连接池，向redis发起Get请求
func (p *Pool) Get(id string, ch chan *Resp) error { // nolint
	if err := p.checkRedisDead(); nil != err {
		return err
	}
	task := &Task{
		taskType: Get,
		id:       id,
		respCh:   ch,
	}
	p.ch[hashValue(id)%p.config.Concurrency] <- task
	return nil
}

// Set 使用连接池，向redis发起Set请求
func (p *Pool) Set(id string, value string, ch chan *Resp) error { // nolint
	if err := p.checkRedisDead(); nil != err {
		return err
	}
	task := &Task{
		taskType: Set,
		id:       id,
		value:    value,
		respCh:   ch,
	}
	p.ch[hashValue(id)%p.config.Concurrency] <- task
	return nil
}

// Del 使用连接池，向redis发起Del请求
func (p *Pool) Del(id string, ch chan *Resp) error { // nolint
	if err := p.checkRedisDead(); nil != err {
		return err
	}
	task := &Task{
		taskType: Del,
		id:       id,
		respCh:   ch,
	}
	p.ch[hashValue(id)%p.config.Concurrency] <- task
	return nil
}

func (p *Pool) checkRedisDead() error {
	if atomic.LoadUint32(&p.redisDead) == 1 {
		return errors.New(fmt.Sprintf("redis %s is dead", p.config.KvAddr))
	}
	return nil
}

// Start 启动ckv连接池工作
func (p *Pool) Start() {
	for i := 0; i < p.config.Concurrency; i++ {
		go p.worker(i)
	}
	log.Infof("[RedisPool]redis pool started")
}

func (p *Pool) worker(idx int) {
	log.Infof("[Health Check]start redis pool %d", idx)
	for {
		select {
		case task := <-p.ch[idx]:
			p.handleTask(task)
		case <-p.ctx.Done():
			return
		}
	}
}

const (
	redisCheckInterval = 1 * time.Second
	errCountThreshold  = 2
	maxCheckCount      = 3
)

// checkRedis check redis alive
func (p *Pool) checkRedis() {
	ticker := time.NewTicker(redisCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var errCount int
			for i := 0; i < maxCheckCount; i++ {
				if !p.doCheckRedis() {
					errCount++
				}
			}
			if errCount >= errCountThreshold {
				if atomic.CompareAndSwapUint32(&p.redisDead, 0, 1) {
					atomic.StoreInt64(&p.recoverTimeSec, 0)
				}
			} else {
				if atomic.CompareAndSwapUint32(&p.redisDead, 1, 0) {
					atomic.StoreInt64(&p.recoverTimeSec, time.Now().Unix())
				}
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// RecoverTimeSec the time second record when recover
func (p *Pool) RecoverTimeSec() int64 {
	return atomic.LoadInt64(&p.recoverTimeSec)
}

// doCheckRedis test the connection
func (p *Pool) doCheckRedis() bool {
	conn := p.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	if err != nil {
		return false
	}
	atomic.StoreUint32(&p.redisDead, 0)
	return true
}

// handleTask 任务处理函数
func (p *Pool) handleTask(task *Task) {
	if task == nil {
		log.Errorf("receive nil task")
		return
	}
	con := p.redisPool.Get()
	defer con.Close()

	var resp Resp
	switch task.taskType {
	case Get:
		resp.Value, resp.Err = redis.String(con.Do("GET", toRedisKey(task.id)))
		resp.Exists = true
		if resp.Err == redis.ErrNil {
			resp.Err = nil
			resp.Exists = false
		}
		task.respCh <- &resp
	case Set:
		_, resp.Err = con.Do("SET", toRedisKey(task.id), task.value)
		task.respCh <- &resp
	case Del:
		_, resp.Err = con.Do("DEL", toRedisKey(task.id))
		task.respCh <- &resp
	default:
		log.Errorf("[ckv] set key:%s type:%d wrong", task.id, task.taskType)
	}
}
