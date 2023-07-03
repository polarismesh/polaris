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

package eurekaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"

	"github.com/polarismesh/polaris/common/version"
)

type ReplicateWorker struct {
	namespace   string
	peers       []string
	taskChannel chan *ReplicationInstance
	ctx         context.Context
}

const (
	batchMaxInterval   = 5 * time.Second
	batchReplicateSize = 10
)

func NewReplicateWorker(ctx context.Context, namespace string, peers []string) *ReplicateWorker {
	worker := &ReplicateWorker{
		namespace:   namespace,
		peers:       peers,
		taskChannel: make(chan *ReplicationInstance, 1000),
		ctx:         ctx,
	}
	go worker.batchReplicate()
	return worker
}

func (r *ReplicateWorker) AddReplicateTask(task *ReplicationInstance) {
	r.taskChannel <- task
}

func (r *ReplicateWorker) batchReplicate() {
	batchTasks := make([]*ReplicationInstance, 0)
	batchTicker := time.NewTicker(batchMaxInterval)
	for {
		select {
		case <-r.ctx.Done():
			eurekalog.Infof("[EUREKA-SERVER] replicate worker done")
			batchTicker.Stop()
		case task := <-r.taskChannel:
			batchTasks = append(batchTasks, task)
			if len(batchTasks) == batchReplicateSize {
				// replicate at once when batch size reach max threshold
				r.doBatchReplicate(batchTasks)
				batchTasks = make([]*ReplicationInstance, 0)
			}
		case <-batchTicker.C:
			if len(batchTasks) > 0 {
				// replicate when time interval reached
				r.doBatchReplicate(batchTasks)
				batchTasks = make([]*ReplicationInstance, 0)
			}
		}
	}
}

func (r *ReplicateWorker) doBatchReplicate(tasks []*ReplicationInstance) {
	request := &ReplicationList{ReplicationList: make([]*ReplicationInstance, 0, len(tasks))}
	for _, task := range tasks {
		if task.Action == actionHeartbeat {
			request.ReplicationList = append(request.ReplicationList, &ReplicationInstance{
				AppName:            task.AppName,
				Id:                 task.Id,
				LastDirtyTimestamp: task.LastDirtyTimestamp,
				OverriddenStatus:   task.OverriddenStatus,
				Status:             task.Status,
				Action:             task.Action,
			})
		} else {
			request.ReplicationList = append(request.ReplicationList, task)
		}
	}
	jsonData, err := json.Marshal(request)
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to marshal replicate tasks: %v", err)
		return
	}
	replicateInfo := make([]string, 0, len(tasks))
	for _, task := range tasks {
		replicateInfo = append(replicateInfo, fmt.Sprintf("%s:%s", task.Action, task.Id))
	}
	eurekalog.Infof("start to send replicate text %s, peers %v", string(jsonData), r.peers)
	for _, peer := range r.peers {
		go r.doReplicateToPeer(peer, tasks, jsonData, replicateInfo)
	}
}

func (r *ReplicateWorker) doReplicateToPeer(
	peer string, tasks []*ReplicationInstance, jsonData []byte, replicateInfo []string) {
	response, err := sendHttpRequest(r.namespace, peer, jsonData, replicateInfo)
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to batch replicate to %s, err: %v", peer, err)
		return
	}
	if len(response.ResponseList) == 0 {
		return
	}
	for i, respInstance := range response.ResponseList {
		if respInstance.StatusCode == http.StatusNotFound {
			task := tasks[i]
			if task.Action == actionHeartbeat {
				eurekalog.Infof("[EUREKA-SERVER] instance %s of service %s not exists in %s, do register instance info %+v",
					task.Id, task.AppName, peer, task.InstanceInfo)
				// do the re-register
				registerTask := &ReplicationInstance{
					AppName:            task.AppName,
					Id:                 task.Id,
					LastDirtyTimestamp: task.LastDirtyTimestamp,
					Status:             StatusUp,
					InstanceInfo:       task.InstanceInfo,
					Action:             actionRegister,
				}
				r.AddReplicateTask(registerTask)
			}
		}
	}
}

type ReplicateWorkers struct {
	works map[string]*ReplicateWorker
}

func NewReplicateWorkers(ctx context.Context, namespacePeers map[string][]string) *ReplicateWorkers {
	works := make(map[string]*ReplicateWorker)
	for namespace, peers := range namespacePeers {
		works[namespace] = NewReplicateWorker(ctx, namespace, peers)
	}
	return &ReplicateWorkers{
		works: works,
	}
}

func (r *ReplicateWorkers) Get(namespace string) (*ReplicateWorker, bool) {
	work, exist := r.works[namespace]
	return work, exist
}

func sendHttpRequest(namespace string, peer string,
	jsonData []byte, replicateInfo []string) (*ReplicationListResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://%s/eureka/peerreplication/batch/", peer), bytes.NewBuffer(jsonData))
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail to create replicate request: %v", err)
		return nil, err
	}
	req.Header.Set(headerIdentityName, valueIdentityName)
	req.Header.Set(headerIdentityVersion, version.Version)
	req.Header.Set(restful.HEADER_ContentType, restful.MIME_JSON)
	req.Header.Set(restful.HEADER_Accept, restful.MIME_JSON)
	if len(namespace) != 0 {
		req.Header.Set(HeaderNamespace, namespace)
	}
	response, err := client.Do(req)
	if err != nil {
		eurekalog.Errorf("[EUREKA-SERVER] fail to send replicate request: %v", err)
		return nil, err
	}
	defer func() {
		if nil != response && nil != response.Body {
			_ = response.Body.Close()
		}
	}()
	respStr, _ := io.ReadAll(response.Body)
	respObj := &ReplicationListResponse{}
	err = json.Unmarshal(respStr, respObj)
	if nil != err {
		eurekalog.Errorf("[EUREKA-SERVER] fail unmarshal text %s to ReplicationListResponse: %v", string(respStr), err)
		return nil, err
	}
	eurekalog.Infof("[EUREKA-SERVER] success to replicate to %s, instances %v", peer, replicateInfo)
	return respObj, nil
}
