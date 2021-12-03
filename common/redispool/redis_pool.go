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
	"errors"
	"fmt"
	"github.com/polarismesh/polaris-server/common/utils"
	"github.com/polarismesh/polaris-server/plugin"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/polarismesh/polaris-server/common/log"
)

const (
	// Get get method define
	Get = iota
	// Set set method define
	Set
	// Del del method define
	Del
	// Sadd del method define
	Sadd
	// Srem del method define
	Srem
)

var (
	typeToCommand = map[int]string{
		Get:  "GET",
		Set:  "SET",
		Del:  "DEL",
		Sadd: "SADD",
		Srem: "SREM",
	}
)

const (
	// keyPrefix the prefix for hb key
	keyPrefix = "hb_"
)

func toRedisKey(instanceId string, compatible bool) string {
	if compatible {
		return instanceId
	}
	return fmt.Sprintf("%s%s", keyPrefix, instanceId)
}

// Task ckv任务请求结构体
type Task struct {
	taskType int
	id       string
	value    string
	members  []string
	respChan chan *Resp
}

// String
func (t Task) String() string {
	return fmt.Sprintf("{taskType: %s, id: %s", typeToCommand[t.taskType], t.id)
}

/**
 * Resp ckv任务结果
 */
type Resp struct {
	Value      string
	Err        error
	Exists     bool
	Compatible bool
}

// Config redis pool configuration
type Config struct {
	KvAddr         string         `json:"kvAddr"`
	KvPasswd       string         `json:"kvPasswd"`
	MaxIdle        int            `json:"maxIdle"`
	IdleTimeout    utils.Duration `json:"idleTimeout"`
	ConnectTimeout utils.Duration `json:"connectTimeout"`
	MsgTimeout     utils.Duration `json:"msgTimeout"`
	Concurrency    int            `json:"concurrency"`
	Compatible     bool           `json:"compatible"`
	MaxRetry       int            `json:"maxRetry"`
	MinBatchCount  int            `json:"minBatchCount"`
	WaitTime       utils.Duration `json:"waitTime"`
}

// DefaultConfig redis pool configuration with default values
func DefaultConfig() *Config {
	return &Config{
		MaxIdle:        200,
		IdleTimeout:    utils.Duration(120 * time.Second),
		ConnectTimeout: utils.Duration(300 * time.Millisecond),
		MsgTimeout:     utils.Duration(300 * time.Millisecond),
		Concurrency:    200,
		Compatible:     false,
		MaxRetry:       2,
		MinBatchCount:  10,
		WaitTime:       utils.Duration(50 * time.Millisecond),
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
	if c.MaxRetry < 0 {
		return errors.New("maxRetry is empty")
	}
	return nil
}

// Pool ckv连接池结构体
type Pool struct {
	config         *Config
	ctx            context.Context
	redisClient    *redis.Client
	redisDead      uint32
	recoverTimeSec int64
	statis         plugin.Statis
	taskChans      []chan *Task
}

// NewPool init a redis connection pool instance
func NewPool(ctx context.Context, config *Config, statis plugin.Statis) *Pool {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         config.KvAddr,
		Password:     config.KvPasswd,
		MaxRetries:   config.MaxRetry,
		DialTimeout:  time.Duration(config.ConnectTimeout),
		ReadTimeout:  time.Duration(config.MsgTimeout),
		WriteTimeout: time.Duration(config.MsgTimeout),
		PoolSize:     config.MaxIdle,
		MinIdleConns: config.MaxIdle,
		IdleTimeout:  time.Duration(config.IdleTimeout),
	})
	pool := &Pool{
		config:         config,
		ctx:            ctx,
		redisClient:    redisClient,
		recoverTimeSec: time.Now().Unix(),
		statis:         statis,
		taskChans:      make([]chan *Task, 0, config.Concurrency),
	}
	for i := 0; i < config.Concurrency; i++ {
		pool.taskChans = append(pool.taskChans, make(chan *Task, 1024))
	}
	return pool
}

// Get 使用连接池，向redis发起Get请求
func (p *Pool) Get(id string) *Resp { // nolint
	if err := p.checkRedisDead(); nil != err {
		return &Resp{Err: err}
	}
	task := &Task{
		taskType: Get,
		id:       id,
	}
	return p.handleTask(task)
}

// Get 使用连接池，向redis发起Sdd请求
func (p *Pool) Sdd(id string, members []string) *Resp { // nolint
	if err := p.checkRedisDead(); nil != err {
		return &Resp{Err: err}
	}
	task := &Task{
		taskType: Sadd,
		id:       id,
		members:  members,
	}
	return p.handleTask(task)
}

// Get 使用连接池，向redis发起Srem请求
func (p *Pool) Srem(id string, members []string) *Resp { // nolint
	if err := p.checkRedisDead(); nil != err {
		return &Resp{Err: err}
	}
	task := &Task{
		taskType: Srem,
		id:       id,
		members:  members,
	}
	return p.handleTask(task)
}

// RedisObject 序列化对象
type RedisObject interface {
	// Serialize 序列化成字符串
	Serialize(compatible bool) string
	// Deserialize 反序列为对象
	Deserialize(value string, compatible bool) error
}

// Set 使用连接池，向redis发起Set请求
func (p *Pool) Set(id string, redisObj RedisObject) *Resp { // nolint
	if err := p.checkRedisDead(); nil != err {
		return &Resp{Err: err}
	}
	task := &Task{
		taskType: Set,
		id:       id,
		value:    redisObj.Serialize(p.config.Compatible),
	}
	return p.handleTask(task)
}

// Del 使用连接池，向redis发起Del请求
func (p *Pool) Del(id string) *Resp { // nolint
	if err := p.checkRedisDead(); nil != err {
		return &Resp{Err: err}
	}
	task := &Task{
		taskType: Del,
		id:       id,
	}
	return p.handleTask(task)
}

func (p *Pool) checkRedisDead() error {
	if atomic.LoadUint32(&p.redisDead) == 1 {
		return errors.New(fmt.Sprintf("redis %s is dead", p.config.KvAddr))
	}
	return nil
}

// Start 启动ckv连接池工作
func (p *Pool) Start() {
	wg := &sync.WaitGroup{}
	wg.Add(p.config.Concurrency)
	p.startWorkers(wg)
	go p.checkRedis(wg)
	log.Infof("[RedisPool]redis pool started")
}

func (p *Pool) startWorkers(wg *sync.WaitGroup) {
	for i := 0; i < p.config.Concurrency; i++ {
		go p.process(wg, i)
	}
}

func (p *Pool) process(wg *sync.WaitGroup, idx int) {
	log.Infof("[RedisPool]redis worker %d started", idx)
	ticker := time.NewTicker(time.Duration(p.config.WaitTime))
	piper := p.redisClient.Pipeline()
	defer func() {
		ticker.Stop()
		_ = piper.Close()
		wg.Done()
	}()
	var tasks []*Task
	for {
		select {
		case task := <-p.taskChans[idx]:
			tasks = append(tasks, task)
			if len(tasks) >= p.config.MinBatchCount {
				p.handleTasks(tasks, piper)
				tasks = nil
			}
		case <-ticker.C:
			if len(tasks) > 0 {
				p.handleTasks(tasks, piper)
				tasks = nil
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) handleTasks(tasks []*Task, piper redis.Pipeliner) {
	cmders := make([]redis.Cmder, len(tasks))
	for i, task := range tasks {
		cmders[i] = p.doHandleTask(task, piper)
	}
	_, _ = piper.Exec(context.Background())
	for i, cmder := range cmders {
		func(idx int, cmd redis.Cmder) {
			var resp = &Resp{}
			task := tasks[idx]
			defer func() {
				task.respChan <- resp
			}()
			switch typedCmd := cmd.(type) {
			case *redis.StringCmd:
				resp.Value, resp.Err = typedCmd.Result()
				resp.Exists = true
				if resp.Err == redis.Nil {
					resp.Err = nil
					resp.Exists = false
				}
			case *redis.StatusCmd:
				_, resp.Err = typedCmd.Result()
			case *redis.IntCmd:
				_, resp.Err = typedCmd.Result()
			default:
				resp.Err = fmt.Errorf("unknown type %s for task %s", typedCmd, *task)
			}
		}(i, cmder)
	}
}

const (
	redisCheckInterval = 1 * time.Second
	errCountThreshold  = 2
	maxCheckCount      = 3
)

// checkRedis check redis alive
func (p *Pool) checkRedis(wg *sync.WaitGroup) {
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
			wg.Wait()
			_ = p.redisClient.Close()
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
	_, err := p.redisClient.Ping(context.Background()).Result()
	if err != nil {
		return false
	}
	return true
}

const (
	maxProcessDuration = 1000 * time.Millisecond
)

var indexer int64

func nextIndex() int64 {
	value := atomic.AddInt64(&indexer, 1)
	if value == math.MaxInt64 {
		atomic.CompareAndSwapInt64(&indexer, value, 0)
		value = atomic.AddInt64(&indexer, 1)
	}
	return value
}

// handleTask 任务处理函数
func (p *Pool) handleTask(task *Task) *Resp {
	var startTime = time.Now()
	task.respChan = make(chan *Resp, 1)
	idx := int(nextIndex()) % len(p.taskChans)
	select {
	case p.taskChans[idx] <- task:
	case <-p.ctx.Done():
		return &Resp{Err: fmt.Errorf("worker has been stopped while sheduling task %s", *task),
			Compatible: p.config.Compatible}
	}
	var resp *Resp
	select {
	case resp = <-task.respChan:
	case <-p.ctx.Done():
		return &Resp{Err: fmt.Errorf("worker has been stopped while fetching resp for task %s", *task),
			Compatible: p.config.Compatible}
	}
	resp.Compatible = p.config.Compatible
	p.afterHandleTask(startTime, typeToCommand[task.taskType], task, resp)
	return resp
}

const (
	callResultOk   = 0
	callResultFail = 1
)

func (p *Pool) afterHandleTask(startTime time.Time, command string, task *Task, resp *Resp) {
	costDuration := time.Since(startTime)
	if costDuration >= maxProcessDuration && task.taskType != Get {
		log.Warnf("[Health Check][RedisCheck]to slow to process task %s, "+
			"duration %s, greater than %s", task.String(), costDuration, maxProcessDuration)
	}
	code := callResultOk
	if nil != resp.Err {
		code = callResultFail
	}
	if nil != p.statis {
		_ = p.statis.AddRedisCall(command, code, costDuration.Nanoseconds())
	}
}

func (p *Pool) doHandleTask(task *Task, piper redis.Pipeliner) redis.Cmder {
	switch task.taskType {
	case Set:
		return piper.Set(context.Background(), toRedisKey(task.id, p.config.Compatible), task.value, 0)
	case Del:
		return piper.Del(context.Background(), toRedisKey(task.id, p.config.Compatible))
	case Sadd:
		return piper.SAdd(context.Background(), task.id, task.members)
	case Srem:
		return piper.SRem(context.Background(), task.id, task.members)
	default:
		return piper.Get(context.Background(), toRedisKey(task.id, p.config.Compatible))
	}
}
