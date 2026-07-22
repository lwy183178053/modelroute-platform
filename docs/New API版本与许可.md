# New API 版本与许可记录

## 固定版本

- 镜像：`calciumion/new-api:v1.0.0-rc.21`
- 多架构镜像摘要：`sha256:428018a37c0b26c163a3367c18401161707cd0e08d0f26a3dde9ff0caa05e34c`
- 对应源码提交：`bde9b2f44887d34ec54799ae191d50f97914359e`
- 源码地址：`https://github.com/QuantumNous/new-api`

Compose 同时固定标签和摘要，生产升级必须先备份两个数据库，再用同一套适配契约测试验证候选版本。

## 候选版本审计

2026-07-22 实时读取上游标签后，最高正式标签仍为 `v1.0.0-rc.21`，没有高于生产固定版本的新候选标签：

- 标签对象：`8a784d7c8f8da9041b30c6c6ea4c3b3df0d683de`
- 解引用源码提交：`bde9b2f44887d34ec54799ae191d50f97914359e`

因此本阶段没有执行虚构的跨版本升级，而是把 `rc.21` 作为候选基线运行真实容器契约。默认执行：

```sh
deploy/scripts/verify-newapi-contract.sh
```

未来出现新版本时，通过环境变量传入完整镜像标签和摘要，复用同一套契约：

```sh
NEW_API_CANDIDATE_IMAGE=REGISTRY/NEW_API:TAG@sha256:DIGEST \
  deploy/scripts/verify-newapi-contract.sh
```

契约环境与生产 Compose 隔离，覆盖初始化、账号、令牌、渠道、模型映射、倍率、Chat、Responses、Embeddings、流式响应、缓存计费日志、退款日志结构和无渠道错误，结束后删除临时卷。

## 服务边界

New API 作为独立内网容器运行。ModelRoute 前端和公网网关均不开放 New API 原生 UI、管理接口和容器端口。

## 许可

New API 使用 AGPL-3.0。部署页面和运维文档保留其项目名称、许可证和对应版本源码链接；若未来形成 New API 源码改动，必须保存补丁并提供对应源码。
