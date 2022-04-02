# 统计服务发现请求到本地

## 测试

1. 功能测试： 正常
2. 压力测试： 存入服务数量及写入文件耗时：
    ```
    === RUN   TestWriteFile
           local_test.go:52: total num is 250000, duration is 175.246112ms
           local_test.go:52: total num is 500000, duration is 361.50121ms
           local_test.go:52: total num is 1000000, duration is 842.463092ms
           local_test.go:52: total num is 1500000, duration is 1.305463751s
       --- PASS: TestWriteFile (5.66s)
       PASS
       
       Process finished with exit code 0
   ```

   写入channel测试： channel大小为1024，并发1000，可以支持10w/s的AddDiscoverCall请求
   