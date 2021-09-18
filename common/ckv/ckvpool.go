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

package ckv

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
)

/**
* 健康检查功能使用ckv+缓存实例健康状态
* 实现一个无锁连接池供健康检查功能使用
*
* 层次结构：连接池 -> ckv+节点 -> 连接
* 每个ckv+节点建立n个（n:可配置）连接
* 每个连接绑定一个chan，chan用于分发请求到此连接
* 负载均衡方式：轮询
*
* 当ckv+节点发生变动，连接池支持动态增删，业务无感知
 */

const (
	// Get get method define
	Get = iota
	// Set set method define
	Set
	// Del del method define
	Del
)

/**
 * Task ckv任务请求结构体
 */
type Task struct {
	taskType int
	id       string
	status   int
	beatTime int64
	respCh   chan *Resp
}

/**
 * Resp ckv任务结果
 */
type Resp struct {
	Value string
	Err   error
}

/**
 * MetaData ckv连接池元数据
 */
type MetaData struct {
	insConnNum int
	kvPasswd   string
	localHost  string
}

/**
 * Node ckv节点结构体
 */
type Node struct {
	index    int // 节点在连接池中的序号
	addr     string
	conns    []*Conn
	stopCh   chan struct{}
	changeCh chan struct{}
}

/**
 * Pool ckv连接池结构体
 */
type Pool struct {
	mu       sync.Mutex
	meta     *MetaData
	chs      []chan *Task
	nodes    []*Node
	connSize uint32
	index    uint32
}

/**
 * newNode 初始化一个ckv节点实例
 */
func newNode(index, connNum int, kvPasswd string, ins *model.Instance) *Node {
	// 计算连接的序号，即连接要与序号为index的chan绑定
	connIndexStart := index * connNum
	connIndexEnd := (index + 1) * connNum

	node := &Node{
		index:    index,
		addr:     fmt.Sprintf("%s:%d", ins.Host(), ins.Port()),
		stopCh:   make(chan struct{}, 1),
		changeCh: make(chan struct{}, 1),
		conns:    make([]*Conn, 0, connIndexEnd), // pre-allocated in advance
	}

	log.Infof("[ckv] instance:%s connect, conn num:%d", node.addr, connNum)
	for i := connIndexStart; i < connIndexEnd; i++ {
		conn, err := newConn(i, node.addr, kvPasswd)
		if err != nil {
			log.Errorf("[ckv] instance:%s connect failed:%s", node.addr, err)
			return nil
		}
		node.conns = append(node.conns, conn)
	}
	return node
}

/**
 * NewPool 初始化一个ckv连接池实例
 */
func NewPool(insConnNum int, kvPasswd, localHost string, kvInstances []*model.Instance) (*Pool, error) {
	kvPool := &Pool{
		meta: &MetaData{
			insConnNum: insConnNum,
			kvPasswd:   kvPasswd,
			localHost:  localHost,
		},
		connSize: uint32(len(kvInstances) * insConnNum),
	}
	err := kvPool.connect(kvInstances)
	if err != nil {
		log.Errorf("[kv] connect kv err:%s", err)
		return nil, err
	}

	rand.Seed(time.Now().Unix())
	// 从一个随机位置开始，防止所有server都从一个ckv开始
	kvPool.index = uint32(rand.Intn(int(kvPool.connSize)))
	return kvPool, nil
}

/**
 * Start 启动ckv连接池工作
 */
func (p *Pool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, node := range p.nodes {
		for _, conn := range node.conns {
			go p.worker(conn, node.changeCh, node.stopCh)
		}
	}
	log.Infof("[ckv] ckv pool start")
}

/**
 * Update 更新ckv连接池中的节点
 * 重新建立ckv连接
 * 对业务无影响
 */
func (p *Pool) Update(newKvInstances []*model.Instance) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	change := len(newKvInstances) - len(p.nodes)
	log.Infof("[ckv] update, old ins num:%d, new ins num:%d, change:%d", len(p.nodes), len(newKvInstances), change)
	newConnSize := len(newKvInstances) * p.meta.insConnNum
	oldConnSize := p.connSize
	log.Infof("[ckv] update, old conn num:%d, new conn num:%d", oldConnSize, newConnSize)

	if change > 0 {
		// ckv+节点数如果增多，则新增对应数量的chan
		for i := 0; i < change*p.meta.insConnNum; i++ {
			p.chs = append(p.chs, make(chan *Task, 100))
		}
	} else if change < 0 {
		// ckv+节点数如果减少，修改最大连接数(chan数)，防止新的请求进入需要关闭的连接(chan)
		// 等待chan中的积留请求被处理完再关闭
		p.connSize = uint32(newConnSize)
	}

	newNodes := make([]*Node, 0, len(newKvInstances))
	for index, ins := range newKvInstances {
		// 建立新连接
		node := newNode(index, p.meta.insConnNum, p.meta.kvPasswd, ins)
		if node == nil {
			return errors.New("create ckv node failed")
		}
		newNodes = append(newNodes, node)

		// 关闭原先的连接，绑定chan到新连接
		if index < len(p.nodes) {
			close(p.nodes[index].changeCh)
		}
		for _, conn := range node.conns {
			go p.worker(conn, node.changeCh, node.stopCh)
		}
	}

	if change > 0 {
		// 修改最大连接数到正确状态，启用新增的chan
		p.connSize = uint32(newConnSize)
	} else if change < 0 {
		// 如果节点数减少，关闭不需要的连接和chan
		for index := len(newKvInstances); index < len(p.nodes); index++ {
			close(p.nodes[index].stopCh)
		}
		time.Sleep(10 * time.Millisecond)
		for index := p.connSize; index < oldConnSize; index++ {
			close(p.chs[index])
		}
		p.chs = p.chs[:p.connSize]
	}
	p.nodes = newNodes
	log.Infof("[ckv] update success, node num:%d, conn num:%d, chan num:%d", len(p.nodes), len(p.chs), p.connSize)

	return nil
}

/**
 * connect 建立连接
 */
func (p *Pool) connect(kvInstances []*model.Instance) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for index, ins := range kvInstances {
		node := newNode(index, p.meta.insConnNum, p.meta.kvPasswd, ins)
		if node == nil {
			return errors.New("create ckv node failed")
		}

		for i := 0; i < len(node.conns); i++ {
			p.chs = append(p.chs, make(chan *Task, 100))
		}
		p.nodes = append(p.nodes, node)
	}
	return nil
}

/**
 * Get 使用连接池，向ckv发起Get请求
 */
func (p *Pool) Get(id string, ch chan *Resp) { // nolint
	task := &Task{
		taskType: Get,
		id:       id,
		respCh:   ch,
	}

	index := atomic.AddUint32(&p.index, 1) % p.connSize
	p.chs[index] <- task
}

/**
 * Set 使用连接池，向ckv发起Set请求
 */
func (p *Pool) Set(id string, status int, beatTime int64, ch chan *Resp) { // nolint
	task := &Task{
		taskType: Set,
		id:       id,
		status:   status,
		beatTime: beatTime,
		respCh:   ch,
	}

	index := atomic.AddUint32(&p.index, 1) % p.connSize
	p.chs[index] <- task
}

/**
 * Del 使用连接池，向ckv发起Del请求
 */
func (p *Pool) Del(id string, ch chan *Resp) { // nolint
	task := &Task{
		taskType: Del,
		id:       id,
		respCh:   ch,
	}

	index := atomic.AddUint32(&p.index, 1) % p.connSize
	p.chs[index] <- task
}

/**
 * worker 接收任务worker
 */
func (p *Pool) worker(conn *Conn, changeCh, stopCh chan struct{}) {
	ch := p.chs[conn.index]
	defer conn.conn.Close()

	for {
		select {
		case task := <-ch:
			p.handleTask(conn, task)
		case <-stopCh:
			// 发现ckv+节点变少，不再需要此chan
			// 处理滞留请求，关闭连接
			for task := range ch {
				p.handleTask(conn, task)
			}

			log.Infof("[ckv] instance:%s chan:%d close", conn.addr, conn.index)
			return
		case <-changeCh:
			// 发现ckv+节点变动，更新chan绑定的连接
			log.Infof("[ckv] instance:%s chan:%d change", conn.addr, conn.index)
			return
		}
	}
}

/**
 * handleTask 任务处理函数
 */
func (p *Pool) handleTask(conn *Conn, task *Task) {
	if task == nil {
		log.Errorf("chan:%d receive nil task", conn.index)
		return
	}

	var resp Resp
	switch task.taskType {
	case Get:
		resp.Value, resp.Err = conn.Get(task.id)
		task.respCh <- &resp
	case Set:
		value := fmt.Sprintf("%d:%d:%s", task.status, task.beatTime, p.meta.localHost)
		resp.Err = conn.Set(task.id, value)
		task.respCh <- &resp
	case Del:
		resp.Err = conn.Del(task.id)
		task.respCh <- &resp
	default:
		log.Errorf("[ckv] set key:%s type:%d wrong", task.id, task.taskType)
	}
}
