# New API 版本与许可记录

## 固定版本

- 镜像：`calciumion/new-api:v1.0.0-rc.21`
- 多架构镜像摘要：`sha256:428018a37c0b26c163a3367c18401161707cd0e08d0f26a3dde9ff0caa05e34c`
- 对应源码提交：`bde9b2f44887d34ec54799ae191d50f97914359e`
- 源码地址：`https://github.com/QuantumNous/new-api`

Compose 同时固定标签和摘要，生产升级必须先备份两个数据库，再用同一套适配契约测试验证候选版本。

## 服务边界

New API 作为独立内网容器运行。ModelRoute 前端和公网网关均不开放 New API 原生 UI、管理接口和容器端口。

## 许可

New API 使用 AGPL-3.0。部署页面和运维文档保留其项目名称、许可证和对应版本源码链接；若未来形成 New API 源码改动，必须保存补丁并提供对应源码。
