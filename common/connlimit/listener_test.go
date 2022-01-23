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
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/polarismesh/polaris-server/common/connlimit/mock_net"
	. "github.com/smartystreets/goconvey/convey"
)

// TestConnLimit 模拟一下连接限制
func TestConnLimit(t *testing.T) {
	addr := "127.0.0.1:44444"
	host := "127.0.0.1"
	config := &Config{
		OpenConnLimit:        true,
		MaxConnPerHost:       5,
		MaxConnLimit:         3,
		PurgeCounterInterval: time.Hour,
		PurgeCounterExpire:   time.Minute,
	}
	connCount := 100
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("%s", err)
	}

	lis, err = NewListener(lis, "tcp", config)
	if err != nil {
		t.Fatalf("%s", err)
	}

	if lis.(*Listener).GetHostConnCount(host) != 0 {
		t.Fatalf("%s connNum should be 0 when no connections", host)
	}

	// 启动Server
	go func() {
		for {
			conn, _ := lis.Accept()
			go func(c net.Conn) {
				buf := make([]byte, 10)
				if _, err := c.Read(buf); err != nil {
					t.Logf("server read err: %s", err.Error())
					_ = c.Close()
					return
				}
				t.Logf("server read data: %s", string(buf))
				time.Sleep(time.Millisecond * 200)
				_ = c.Close()
			}(conn)
		}
	}()
	time.Sleep(1 * time.Second)

	var total int32
	for i := 0; i < connCount; i++ {
		go func(index int) {
			conn, err := net.Dial("tcp", addr)
			atomic.AddInt32(&total, 1)
			if err != nil {
				t.Logf("client conn server error: %s", err.Error())
				return
			}
			buf := []byte("hello")
			if _, err := conn.Write(buf); err != nil {
				t.Logf("client write error: %s", err.Error())
				_ = conn.Close()
				return
			}
		}(i)
	}

	// 等待连接全部关闭
	// time.Sleep(5 * time.Second)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if atomic.LoadInt32(&total) != int32(connCount) {
			t.Logf("connection is not finished")
			continue
		}
		hostCnt := lis.(*Listener).GetHostConnCount(host)
		lisCnt := lis.(*Listener).GetListenerConnCount()
		if hostCnt == 0 && lisCnt == 0 {
			t.Logf("pass")
			return
		}

		t.Logf("host conn count:%d: lis conn count:%d", hostCnt, lisCnt)
	}

}

// test readTimeout场景
/*func TestConnLimiterReadTimeout(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:55555")
	if err != nil {
		t.Fatalf("%s", err)
	}

	cfg := &Config{
		OpenConnLimit:  true,
		MaxConnLimit:   16,
		MaxConnPerHost: 8,
		ReadTimeout:    time.Millisecond * 500,
	}
	lis, err = NewListener(lis, "http", cfg)
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	defer lis.Close()
	handler := func(conn net.Conn) {
		for {
			reader := bufio.NewReader(conn)
			buf := make([]byte, 12)
			if _, err := io.ReadFull(reader, buf); err != nil {
				t.Logf("read full return: %s", err.Error())
				if e, ok := err.(net.Error); ok && e.Timeout() {
					t.Logf("pass")
				} else {
					t.Fatalf("error")
				}
				return
			}
			t.Logf("%s", string(buf))
			go func() {conn.Close()}()
		}
	}
	go func() {
		conn, err := lis.Accept()
		if err != nil {
			t.Fatalf("error: %s", err.Error())
		}
		go handler(conn)
	}()

	conn, err := net.Dial("tcp", "127.0.0.1:55555")
	if err != nil {
		t.Fatalf("error: %s", err.Error())
	}
	//time.Sleep(time.Second * 1)
	_, err = conn.Write([]byte("hello world!"))
	if err != nil {
		t.Logf("%s", err.Error())
	}
	time.Sleep(time.Second)
	conn.Close()
	time.Sleep(time.Second)
}*/

// TestInvalidParams test invalid conn limit param
func TestInvalidParams(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:44445")
	if err != nil {
		t.Fatalf("%s", err)
	}
	defer func() { _ = lis.Close() }()
	config := &Config{
		OpenConnLimit:        true,
		MaxConnPerHost:       0,
		MaxConnLimit:         10,
		PurgeCounterInterval: time.Hour,
		PurgeCounterExpire:   time.Minute,
	}

	t.Run("host连接限制小于1", func(t *testing.T) {
		if _, newErr := NewListener(lis, "tcp", config); newErr == nil {
			t.Fatalf("must be wrong for invalidMaxConnNum")
		}
	})
	t.Run("protocol为空", func(t *testing.T) {
		config.MaxConnPerHost = 10
		if _, err := NewListener(lis, "", config); err == nil {
			t.Fatalf("error")
		} else {
			t.Logf("%s", err.Error())
		}
	})
	t.Run("purge参数错误", func(t *testing.T) {
		config.PurgeCounterInterval = 0
		if _, err := NewListener(lis, "tcp1", config); err == nil {
			t.Fatalf("error")
		}
		config.PurgeCounterInterval = time.Hour
		config.PurgeCounterExpire = 0
		if _, err := NewListener(lis, "tcp2", config); err == nil {
			t.Fatalf("error")
		} else {
			t.Logf("%s", err.Error())
		}
	})
}

// TestListener_Accept 测试accept
func TestListener_Accept(t *testing.T) {
	Convey("正常accept", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		addr := mock_net.NewMockAddr(ctrl)
		conn := mock_net.NewMockConn(ctrl)
		conn.EXPECT().Close().Return(nil).AnyTimes()
		addr.EXPECT().String().Return("1.2.3.4:8080").AnyTimes()
		conn.EXPECT().RemoteAddr().Return(addr).AnyTimes()
		lis := NewTestLimitListener(100, 10)
		So(lis.accept(conn).(*Conn).isValid(), ShouldBeTrue)
	})
}

// TestLimitListener_Acquire 测试acquire
func TestLimitListener_Acquire(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conn := mock_net.NewMockConn(ctrl)
	conn.EXPECT().Close().Return(nil).AnyTimes()
	Convey("acquire测试", t, func() {
		Convey("超过server监听的最大限制，返回false", func() {
			lis := &Listener{maxConnPerHost: 1, maxConnLimit: 10, connCount: 10}
			c := lis.acquire(conn, "1.2.3.4:8080", "1.2.3.4")
			So(c.isValid(), ShouldBeFalse)
		})
		Convey("host首次请求，可以正常获取连接", func() {
			lis := NewTestLimitListener(100, 10)
			c := lis.acquire(conn, "2.3.4.5:8080", "2.3.4.5")
			So(c.isValid(), ShouldBeTrue)
		})
		Convey("host多次获取，正常", func() {
			lis := NewTestLimitListener(15, 10)
			for i := 0; i < 10; i++ {
				So(lis.acquire(conn, fmt.Sprintf("1.2.3.4:%d", i), "1.2.3.4").isValid(), ShouldBeTrue)
			}
			So(lis.acquire(conn, fmt.Sprintf("1.2.3.4:%d", 20), "1.2.3.4").isValid(), ShouldBeFalse)

			// 其他host没有超过限制，true
			So(lis.acquire(conn, fmt.Sprintf("1.2.3.9:%d", 200), "1.2.3.9").isValid(), ShouldBeTrue)
			// 占满listen的最大连接，前面成功了11个，剩下4个还没有满
			for i := 0; i < 4; i++ {
				So(lis.acquire(conn, fmt.Sprintf("1.2.3.8:%d", i), "1.2.3.8").isValid(), ShouldBeTrue)
			}

			// 总连接数被占满，false
			So(lis.acquire(conn, fmt.Sprintf("1.2.3.19:%d", 123), "1.2.3.9").isValid(), ShouldBeFalse)
		})
	})
}

// TestLimitListener_ReLease release
func TestLimitListener_ReLease(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conn := mock_net.NewMockConn(ctrl)
	conn.EXPECT().Close().Return(nil).AnyTimes()
	t.Run("并发释放测试", func(t *testing.T) {
		lis := NewTestLimitListener(2048000, 204800)
		conns := make([]net.Conn, 0, 10240)
		for i := 0; i < 10240; i++ {
			c := lis.acquire(conn, "1.2.3.4:8080", "1.2.3.4")
			conns = append(conns, c)
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10240; i++ {
				lis.acquire(conn, "1.2.3.4:8080", "1.2.3.4")
			}
		}()

		for i := 0; i < 2048; i++ {
			wg.Add(1)
			go func(index int) {
				for j := 0; j < 5; j++ {
					c := conns[index*5+j]
					_ = c.Close()
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
		var remain int32 = 10240 + 10240 - 2048*5
		if lis.GetListenerConnCount() == remain && lis.GetHostConnCount("1.2.3.4") == remain {
			t.Logf("pass")
		} else {
			t.Fatalf("error: %d, %d", lis.GetListenerConnCount(), lis.GetHostConnCount("1.2.3.4"))
		}
	})
}

// TestWhiteList 白名单测试
func TestWhiteList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conn := mock_net.NewMockConn(ctrl)
	conn.EXPECT().Close().Return(nil).AnyTimes()

	Convey("白名单下，限制不生效", t, func() {
		listener := NewTestLimitListener(100, 2)
		listener.whiteList = map[string]bool{
			"8.8.8.8": true,
		}
		for i := 0; i < 100; i++ {
			So(listener.acquire(conn, "8.8.8.8:123", "8.8.8.8").isValid(), ShouldBeTrue)
		}
		// 超过了机器的100限制，白名单也不放过
		So(listener.acquire(conn, "8.8.8.8:123", "8.8.8.8").isValid(), ShouldBeFalse)
		So(listener.acquire(conn, "8.8.8.9:123", "8.8.8.9").isValid(), ShouldBeFalse)
		So(listener.acquire(conn, "8.8.8.10:123", "8.8.8.10").isValid(), ShouldBeFalse)
	})
}

// TestActiveConns 测试activeConns
func TestActiveConns(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	conn := mock_net.NewMockConn(ctrl)
	conn.EXPECT().Close().Return(nil).AnyTimes()
	listener := NewTestLimitListener(1024, 64)
	var conns []*Conn
	Convey("初始化", t, func() {
		for i := 0; i < 32; i++ {
			c := listener.acquire(conn, fmt.Sprintf("8.8.8.8:%d", i), "8.8.8.8")
			So(c.isValid(), ShouldBeTrue)
			conns = append(conns, c)
		}
	})
	Convey("测试活跃连接", t, func() {
		Convey("已活跃的连接可以正常存储", func() {
			actives := listener.GetHostActiveConns("8.8.8.8")
			So(actives, ShouldNotBeNil)
			So(len(actives), ShouldEqual, 32)
		})
		Convey("连接关闭，活跃连接map会剔除", func() {
			for i := 0; i < 8; i++ {
				_ = conns[i].Close()
			}
			actives := listener.GetHostActiveConns("8.8.8.8")
			So(actives, ShouldNotBeNil)
			So(len(actives), ShouldEqual, 24) // 32 - 8
		})
		Convey("重复关闭连接，活跃连接map不受影响，size不受影响", func() {
			for i := 0; i < 8; i++ {
				_ = conns[i].Close()
			}
			actives := listener.GetHostActiveConns("8.8.8.8")
			So(actives, ShouldNotBeNil)
			So(len(actives), ShouldEqual, 24)
			So(listener.GetHostConnCount("8.8.8.8"), ShouldEqual, 24)
		})
		Convey("多主机数据，可以正常存储", func() {
			for i := 0; i < 16; i++ {
				c := listener.acquire(conn, fmt.Sprintf("8.8.8.16:%d", i), "8.8.8.16")
				So(c.isValid(), ShouldBeTrue)
				conns = append(conns, c)
			}
			actives := listener.GetHostActiveConns("8.8.8.16")
			So(actives, ShouldNotBeNil)
			So(len(actives), ShouldEqual, 16)
		})
	})
}

// TestPurgeExpireCounterHandler 测试回收过期Counter函数
func TestPurgeExpireCounterHandler(t *testing.T) {
	Convey("可以正常purge", t, func() {
		listener := NewTestLimitListener(1024, 16)
		listener.purgeCounterExpire = 3
		for i := 0; i < 102400; i++ {
			ct := newCounter()
			ct.size = 0
			listener.conns.Store(fmt.Sprintf("127.0.0.:%d", i), ct)
		}
		time.Sleep(time.Second * 4)
		for i := 0; i < 102400; i++ {
			ct := newCounter()
			ct.size = 0
			listener.conns.Store(fmt.Sprintf("127.0.1.%d", i), ct)
		}
		So(listener.GetDistinctHostCount(), ShouldEqual, 204800)
		listener.purgeExpireCounterHandler()
		So(listener.GetDistinctHostCount(), ShouldEqual, 102400)
	})
	Convey("并发store和range，扫描的速度测试", t, func() {
		listener := NewTestLimitListener(1024, 16)
		listener.purgeCounterInterval = time.Microsecond * 10
		listener.purgeCounterExpire = 1
		rand.Seed(time.Now().UnixNano())
		ctx, cancel := context.WithCancel(context.Background())
		for i := 0; i < 10240; i++ {
			go func(index int) {
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					for j := 0; j < 100; j++ {
						ct := newCounter()
						ct.size = 0
						listener.conns.Store(fmt.Sprintf("%d.%d", index, j), ct)
						time.Sleep(time.Millisecond)
					}
				}

			}(i)
		}
		listener.purgeExpireCounter(ctx)
		<-time.After(time.Second * 5)
		cancel()
	})
}

// NewTestLimitListener 返回一个测试listener
func NewTestLimitListener(maxLimit int32, hostLimit int32) *Listener {
	return &Listener{
		maxConnLimit:         maxLimit,
		maxConnPerHost:       hostLimit,
		purgeCounterInterval: time.Hour,
		purgeCounterExpire:   300,
	}
}
