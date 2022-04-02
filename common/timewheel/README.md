# timewheel

为心跳上报定制实现的、线程安全的多层时间轮，简化功能以追求更好的性能。

基于链表和ticker实现。 只有插入操作：插入任务后就必须执行。

## 测试/压测记录

覆盖率： 90%+

1. 功能测试：正常

2. 压力测试： 16c 16g机器

   开启5w个协程并发往同一个slot加10w个任务，单个操作用时280ns

   ```
   goos: windows
   goarch: amd64
   pkg: github.com/polarismesh/polaris-server/common/timewheel
   BenchmarkAddTask1-8   	  100000	       280 ns/op	     103 B/op	       2 allocs/op
   PASS
   ```

   开启16w个协程并发往同一个slot加500w个任务，单个操作用时376ns

   ```
   goos: windows
   goarch: amd64
   pkg: github.com/polarismesh/polaris-server/common/timewheel
   BenchmarkAddTask1-8   	 5000000	       376 ns/op	      97 B/op	       2 allocs/op
   PASS
   ```

   对比nosix/timewheel：
   ```
   goos: linux
   goarch: amd64
   BenchmarkAddTask1-8   	  100000	      2021 ns/op	     721 B/op	       3 allocs/op
   --- BENCH: BenchmarkAddTask1-8
       timewheel_test.go:40: N:100000, use time:255.514068ms
       timewheel_test.go:40: N:100000, use time:324.360456ms
       timewheel_test.go:40: N:100000, use time:402.377702ms
       timewheel_test.go:40: N:100000, use time:419.118132ms
       timewheel_test.go:40: N:100000, use time:195.719517ms
       timewheel_test.go:40: N:100000, use time:215.3815ms
       timewheel_test.go:40: N:100000, use time:176.733241ms
       timewheel_test.go:40: N:100000, use time:188.846803ms
       timewheel_test.go:40: N:100000, use time:164.038559ms
       timewheel_test.go:40: N:100000, use time:200.684048ms
   ```