# gos3browser

基于"AWS SDK for Go v2" + "gin"开发的支持s3协议存储对象浏览器demo。

## 使用方法

1.修改config.yaml

填入正确的aws_access_key_id，aws_secret_access_key及endpoint

```yaml
# 后端配置
gin:
  host: "127.0.0.1"
  port: 8080
  # debug release
  app_mode: "release"
# s3配置
s3:
  aws_access_key_id: ""
  aws_secret_access_key: ""
  endpoint: ""
```

2.运行demo

> go run main.go

浏览器打开http://127.0.0.1:8080/