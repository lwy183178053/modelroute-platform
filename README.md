# ModelRoute API 聚合平台

ModelRoute 是一个统一面向普通用户、商户和管理员的模型 API 聚合交易平台。

## 当前结构

- `apps/web`：已确认视觉方向的 React 前端。
- `apps/api`：外围平台 Go 接口与模型路由网关。
- `deploy`：Docker Compose、网关和部署脚本。
- `docs`：中文设计、实施、部署和验证文档。

New API 作为内网服务负责账号、API 密钥、余额、模型转发、缓存计费、退款和权威用量日志。ModelRoute 负责商户审核、报价路由、渠道健康、易支付、抽成、人工提现和统一 UI。

## 开发原则

- New API 固定版本运行，源码保持原样。
- 密钥、证书、生产环境变量和运行数据不得提交到 Git。
- 每个阶段完成实现、验证、文档和 `progress.md` 记录后再进入下一阶段。
