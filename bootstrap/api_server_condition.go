package bootstrap

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/polarismesh/polaris/common/log"
)

// ApiServerCondition 启动条件
type ApiServerCondition struct {
	cond *sync.Cond
	cnt  int // 已经成功listen的ApiServer的数量
}

// NewApiServerCond 初始化一个 ApiServerCondition指针
func NewApiServerCond() *ApiServerCondition {
	fmt.Printf("NewApiServerCond \n")
	var condLock sync.Mutex
	return &ApiServerCondition{cond: sync.NewCond(&condLock)}
}

// Incr 增加Condition的Cnt
func (cc *ApiServerCondition) Incr(serverName string) {
	fmt.Printf("ApiServerCondition1 -> server:%s Cnt:%d \n", serverName, cc.cnt)
	cc.cond.L.Lock()
	cc.cnt++
	cc.cond.L.Unlock()
	fmt.Printf("ApiServerCondition2 -> server:%s Cnt:%d \n", serverName, cc.cnt)
	cc.cond.Broadcast()
}

func (cc *ApiServerCondition) Set(i int) {
	fmt.Printf("ApiServerCondition1 -> Set:%d Cnt:%d \n", i, cc.cnt)
	cc.cond.L.Lock()
	cc.cnt = i
	cc.cond.L.Unlock()
	fmt.Printf("ApiServerCondition2 -> Set:%d Cnt:%d \n", i, cc.cnt)
	cc.cond.Broadcast()
}

func (cc *ApiServerCondition) GetCnt() int {
	fmt.Printf("ApiServerCondition->GetCnt cnt:%d \n", cc.cnt)
	return cc.cnt

}

func (cc *ApiServerCondition) Wait(cnt int) error {
	fmt.Printf("ApiServerCondition->wait cnt:%d \n", cnt)
	cc.cond.L.Lock()

	// 超时
	time.AfterFunc(3*time.Second, func() {
		if cc.cnt < cnt {
			log.Warnf("ApiServerCondition->wait timeout, current cnt:%d \n", cc.cnt)
			cc.Set(cnt)
		}
	})

	for cc.cnt < cnt {
		log.Infof("BeforeApiServerCondition->wait cnt:%d \n", cc.cnt)
		cc.cond.Wait()
		log.Infof("AfterApiServerCondition->wait cnt:%d \n", cc.cnt)
	}
	cc.cond.L.Unlock()

	if cc.cnt > cnt {
		log.Infof("[ERROR] ApiServerCondition timeout \n")
		return errors.New("[ERROR] ApiServerCondition timeout")
	}
	return nil
}
