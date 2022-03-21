 # 对Auth的API的mock
 
 ## mock文件生成方法
 在`./store`目录执行
 ```
 mockgen -source=api.go -destination=./mock/api_mock.go -package=mock
 ```