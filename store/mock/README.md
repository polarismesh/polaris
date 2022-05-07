# 对Store的API的mock

## mock文件生成方法

在`./store`目录执行

```
mockgen -source=api.go -aux_files github.com/polarismesh/polaris-server/store=config_file_api.go,github.com/polarismesh/polaris-server/store=discover_api.go,github.com/polarismesh/polaris-server/store=auth_api.go -destination=mock/api_mock.go -package=mock
```
