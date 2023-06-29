# envoy outlier detection and health check

frontend 服务定时向 backend 服务发送http 请求，默认会正常返回，可以给backend 某些pod 设置异常让其返回 http 500 模拟异常

- 设置异常
  `curl http://localhost:8090/fail`

- 异常恢复
  `curl http://localhost:8090/success`

## 安装

- 构建镜像： 修改 build.sh repository 为可用的镜像仓库地址  ./build.sh 生成image

- 创建namespace 并设置可注入
  ```bash
  kubectl create namespace polaris-test
  kubectl label namespace polaris-test polaris-injection=enabled 
  ```
- 安装服务
  ```bash
  kubectl apply -f outlier.yaml
  ```