/*
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2020. Lorem THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 *  you may not use this file except in compliance with the License.
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
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"hash/crc32"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Get = 0
	Set = 1
	Del = 2
)

/**
 * @brief ckv任务请求结构体
 */
type Task struct {
	taskType int
	id       string
	status   int
	beatTime int64
	respCh   chan *Resp
}

/**
 * @brief ckv任务结果
 */
type Resp struct {
	Value string
	Err   error
	Local bool
}

/**
 * @brief ckv连接池元数据
 */
type MetaData struct {
	insConnNum  int
	kvPasswd    string
	localHost   string
	MaxIdle     int
	IdleTimeout int
}

/**
 * @brief ckv节点结构体
 */
type Instance struct {
	// 节点在连接池中的序号
	index     uint32
	addr      string
	redisPool *redis.Pool
	ch        []chan *Task
	stopCh    chan bool
}

/**
 * @brief ckv连接池结构体
 */
type Pool struct {
	mu          sync.Mutex
	meta        *MetaData
	instances   []*Instance
	instanceNum int32
}

/**
 * @brief 初始化一个redis连接池实例
 */
func NewPool(insConnNum int, kvPasswd, localHost string, redisInstances []*model.Instance,
	maxIdle, idleTimeout int) (*Pool, error) {
	var instances []*Instance
	if len(redisInstances) > 0 {
		for _, instance := range redisInstances {
			instance := &Instance{
				redisPool: genRedisPool(insConnNum, kvPasswd, instance, maxIdle, idleTimeout),
				stopCh:    make(chan bool),
			}
			instance.ch = make([]chan *Task, 0, 100*insConnNum)
			for i := 0; i < 100*insConnNum; i++ {
				instance.ch = append(instance.ch, make(chan *Task))
			}
			rand.Seed(time.Now().Unix())
			// 从一个随机位置开始，防止所有server都从一个ckv开始
			instance.index = uint32(rand.Intn(100 * insConnNum))
			instances = append(instances, instance)
		}
	}

	pool := &Pool{
		meta: &MetaData{
			insConnNum:  insConnNum,
			kvPasswd:    kvPasswd,
			localHost:   localHost,
			MaxIdle:     maxIdle,
			IdleTimeout: idleTimeout,
		},
		instances:   instances,
		instanceNum: int32(len(redisInstances)),
	}

	return pool, nil
}

func genRedisPool(insConnNum int, kvPasswd string, instance *model.Instance, maxIdle, idleTimeout int) *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     maxIdle,
		MaxActive:   0,
		IdleTimeout: time.Duration(idleTimeout),
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", instance.Host()+":"+
				strconv.Itoa(int(instance.Port())), redis.DialPassword(kvPasswd))
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
	return pool
}

/**
 * @brief 更新ckv连接池中的节点
 * 重新建立ckv连接
 * 对业务无影响
 */
func (p *Pool) Update(newKvInstances []*model.Instance) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	change := len(newKvInstances) - int(atomic.LoadInt32(&p.instanceNum))
	log.Infof("[ckv] update, old ins num:%d, new ins num:%d, change:%d", p.instanceNum, len(newKvInstances), change)

	// 新建一个pool.instances数组
	var instances []*Instance
	for _, instance := range newKvInstances {
		instance := &Instance{
			redisPool: genRedisPool(p.meta.insConnNum, p.meta.kvPasswd, instance, p.meta.MaxIdle, p.meta.IdleTimeout),
			stopCh:    make(chan bool),
		}
		instance.ch = make([]chan *Task, 0, 100*p.meta.insConnNum)
		for i := 0; i < 100*p.meta.insConnNum; i++ {
			instance.ch = append(instance.ch, make(chan *Task))
		}
		instance.index = uint32(rand.Intn(100 * p.meta.insConnNum))
		instances = append(instances, instance)
	}

	// 关闭前一个连接池
	for i := 0; i < len(p.instances); i++ {
		close(p.instances[i].stopCh)
		time.Sleep(10 * time.Millisecond)
		for j := 0; j < len(p.instances[i].ch); j++ {
			close(p.instances[i].ch[j])
		}
		err := p.instances[i].redisPool.Close()
		if err != nil {
			log.Errorf("close redis pool :%s", err)
		}
	}

	time.Sleep(10 * time.Millisecond)
	// 结构体属性重新赋值，并重新开始消费
	p.instances = instances
	atomic.StoreInt32(&p.instanceNum, int32(len(p.instances)))

	for i := 0; i < len(p.instances); i++ {
		for k := 0; k < len(p.instances[i].ch); k++ {
			go p.worker(i, k)
		}
	}

	log.Infof("[redis] update success, node num:%d", len(p.instances))

	return nil
}

func (p *Pool) checkHasKvInstances(ch chan *Resp) bool {
	if atomic.LoadInt32(&p.instanceNum) ==  0 {
		go func() {
			ch <- &Resp{
				Local: true,
			}
		}()
		return true
	}
	return false
}

/**
 * @brief 使用连接池，向redis发起Get请求
 */
func (p *Pool) Get(id string, ch chan *Resp) { // nolint
	if p.checkHasKvInstances(ch) {
		return
	}
	task := &Task{
		taskType: Get,
		id:       id,
		respCh:   ch,
	}

	insIndex, chIndex := p.genInsChIndex(id)
	p.instances[insIndex].ch[chIndex] <- task
}

/**
 * @brief 使用连接池，向redis发起Set请求
 */
func (p *Pool) Set(id string, status int, beatTime int64, ch chan *Resp) { // nolint
	if p.checkHasKvInstances(ch) {
		return
	}
	task := &Task{
		taskType: Set,
		id:       id,
		status:   status,
		beatTime: beatTime,
		respCh:   ch,
	}

	insIndex, chIndex := p.genInsChIndex(id)
	p.instances[insIndex].ch[chIndex] <- task
}

/**
 * @brief 使用连接池，向redis发起Del请求
 */
func (p *Pool) Del(id string, ch chan *Resp) { // nolint
	task := &Task{
		taskType: Del,
		id:       id,
		respCh:   ch,
	}

	insIndex, chIndex := p.genInsChIndex(id)
	p.instances[insIndex].ch[chIndex] <- task
}

/**
 * @brief 生成index公共方法
 */
func (p *Pool) genInsChIndex(id string) (int, uint32) {
	insIndex := String(id) % int(atomic.LoadInt32(&p.instanceNum))

	chIndex := atomic.AddUint32(&p.instances[insIndex].index, 1) % uint32(p.meta.insConnNum*100)
	return insIndex, chIndex
}

/**
 * @brief 启动ckv连接池工作
 */
func (p *Pool) Start() {
	p.mu.Lock()
	for i := 0; i < len(p.instances); i++ {
		for k := 0; k < len(p.instances[i].ch); k++ {
			go p.worker(i, k)
		}
	}
	p.mu.Unlock()
	log.Infof("[redis] redis pool start")
}

/**
 * @brief 接收任务worker
 */
func (p *Pool) worker(instanceIndex, chIndex int) {
	for {
		select {
		case task := <-p.instances[instanceIndex].ch[chIndex]:
			p.handleTask(task, instanceIndex)
		case <-p.instances[instanceIndex].stopCh:
			return
		}
	}
}

/**
 * @brief 任务处理函数
 */
func (p *Pool) handleTask(task *Task, index int) {
	if task == nil {
		log.Errorf("receive nil task")
		return
	}
	con := p.instances[index].redisPool.Get()
	defer con.Close()
	var resp Resp
	switch task.taskType {
	case Get:
		value, err := redis.String(con.Do("GET", task.id))
		if err != nil {
			resp.Err = err
		} else {
			resp.Value = value
		}
		task.respCh <- &resp
	case Set:
		value := fmt.Sprintf("%d:%d:%s", task.status, task.beatTime, p.meta.localHost)
		_, err := con.Do("SET", task.id, value)
		if err != nil {
			resp.Err = err
		}
		task.respCh <- &resp
	case Del:
		_, err := con.Do("DEL", task.id)
		if err != nil {
			resp.Err = err
		}
		task.respCh <- &resp
	default:
		log.Errorf("[ckv] set key:%s type:%d wrong", task.id, task.taskType)
	}
}

// 字符串转hash值
func String(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	return 0
}
