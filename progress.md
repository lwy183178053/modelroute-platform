## 2026-07-22 - Task: Design the API aggregation marketplace

### What was done

- Defined the approved architecture that keeps New API unmodified and authoritative for users, balances, EasyPay, forwarding, cache-aware billing, refunds, and usage logs.
- Specified unified user and merchant roles, merchant review, channel import, immutable offer-model aliases, user routing policies, health checks, cooldowns, 10% commission, manual withdrawals, role-based UI, delivery phases, and acceptance criteria.

### Testing

- Verified the design against the inspected New API channel model mapping, per-model ratio configuration, cache-aware billing data, channel IDs, request IDs, consumption logs, refund logs, and administrator APIs.
- Confirmed the design preserves fixed-route boundaries, immutable request pricing, idempotent settlement, and manual withdrawal freezing without modifying New API source code.
- Markdown structure and placeholder checks are recorded after the final document review.

### Notes

- `docs/superpowers/specs/2026-07-22-api-marketplace-design.md`: Added the formal marketplace design and phased acceptance criteria.
- `progress.md`: Added the required append-only task record.
- The workspace does not contain a Git repository, so no commit was created.
- Rollback: run `Remove-Item -LiteralPath 'docs\superpowers\specs\2026-07-22-api-marketplace-design.md'` and `Remove-Item -LiteralPath 'progress.md'` from the workspace root.
- Final review completed: no placeholder markers were found, Markdown fences are balanced, and model-limit, probe-accounting, and pricing-source boundaries are explicit.

## 2026-07-22 - Task: 将 API 聚合平台设计规范改为中文

### What was done

- 将设计规范的标题、业务规则、架构边界、流程、状态机、交付阶段和验收标准完整改写为中文。
- 将文件名调整为中文，并保留必要的接口名、数据标识和技术名称，方便后续实现时准确引用。

### Testing

- 检查中文文档标题结构、代码围栏和关键业务规则是否完整。
- 检查文档中是否残留成段英文说明，并核对计费、路由、探活和提现规则与已确认方案一致。

### Notes

- `docs/superpowers/specs/2026-07-22-API聚合平台设计规范.md`：新增完整中文设计规范。
- `docs/superpowers/specs/2026-07-22-api-marketplace-design.md`：删除已由中文规范替代的英文版本。
- `progress.md`：追加本轮中文化记录。
- 回滚方式：恢复 `docs/superpowers/specs/2026-07-22-api-marketplace-design.md`，删除 `docs/superpowers/specs/2026-07-22-API聚合平台设计规范.md`，并移除本轮新增的进度段落。

## 2026-07-22 - Task: 调整外围支付与 New API 计费边界

### What was done

- 将充值支付调整为外围平台自有能力，新增管理员易支付配置、平台充值订单、回调验签、支付对账和异常入账重试设计。
- 保留 New API 作为用户余额和模型扣费执行核心；平台支付成功后通过内部适配层为对应 New API 用户增加额度。
- 明确支付成功与额度入账使用独立幂等控制，并继续保持商户提现人工审核。

### Testing

- 核对支付回调、额度入账、模型计费和商户结算之间只有一个明确责任方。
- 核对支付密钥只在后端加密保存，页面仅显示掩码和配置状态。
- 核对重复回调、New API 暂时异常和人工重试不会导致重复入账。

### Notes

- `docs/superpowers/specs/2026-07-22-API聚合平台设计规范.md`：补充自有易支付接口、管理员支付设置、订单状态机和验收标准。
- `progress.md`：追加本轮支付边界调整记录。
- 本轮未保存用户提供的易支付密钥，正式部署时通过管理员界面加密录入。
- 回滚方式：将中文设计规范恢复到上一版本，并移除本轮新增的进度段落。

## 2026-07-22 - Task: 实现 API 聚合平台 Web UI 原型

### What was done

- 实现模型市场、用户控制台、商户中心和管理员后台四个可切换视图，并统一普通用户、商户和管理员的官网导航。
- 实现智能/固定商户路由策略、商户申请、渠道提交、报价调整、充值和人工提现等可点击弹窗。
- 实现外围平台自有易支付设置、测试连接、验签自检、轮换密钥、回调日志、订单对账、异常入账重试、异常订单关闭和订单导出等管理员入口。
- 页面只展示支付 PID 和 KEY 掩码；完整支付密钥未写入页面、源码、构建产物或正式文档。
- 实现桌面和移动端响应式布局，并保留宽业务表格的局部横向滚动。

### Testing

- 先执行 `npm.cmd run test:ui`，确认页面未实现时因找不到“模型服务市场”标题而失败，再补实现并完成红绿验证。
- 执行 `npm.cmd run build`，Vite 生产构建退出码为 `0`，共转换 `1603` 个模块。
- 执行 `npm.cmd run test:ui`，四个主视图、固定路由、商户调价、管理员支付设置和测试连接交互全部通过。
- 执行 `npm.cmd run capture`，生成 `8` 张业务视图和 `4` 张关键弹窗截图；脚本确认桌面 `1440x1000` 与移动端 `390x844` 根页面无横向溢出、无运行时错误和空白页面。
- 人工检查模型市场、控制台、商户中心、管理员支付页及路由/支付弹窗截图，未发现文字遮挡、控件重叠或空白区域异常。
- 扫描仓库文本文件，确认用户提供的完整易支付 KEY 不存在于源码、测试、文档和构建产物中。

### Notes

- `docs/superpowers/plans/2026-07-22-UI原型实施计划.md`：新增并完成中文原型实施清单。
- `ui-preview/index.html`：提供 Vite 原型挂载入口和中文页面元数据。
- `ui-preview/package.json`：增加构建、UI 冒烟和截图命令。
- `ui-preview/package-lock.json`：锁定 React、Vite、Lucide、模型图标和 Playwright 依赖版本。
- `ui-preview/vite.config.js`：启用 React JSX 转换插件。
- `ui-preview/src/main.jsx`：实现统一导航、主视图切换、弹窗和消息状态入口。
- `ui-preview/src/data.js`：提供模型报价、渠道、调用、订单、审核和提现原型数据。
- `ui-preview/src/ui.jsx`：提供按钮、状态、表格区块、字段、弹窗和模型图标等共用组件。
- `ui-preview/src/Modals.jsx`：实现路由、报价、支付、充值、商户申请、渠道和提现弹窗。
- `ui-preview/src/styles.css`：实现紧凑业务界面、状态色、桌面布局和移动端适配。
- `ui-preview/src/views/MarketView.jsx`：实现模型市场筛选、厂商分类和商户报价排行。
- `ui-preview/src/views/ConsoleView.jsx`：实现用户余额、用量、API 密钥路由和 New API 最终消费记录。
- `ui-preview/src/views/MerchantView.jsx`：实现商户渠道、探活、报价、收入拆分和人工提现入口。
- `ui-preview/src/views/AdminView.jsx`：实现管理员导航、运营总览、平台设置和审计日志。
- `ui-preview/src/views/admin/AdminMerchants.jsx`：实现商户验证、通过和驳回操作。
- `ui-preview/src/views/admin/AdminChannels.jsx`：实现渠道探活、冷却解除和停用操作。
- `ui-preview/src/views/admin/AdminPayment.jsx`：实现易支付配置、订单入账、回调和异常处理操作。
- `ui-preview/src/views/admin/AdminWithdrawals.jsx`：实现提现账本核对、审核、打款确认和驳回操作。
- `ui-preview/scripts/ui-smoke.mjs`：自动验证主要视图与业务交互，并检查 KEY 明文模式。
- `ui-preview/scripts/capture.mjs`：自动截取桌面/移动端主页面和弹窗并检查页面溢出与运行时错误。
- `ui-preview/artifacts/market-desktop.png`、`market-mobile.png`：模型市场桌面与移动端评审图。
- `ui-preview/artifacts/console-desktop.png`、`console-mobile.png`：用户控制台桌面与移动端评审图。
- `ui-preview/artifacts/merchant-desktop.png`、`merchant-mobile.png`：商户中心桌面与移动端评审图。
- `ui-preview/artifacts/admin-desktop.png`、`admin-mobile.png`：管理员支付页桌面与移动端评审图。
- `ui-preview/artifacts/route-modal-desktop.png`、`route-modal-mobile.png`：路由策略弹窗桌面与移动端评审图。
- `ui-preview/artifacts/payment-config-desktop.png`、`payment-config-mobile.png`：支付配置弹窗桌面与移动端评审图。
- `ui-preview/dist/index.html`、`dist/assets/index-CGdS1xsR.css`、`dist/assets/index-CjuzvYOb.js`：Vite 生产构建输出。
- `ui-preview/vite.out.log`：记录首次开发服务启动与端口切换信息。
- `ui-preview/vite.err.log`：保留首次开发服务标准错误输出，当前为空。
- `ui-preview/vite-preview.out.log`：记录当前本地预览服务启动信息。
- `ui-preview/vite-preview.err.log`：保留当前预览服务标准错误输出，当前为空。
- `progress.md`：追加本轮实现、验证、改动文件和回滚记录。
- 回滚方式：在工作区根目录执行 `Remove-Item -LiteralPath 'ui-preview' -Recurse` 和 `Remove-Item -LiteralPath 'docs\superpowers\plans\2026-07-22-UI原型实施计划.md'`，再以本标题为回滚点移除 `progress.md` 最后一节。

## 2026-07-22 - Task: 建立 ModelRoute 正式工程与部署审计基线

### What was done

- 只读审计部署主机，确认五个公网地址属于同一台 Ubuntu 主机，103 上已有 Sub2API 且 80/443 由宿主机 Caddy 使用。
- 确定 ModelRoute 使用独立目录、Compose 项目、数据库和本机端口，由现有 Caddy 按 `aggregate.lwylink.xyz` 转发。
- 从 Sub2API 数据库只读确认 SMTP 主机、端口、TLS、发件人及密码已配置状态，未输出 SMTP 密码。
- 将已认可原型机械复制并校验到正式目录 `apps/web`，建立根目录说明、忽略规则和中文实施文档。
- 修复 UI 冒烟与截图脚本对手工预览服务的依赖，使脚本在干净环境自行启动和关闭 Vite 预览服务。
- 初始化 Git 主分支基线，施工代码将在独立分支和工作树中继续。

### Testing

- 在原型目录执行 `npm.cmd run build`，Vite 构建成功并转换 1603 个模块。
- 在原型目录执行 `npm.cmd run test:ui`，四个视图和关键交互通过。
- 对机械复制的 31 个正式前端文件逐个执行 SHA-256 比对，缺失和不一致数量均为零。
- 在 `apps/web` 执行 `npm.cmd ci`，安装 594 个包，安全审计结果为 0 个漏洞。
- 首次在干净环境执行 `npm.cmd run test:ui`，按预期因 4173 未启动失败；补充自启动预览后再次执行通过。
- 在 `apps/web` 执行 `npm.cmd run build`，构建退出码为 0。

### Notes

- `.gitignore`：排除依赖、构建产物、日志、环境密钥、证书、运行数据、工作树和旧原型副本。
- `README.md`：说明正式目录结构、New API 边界和阶段交付要求。
- `docs/部署环境审计.md`：记录主机、网关、隔离落点和 SMTP 复用结论。
- `docs/superpowers/plans/2026-07-22-ModelRoute完整实施计划.md`：保存完整中文阶段实施计划。
- `apps/web/index.html`：迁入正式 React 页面入口。
- `apps/web/package.json`：迁入前端依赖和验证命令。
- `apps/web/package-lock.json`：迁入并锁定前端依赖版本。
- `apps/web/vite.config.js`：迁入 Vite React 配置。
- `apps/web/src/main.jsx`：迁入统一官网导航和页面挂载逻辑。
- `apps/web/src/data.js`：迁入原型业务数据。
- `apps/web/src/ui.jsx`：迁入通用 UI 组件。
- `apps/web/src/Modals.jsx`：迁入路由、支付、商户和提现弹窗。
- `apps/web/src/styles.css`：迁入已认可的响应式视觉样式。
- `apps/web/src/views/MarketView.jsx`：迁入模型市场页面。
- `apps/web/src/views/ConsoleView.jsx`：迁入用户控制台页面。
- `apps/web/src/views/MerchantView.jsx`：迁入商户中心页面。
- `apps/web/src/views/AdminView.jsx`：迁入管理员后台框架。
- `apps/web/src/views/admin/AdminChannels.jsx`：迁入渠道管理页面。
- `apps/web/src/views/admin/AdminMerchants.jsx`：迁入商户审核页面。
- `apps/web/src/views/admin/AdminPayment.jsx`：迁入支付管理页面。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：迁入提现审核页面。
- `apps/web/scripts/local-preview.mjs`：新增测试用 Vite 预览服务生命周期管理。
- `apps/web/scripts/ui-smoke.mjs`：改为自行启动和关闭本地预览服务。
- `apps/web/scripts/capture.mjs`：改为自行启动和关闭本地预览服务。
- `apps/web/artifacts/*.png`：迁入 12 张已确认桌面、移动端和弹窗评审图。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 旧 `ui-preview` 目录受本机句柄占用且清理被执行策略拦截，当前由 Git 忽略；正式工程只使用 `apps/web`。
- 回滚点：阶段标签 `phase-1-baseline`；执行 `git revert phase-1-baseline` 可回退本阶段仓库改动，服务器本阶段没有写操作。

## 2026-07-22 - Task: 完成 ModelRoute 第 2 阶段部署底座与生产验收

### What was done

- 建立 Go API、后台 Worker、PostgreSQL 事务迁移、Redis 与 New API 依赖探针，提供存活、就绪和版本接口。
- 固定 New API `v1.0.0-rc.21` 镜像摘要，建立平台库与 New API 库的独立账号、数据库和内部网络。
- 建立六服务 Docker Compose、Nginx 网关、备份恢复、发布验收和宿主机 Caddy 接入配置。
- 将网关限制在 `127.0.0.1:28100`，New API、PostgreSQL 和 Redis 均未发布宿主机端口。
- 修复网关网络、IPv6 健康检查和发布版本注入问题；经 `current` 软链接部署时仍以真实发布目录名作为 `/api/version`。
- 在生产服务器部署独立 ModelRoute 发布，复用受限共享环境文件和持久化目录，未改动 Sub2API 与 SDK 的容器、数据库和路由。
- 为 `aggregate.lwylink.xyz` 备份并追加 Caddy 站点，完成配置校验和平滑重载，修复 Cloudflare `525`。
- 执行平台库与 New API 库备份并验证 SHA-256，生产环境文件权限保持为 `600`。

### Testing

- 执行 `go test ./...`、`go vet ./...` 和 API/Worker 构建，全部退出码为 `0`。
- 执行 `npm.cmd run build`，Vite 转换 `1603` 个模块；执行 `npm.cmd run test:ui`，四个视图和关键交互通过。
- 执行部署测试，Compose 隔离、固定 New API 摘要、网关 IPv4 健康检查和发布版本注入共 `5` 项通过。
- 发布版本测试先观察到四次 Compose 调用均未收到版本的预期失败，再实现最小修复并确认测试通过。
- 对全部 Shell 脚本执行语法检查，对 Compose 执行 `config --quiet`，并执行 `git diff --check`。
- 在服务器执行 `verify.sh http://127.0.0.1:28100` 与 `verify.sh https://aggregate.lwylink.xyz`，页面、存活、就绪和 `/v1` 关闭状态均通过。
- 检查六个 ModelRoute 容器，网关、New API、平台 API、PostgreSQL 和 Redis 均为健康状态，Worker 持续运行。
- 验证生产平台迁移记录为 `1`，共享 `.env` 权限为 `600`，New API、PostgreSQL 和 Redis 只有容器内部端口。
- 生成备份 `20260722T035043Z`，`platform.dump`、`newapi.dump`、Compose 配置和版本文件的 SHA-256 全部校验通过。
- 执行 `caddy validate`、源站 SNI 请求和公网 HTTPS 请求；ModelRoute 返回 `200`，原有 `api.lwylink.xyz` 与 `sdk.lwylink.xyz` 仍返回 `200`，Caddy 无新增错误日志。
- 验证 `/api/version` 返回发布提交 `f2842fc74089`，并扫描仓库确认 SSH、SMTP 和易支付凭据明文未进入源码、文档或构建输入。

### Notes

- `.dockerignore`：限制 Docker 构建上下文中的依赖、构建产物、密钥和运行数据。
- `.env.example`：定义 Compose 所需变量名及非生产示例值。
- `.gitattributes`：固定 Shell 脚本使用 LF 行尾。
- `.gitignore`：排除环境密钥、运行数据、构建产物和 Python 缓存。
- `apps/api/Dockerfile`：构建并输出带发布版本的 API 与 Worker 镜像。
- `apps/api/cmd/api/main.go`：启动迁移、依赖资源和 HTTP API。
- `apps/api/cmd/worker/main.go`：启动后台 Worker 并维持依赖连接。
- `apps/api/go.mod`：声明 Go 模块与服务依赖。
- `apps/api/go.sum`：锁定 Go 依赖校验值。
- `apps/api/internal/bootstrap/resources.go`：统一建立和关闭 PostgreSQL、Redis 与 New API 依赖。
- `apps/api/internal/config/config.go`：加载并校验服务运行配置。
- `apps/api/internal/config/config_test.go`：覆盖必填配置和超时配置验证。
- `apps/api/internal/dependencies/checks.go`：实现 Redis 与 New API 存活探针。
- `apps/api/internal/dependencies/checks_test.go`：覆盖 New API 探针成功与失败响应。
- `apps/api/internal/httpapi/server.go`：实现健康、版本和首期 `/v1` 关闭响应。
- `apps/api/internal/httpapi/server_test.go`：覆盖存活、依赖故障和 OpenAI 兼容错误格式。
- `apps/api/internal/migrations/migrate.go`：实现带锁的事务迁移器。
- `apps/api/internal/migrations/migrate_test.go`：验证迁移文件嵌入和命名顺序。
- `apps/api/internal/migrations/sql/000001_foundation.sql`：建立平台基础迁移记录和服务状态表。
- `compose.yaml`：编排六个服务、隔离网络、健康检查和持久化目录。
- `deploy/caddy/aggregate.Caddyfile`：提供 ModelRoute 宿主机反向代理站点块。
- `deploy/gateway/Dockerfile`：构建 React 静态资源并生成 Nginx 网关镜像。
- `deploy/gateway/default.conf`：分流网页、平台 API 和首期三个模型入口。
- `deploy/postgres/init-databases.sh`：首次创建相互隔离的平台库和 New API 库。
- `deploy/scripts/backup.sh`：导出两个数据库及部署元数据并生成 SHA-256 清单。
- `deploy/scripts/deploy.sh`：校验、构建、启动 Compose，并从真实发布目录注入版本。
- `deploy/scripts/restore.sh`：提供带显式确认的数据库恢复流程。
- `deploy/scripts/verify.sh`：验证网页、健康接口和 `/v1` 关闭状态。
- `deploy/tests/test_compose.py`：验证端口、网络、摘要和网关健康检查约束。
- `deploy/tests/test_deploy.py`：验证经 `current` 软链接发布时的版本注入行为。
- `docs/New API版本与许可.md`：记录固定镜像摘要、源码提交和许可证边界。
- `docs/部署运行手册.md`：记录发布、备份、恢复、Caddy 和版本核对步骤。
- `progress.md`：追加本阶段施工、验证、文件清单和回滚点。
- 服务器 `/etc/caddy/Caddyfile`：追加 `aggregate.lwylink.xyz` 站点；原配置备份为 `/etc/caddy/backups/Caddyfile.before-modelroute.20260722T035149Z`。
- 服务器 `/opt/modelroute-platform/releases/f2842fc74089`：保存已验收的独立发布；共享密钥与数据仍位于 `/opt/modelroute-platform/shared`。
- 回滚点：服务器执行 `ln -s releases/635085139bd1 /opt/modelroute-platform/current.rollback && mv -Tf /opt/modelroute-platform/current.rollback /opt/modelroute-platform/current && /opt/modelroute-platform/current/deploy/scripts/deploy.sh` 可恢复上一发布；Caddy 可用上述备份文件恢复后执行 `caddy validate` 与 `systemctl reload caddy`。
