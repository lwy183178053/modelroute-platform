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

## 2026-07-22 - Task: 实现第 3 阶段账号、密钥与管理员用户管理

### What was done

- 建立 Redis 服务端平台会话，浏览器只持有随机平台 Cookie；New API 会话、管理凭据和密钥均不返回浏览器。
- 接入注册、登录、二次验证、邮箱验证码、找回密码、退出和 `/api/me`，登录后主动刷新完整额度与用量资料。
- 接入 API 密钥列表、创建、编辑、启停、查看完整密钥和删除；复制掩码密钥前强制读取 New API 完整密钥。
- 接入管理员用户列表、搜索、详情、封禁、解封、额度调整、密码重置和平台审计日志；封禁和密码重置后清理目标用户会话。
- 前端按普通用户和管理员角色显示导航，未登录和普通用户不显示商户中心、管理后台；服务端对每个受限接口再次校验角色。
- 管理员重置密码时保留 New API 中目标用户的账号、显示名称、分组和备注，避免覆盖相邻资料。
- 将合法客户端来源 IP 传给内网 New API，用于其登录审计和 IP 限流；上游传输错误不再向浏览器暴露内部地址。
- 将用户额度按当前 New API `500000` 整数额度等于 `1 USD` 的口径显示为美元，后台额度调整仍使用整数。
- 新增可重复线上账号契约脚本，使用受限密码文件验证管理员登录、用户列表、临时密钥生命周期、审计和退出。

### Testing

- 先执行 UI 冒烟，确认真实账号尚未接入时因找不到“平台管理员”失败；实现后覆盖管理员资料、真实密钥、用户管理、审计、未登录权限和普通用户登录。
- 密码重置返回值、密钥启停、审计字段名、管理员密码重置保留备注、客户端 IP 转发、内部错误脱敏和完整密钥复制均完成对应失败测试与红绿验证。
- 执行 `go test ./...`、`go vet ./...`、`go build ./cmd/api ./cmd/worker`，全部退出码为 `0`。
- 执行 `npm.cmd run test:ui`，Vite 构建 `1607` 个模块后浏览器冒烟通过；测试命令已固定为先构建再预览，避免读取陈旧 `dist`。
- 执行 `npm.cmd run capture`，生成 8 个业务视图和 4 个弹窗截图，桌面与移动端根页面无横向溢出。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，Compose 隔离和发布版本共 5 项通过。
- 执行 `docker compose --env-file .env.example config --quiet`、`bash -n deploy/scripts/verify-accounts.sh`、`git diff --check` 和凭据模式扫描，均通过。
- 本机 Docker 镜像构建在拉取 Docker Hub 匿名令牌时超时，尚未形成本机构建证据；生产服务器发布时必须重新构建并以线上容器验收补齐。

### Notes

- `.env.example`：增加平台来源、会话 Cookie、会话时长和 New API 请求超时变量。
- `apps/api/cmd/api/main.go`：启动 HTTP 服务时注入账号、会话和审计依赖。
- `apps/api/go.mod`：增加 Redis 测试服务和 SQL 模拟测试依赖。
- `apps/api/go.sum`：锁定本阶段新增 Go 依赖校验值。
- `apps/api/internal/bootstrap/resources.go`：构建并关闭账号服务所需资源。
- `apps/api/internal/bootstrap/resources_test.go`：验证会话、New API 客户端和审计依赖注入。
- `apps/api/internal/config/config.go`：加载并校验账号服务运行配置。
- `apps/api/internal/config/config_test.go`：覆盖会话与公开来源配置校验。
- `apps/api/internal/httpapi/server.go`：注册账号、密钥和管理员接口，并注入客户端来源信息。
- `apps/api/internal/httpapi/server_test.go`：覆盖上游传输错误脱敏。
- `apps/api/internal/httpapi/account_handlers.go`：实现密钥与管理员用户操作。
- `apps/api/internal/httpapi/account_test.go`：覆盖密钥生命周期、额度、密码、备注保护和审计响应。
- `apps/api/internal/httpapi/audit_store.go`：实现 PostgreSQL 审计日志写入与分页读取。
- `apps/api/internal/httpapi/audit_store_test.go`：覆盖审计 SQL 参数和结果解析。
- `apps/api/internal/httpapi/auth_handlers.go`：实现公开账号流程、平台会话、角色、来源校验和错误转换。
- `apps/api/internal/httpapi/auth_test.go`：覆盖登录、二次验证、邮箱、重置、权限、跨站请求和会话清理。
- `apps/api/internal/httpapi/services.go`：定义账号服务依赖、角色和审计契约。
- `apps/api/internal/newapi/client.go`：实现固定 New API JSON 客户端、Cookie 隔离和来源 IP 透传。
- `apps/api/internal/newapi/client_test.go`：覆盖 New API 登录、鉴权头、来源 IP 和业务错误。
- `apps/api/internal/platformauth/store.go`：实现 Redis 会话、用户会话索引和批量清退。
- `apps/api/internal/platformauth/store_test.go`：覆盖会话令牌哈希、保存、删除和按用户清理。
- `apps/web/package.json`：让 UI 冒烟和截图命令先构建最新产物。
- `apps/web/scripts/capture.mjs`：为视觉截图注入管理员和密钥测试数据。
- `apps/web/scripts/ui-smoke.mjs`：覆盖角色导航、登录、额度刷新、密码重置、密钥和管理员页面。
- `apps/web/src/api.js`：新增统一平台请求、错误解析和美元额度格式化。
- `apps/web/src/AuthModal.jsx`：实现登录、注册、邮箱、找回密码、二次验证和新密码展示。
- `apps/web/src/Modals.jsx`：增加密钥与管理员额度、密码操作弹窗。
- `apps/web/src/main.jsx`：加载真实账号、过滤导航并实现退出登录。
- `apps/web/src/styles.css`：增加账号表单、密钥开关、表格状态和移动端样式。
- `apps/web/src/views/ConsoleView.jsx`：改用真实账号与密钥数据，移除虚假用量和调用记录。
- `apps/web/src/views/AdminView.jsx`：接入用户管理和真实审计页面。
- `apps/web/src/views/admin/AdminUsers.jsx`：实现管理员用户查询与操作入口。
- `apps/web/src/views/admin/AdminAudit.jsx`：实现平台审计日志列表。
- `apps/web/artifacts/admin-desktop.png`：更新管理员桌面验收图。
- `apps/web/artifacts/console-desktop.png`：更新真实账号控制台桌面验收图。
- `apps/web/artifacts/console-mobile.png`：更新真实账号控制台移动端验收图。
- `apps/web/artifacts/market-desktop.png`：更新登录态市场桌面验收图。
- `apps/web/artifacts/merchant-desktop.png`：更新登录态商户桌面验收图。
- `apps/web/artifacts/payment-config-desktop.png`：更新登录态支付弹窗桌面验收图。
- `apps/web/artifacts/route-modal-desktop.png`：更新登录态路由弹窗桌面验收图。
- `apps/web/artifacts/route-modal-mobile.png`：更新登录态路由弹窗移动端验收图。
- `compose.yaml`：向 API 和 Worker 传递账号会话配置。
- `deploy/scripts/verify-accounts.sh`：新增不打印管理员密码的线上账号契约验证。
- `docs/账号与管理员接口.md`：记录账号边界、权限、接口和运行配置。
- `docs/部署运行手册.md`：补充账号服务环境变量和线上契约步骤。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚说明。
- 回滚点：阶段标签 `phase-2-foundation`；本阶段提交后可执行 `git revert <第3阶段提交>` 回退仓库，生产服务器继续保留上一发布目录用于软链接回滚。

## 2026-07-22 - Task: 完成第 3 阶段生产部署与账号邮件验收

### What was done

- 将第 3 阶段账号、密钥和管理员功能部署到独立 ModelRoute 生产发布目录，保持 New API、数据库和 Redis 仅内网可达。
- 通过 New API 原生系统设置接口复用既有 SMTP，启用邮箱验证并完成注册验证码和密码重置邮件实发。
- 完成管理员登录、用户列表、密钥生命周期、审计和退出的线上契约验证；修正验收脚本对 New API 原始密钥格式的判断。
- 保留发布前和 SMTP 配置后的数据库备份，并补齐 SMTP 存储位置、稳定窗口日志和生产验收说明。

### Testing

- 在本机执行 `go test ./...`、`go vet ./...`、API 与 Worker 构建，全部退出码为 `0`。
- 执行 `npm.cmd run test:ui` 和 `npm.cmd run capture`，Vite 构建 `1607` 个模块，账号界面、管理员界面、桌面与移动端视图通过。
- 执行部署测试共 `5` 项，并完成 Compose 配置、Shell 语法、Git 差异和凭据模式检查。
- 在生产服务器重新构建 Go 与 React 镜像，六个服务保持运行；`/api/version` 返回发布提交 `4e60d10`。
- 公网基础验收和账号契约通过，覆盖管理员登录、用户列表、密钥创建、查看、启停、删除、审计和退出。
- 注册验证码与密码重置邮件真实发送成功，临时测试用户清理后残留数为 `0`。
- `api.lwylink.xyz` 和 `sdk.lwylink.xyz` 均返回 `200`；生产环境文件权限为 `600`，New API、PostgreSQL 和 Redis未发布宿主机端口。
- 发布前备份 `20260722T052611Z` 与 SMTP 配置后备份 `20260722T054158Z` 的校验文件均通过。
- 在 `2026-07-22T05:46:31Z` 至 `05:46:38Z` 的稳定窗口重复本机、公网和账号契约请求，Caddy 未出现新增错误、TLS、`502`、`503`、`504` 或 `context canceled` 日志。

### Notes

- `docs/部署运行手册.md`：补充 New API 原生 SMTP 设置、敏感备份和稳定窗口验收流程。
- `docs/账号与管理员接口.md`：修正 SMTP 凭据存储说明并记录账号线上契约结果。
- `docs/部署环境审计.md`：补充 HTTPS、隔离、邮件和备份的实施后核验事实。
- `progress.md`：追加第 3 阶段生产部署和邮件验收记录。
- 服务器 `/opt/modelroute-platform/releases/4e60d10`：保存本阶段已部署发布；共享密钥和运行数据位于受限共享目录。
- 回滚点：服务器将 `/opt/modelroute-platform/current` 切换到上一发布目录 `f2842fc74089` 后执行该目录的 `deploy/scripts/deploy.sh`；仓库按 `4e60d10`、`d79b845` 的顺序逐个执行 `git revert` 可生成回退改动。

## 2026-07-22 - Task: 实现第 4 阶段商户申请与渠道审核

### What was done

- 建立平台商户申请、商户资料、渠道审核版本和 New API 渠道归属四类数据，商户身份与 New API 用户角色分开判断。
- 普通用户可提交网站、业务说明、上游地址、密钥和拟售模型；有效商户可继续提交新渠道，同一用户只保留一条进行中的商户申请。
- 使用 AES-256-GCM 加密待审核密钥，并以 HMAC-SHA256 生成唯一指纹；接口不返回密钥、密文或完整指纹，批准和驳回后清除平台密文。
- 管理员可读取申请、渠道审核、商户和渠道列表，执行上游模型验证、批准导入、驳回、撤销和恢复商户权限，写操作均校验角色、账号状态和请求来源并记录审计。
- 渠道批准时调用固定版本 New API 创建渠道、搜索编号并执行真实渠道测试；失败时删除已创建渠道并恢复审核状态，成功后以事务保存商户归属。
- 前端按平台商户身份显示商户中心，接入真实申请、渠道审核版本和 New API 映射；管理员商户与渠道页面接入真实平台接口，移除页面内虚假商户收入、报价和健康数据。
- 新增生产商户契约脚本和中文接口文档，部署配置增加受限的商户凭据加密主密钥。

### Testing

- 有效商户重复申请测试先因仍调用创建仓储而失败，增加入口身份检查后转为通过；New API 渠道测试失败的删除和审核恢复补偿测试通过。
- 执行 `go test ./...`、`go vet ./...` 和 `go build ./cmd/...`，全部退出码为 `0`。
- 在临时 PostgreSQL 17 容器顺序执行两份迁移，核对 `4` 张商户表、`3` 个关键唯一索引和 `6` 个外键后删除临时容器。
- 执行 `npm.cmd run test:ui`，Vite 转换 `1607` 个模块，普通用户申请、角色导航、管理员验证、真实渠道列表和 New API 映射交互通过。
- 执行 `npm.cmd run capture`，生成桌面与移动端商户中心、渠道管理和既有关键视图截图；Playwright 根页面溢出检查通过，并人工检查渠道表格和操作区布局。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py'`，共 `6` 项通过；Compose 配置和全部 Shell 脚本语法检查通过。
- 执行 `git diff --check` 和聊天中认证凭据字面值扫描，均无异常；线上商户契约保留到生产部署后执行。

### Notes

- `.env.example`：增加商户凭据加密主密钥示例变量。
- `apps/api/internal/bootstrap/resources.go`：注入 SQL 商户仓储和商户凭据保险箱。
- `apps/api/internal/bootstrap/resources_test.go`：验证商户服务依赖已完成注入。
- `apps/api/internal/config/config.go`：加载并校验 Base64 编码的 32 字节加密主密钥。
- `apps/api/internal/config/config_test.go`：覆盖加密主密钥缺失、格式错误和正确加载。
- `apps/api/internal/httpapi/auth_handlers.go`：在 `/api/me` 中返回平台商户身份与状态。
- `apps/api/internal/httpapi/server.go`：注册商户、渠道和管理员审核接口。
- `apps/api/internal/httpapi/services.go`：定义商户仓储与凭据保护依赖。
- `apps/api/internal/httpapi/merchant_types.go`：定义商户申请、渠道、身份、状态和仓储契约。
- `apps/api/internal/httpapi/merchant_handlers.go`：实现申请、审核、New API 导入补偿、权限与输入校验。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖密钥保护、上游验证、导入补偿、角色权限和重复申请。
- `apps/api/internal/httpapi/merchant_store.go`：实现商户 SQL 查询、原子状态转换和审批事务。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖申请事务、审核认领和最终批准事务。
- `apps/api/internal/migrations/migrate_test.go`：把商户迁移纳入嵌入迁移顺序验证。
- `apps/api/internal/migrations/sql/000002_merchants.sql`：建立四张商户业务表、状态约束、唯一约束和外键。
- `apps/api/internal/secretbox/box.go`：实现 AES-256-GCM 加解密和 HMAC-SHA256 指纹。
- `apps/api/internal/secretbox/box_test.go`：覆盖密文、解密、指纹、错误密钥和篡改检测。
- `apps/web/src/main.jsx`：按平台返回的商户身份控制导航和申请刷新。
- `apps/web/src/Modals.jsx`：实现商户申请和新增渠道表单与真实提交。
- `apps/web/src/views/ConsoleView.jsx`：读取真实申请状态并显示审核进度。
- `apps/web/src/views/MerchantView.jsx`：展示真实商户身份、审核版本和已导入渠道。
- `apps/web/src/views/admin/AdminMerchants.jsx`：实现商户申请审核和商户权限操作。
- `apps/web/src/views/admin/AdminChannels.jsx`：实现渠道审核操作和 New API 渠道归属列表。
- `apps/web/scripts/ui-smoke.mjs`：覆盖商户申请、管理员验证和真实渠道页面接口调用。
- `apps/web/scripts/capture.mjs`：增加商户与管理员渠道真实数据形状及两张渠道管理截图。
- `apps/web/artifacts/merchant-desktop.png`：更新真实商户渠道桌面验收图。
- `apps/web/artifacts/merchant-mobile.png`：更新真实商户渠道移动端验收图。
- `apps/web/artifacts/admin-channels-desktop.png`：新增管理员渠道桌面验收图。
- `apps/web/artifacts/admin-channels-mobile.png`：新增管理员渠道移动端验收图。
- `compose.yaml`：向 API 和 Worker 容器传递商户凭据加密主密钥。
- `deploy/tests/test_compose.py`：验证两个平台进程均收到加密主密钥变量。
- `deploy/scripts/verify-merchants.sh`：新增匿名权限、申请、列表、密钥清除和驳回线上契约验证。
- `docs/商户与渠道审核接口.md`：记录业务边界、状态机、密钥生命周期、接口和验证方式。
- `docs/部署运行手册.md`：补充加密主密钥和商户契约验收步骤。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：阶段标签 `phase-3-accounts`；第 4 阶段提交后可执行 `git revert <第4阶段提交>` 生成仓库回退改动，生产服务器在本轮记录时尚未部署此阶段。

## 2026-07-22 - Task: 完成第 4 阶段生产部署与商户契约验收

### What was done

- 在生产发布前备份平台库、New API 库、Compose 和版本信息，并单独保留修改前的受限环境文件。
- 将已推送提交 `192f31c` 作为不可变发布上传到服务器，校验本地与服务器归档 SHA-256 一致后构建六服务 Compose。
- 在共享 `.env` 中生成 32 字节随机商户凭据加密主密钥，完成 Base64 长度校验并保持文件权限 `600`。
- 执行第 2 份平台迁移，完成内网与公网基础验收、账号契约和新增商户契约，随后原子切换 `current`。
- 商户契约使用随机测试凭据创建申请，验证列表和响应不回显密钥，再通过管理员接口驳回并确认数据库密文字段为空。

### Testing

- 发布前备份 `20260722T065007Z` 的平台库、New API 库、Compose 和版本文件 SHA-256 全部通过，修改前 `.env` 已保存到同一受限备份目录。
- 服务器发布归档 SHA-256 与本地一致；Compose 部署进程退出码为 `0`，API 自动执行两份平台迁移。
- `deploy/scripts/verify.sh` 在 `http://127.0.0.1:28100` 和 `https://aggregate.lwylink.xyz` 均通过。
- `verify-accounts.sh` 与 `verify-merchants.sh` 均通过；测试申请最终状态为 `rejected`，对应待审核密文为空。
- `/api/version` 返回 `192f31c`；网关、New API、平台 API、Worker、PostgreSQL 和 Redis 共六个服务均为运行状态，带健康检查的服务均为 `healthy`。
- `aggregate.lwylink.xyz` 的 HTTP 返回 `308`、HTTPS 返回 `200`，`api.lwylink.xyz` 与 `sdk.lwylink.xyz` 均返回 `200`。
- New API、PostgreSQL 和 Redis 没有宿主机发布端口；稳定窗口 Caddy 日志未出现新增 TLS、`502`、`503`、`504` 或 `context canceled` 异常。

### Notes

- `docs/商户与渠道审核接口.md`：补充第 4 阶段生产契约与密钥验收结果。
- `docs/部署环境审计.md`：补充生产加密主密钥、平台迁移和隔离核验事实。
- `progress.md`：追加生产部署、契约验收、文件清单和回滚点。
- 服务器 `/opt/modelroute-platform/releases/192f31c`：保存第 4 阶段已验收运行发布。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T065007Z`：保存发布前数据库、Compose、版本和环境文件备份。
- 回滚点：服务器将 `/opt/modelroute-platform/current` 原子切换到 `/opt/modelroute-platform/releases/a3e1bdd`，再执行该目录的 `deploy/scripts/deploy.sh`；平台数据库需恢复时使用上述发布前备份和 `restore.sh`。

## 2026-07-22 - Task: 实现第 5 阶段模型报价、用户路由与渠道健康

### What was done

- 建立标准模型、官方价格版本、商户报价版本、密钥指纹绑定、用户路由策略、渠道模型健康、路由决策、健康事件和性能汇总数据结构，价格和倍率全部使用整数单位。
- 增加 New API 服务账号会话、专用探活令牌、价格选项同步、渠道模型别名映射和报价激活流程；同步失败时保留旧报价继续服务。
- 为每个 New API 密钥建立独立策略，支持智能路由、固定商户、倍率区间、价格优先和稳定性优先；删除密钥时同步撤销平台绑定。
- 开放聊天补全、Responses 和嵌入三个模型接口，仅替换请求模型名并原样转交鉴权、正文与流式响应给 New API；首字节前允许在 `429/5xx` 候选间切换。
- Worker 接入主动探活、三级故障判断、阶梯冷却、两次恢复探测和硬故障双侧停用；真实请求只记录状态、耗时、渠道和请求编号。
- 模型市场、控制台、商户中心和管理后台全部改用真实模型、报价、策略和健康接口，补齐商户创建报价、管理员发布官方价格和恢复渠道操作。
- 增加 PostgreSQL 真实 SQL 集成测试、生产路由契约脚本、中文业务文档和桌面/移动端验收截图。

### Testing

- 路由市场公开接口测试先因旧会话参数预期失败，按公开只读契约修正后通过；商户渠道状态文案测试先稳定复现“在售”误用，单点修正为“正常”后通过。
- 执行 `go test ./...`、`go vet ./...`、`go build ./cmd/...` 和 `go test -race ./...`，全部退出码为 `0`。
- 在临时 PostgreSQL 17 容器顺序执行三份迁移，确认标准模型、报价、路由策略、健康和决策关键表存在；随后用真实 `pgx` 连接验证倍率过滤、固定商户、探活领取和结果写回。
- 执行 `npm.cmd run test:ui`，Vite 转换 `1608` 个模块，真实市场、密钥策略、商户报价、管理员模型价格和渠道恢复交互通过。
- 执行 `npm.cmd run capture`，生成桌面与移动端市场、控制台、商户、模型定价、渠道管理和路由弹窗截图；Playwright 根页面溢出与空白检查通过，并人工检查关键页面。
- 执行部署测试共 `6` 项，Compose 配置和全部 Shell 脚本语法通过；`git diff --check` 和聊天中认证、支付凭据字面值扫描无异常。
- 生产路由契约脚本保留到本阶段提交部署后执行，本轮没有把本地验证描述为线上完成。

### Notes

- `.env.example`：增加 New API 服务端管理员账号和密码示例变量。
- `.gitignore`：忽略 Go 在 Windows 根包构建产生的可执行文件。
- `apps/api/cmd/worker/main.go`：每 10 秒执行渠道探活，并保留原有依赖状态日志周期。
- `apps/api/internal/bootstrap/resources.go`：注入服务账号、SQL 路由仓储、报价同步器和健康任务。
- `apps/api/internal/bootstrap/resources_test.go`：验证模型路由和健康依赖已完成注入。
- `apps/api/internal/config/config.go`：加载并强制校验 New API 服务账号凭据。
- `apps/api/internal/config/config_test.go`：覆盖服务账号凭据缺失和正常加载。
- `apps/api/internal/httpapi/account_handlers.go`：在密钥创建、查询和删除流程中建立或撤销路由绑定。
- `apps/api/internal/httpapi/health_runner.go`：实现专用令牌探活、状态分类和 New API 硬故障停用。
- `apps/api/internal/httpapi/health_runner_test.go`：覆盖探活请求、冷却、恢复、硬故障和令牌登记。
- `apps/api/internal/httpapi/pricing_sync.go`：同步 New API 全部计费维度、渠道模型和内部别名映射。
- `apps/api/internal/httpapi/pricing_sync_test.go`：覆盖 Token、固定价格、缓存维度、原价格保留和同步顺序。
- `apps/api/internal/httpapi/routing_domain.go`：实现倍率换算、模型字段替换、候选排序和冷却状态机。
- `apps/api/internal/httpapi/routing_domain_test.go`：覆盖整数换算、协议正文、排序、三级冷却和两次恢复。
- `apps/api/internal/httpapi/routing_handlers.go`：实现模型、报价、路由策略和管理员健康恢复接口。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖价格上限、报价同步、密钥绑定、策略校验、公开市场和恢复顺序。
- `apps/api/internal/httpapi/routing_proxy.go`：实现三类 `/v1/*` 请求的候选选择、首字节前切换和流式透传。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖价格与稳定性路由、失败切换、流式响应和无候选错误。
- `apps/api/internal/httpapi/routing_store.go`：实现标准模型、报价版本、路由决策、健康记录和查询的 PostgreSQL 仓储。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖报价事务、版本激活、策略绑定、健康更新和官方价格发布 SQL 契约。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：使用真实 PostgreSQL 验证关键候选和探活 SQL。
- `apps/api/internal/httpapi/server.go`：注册模型、报价、策略、健康和三个模型调用接口。
- `apps/api/internal/httpapi/services.go`：扩展路由、报价同步、健康和服务账号依赖契约。
- `apps/api/internal/migrations/migrate_test.go`：验证第 3 份路由迁移顺序和关键表。
- `apps/api/internal/migrations/sql/000003_routing.sql`：创建模型报价、密钥路由和健康记录相关表、约束与索引。
- `apps/api/internal/newapi/client.go`：增加保持原始请求头和响应流的模型代理入口。
- `apps/api/internal/newapi/client_test.go`：验证 Authorization、正文、流式响应和客户端地址透传。
- `apps/api/internal/newapi/service_account.go`：实现 New API 管理会话复用、失效重登和探活令牌管理。
- `apps/api/internal/newapi/service_account_test.go`：覆盖服务账号登录复用、令牌查找、创建和读取。
- `apps/web/src/main.jsx`：把市场“用于路由”入口引导到具体密钥所在的控制台。
- `apps/web/src/Modals.jsx`：接入真实路由策略保存和商户报价创建。
- `apps/web/src/data.js`：删除已被真实接口替代的模型市场、商户渠道和报价演示数据。
- `apps/web/src/views/MarketView.jsx`：展示真实标准模型、商户报价、可用率、P95 和倍率。
- `apps/web/src/views/ConsoleView.jsx`：按密钥读取并显示模式、倍率区间、优先级和固定商户数量。
- `apps/web/src/views/MerchantView.jsx`：展示真实报价版本与健康状态，并提供创建报价入口。
- `apps/web/src/views/AdminView.jsx`：增加模型与定价管理导航。
- `apps/web/src/views/admin/AdminModels.jsx`：实现标准模型创建和官方价格版本发布界面。
- `apps/web/src/views/admin/AdminChannels.jsx`：展示渠道关联报价健康并提供硬故障恢复操作。
- `apps/web/scripts/ui-smoke.mjs`：覆盖第 5 阶段真实接口请求、提交参数和角色页面交互。
- `apps/web/scripts/capture.mjs`：使用真实接口数据形状生成第 5 阶段桌面与移动端截图。
- `apps/web/artifacts/market-desktop.png`：更新真实市场桌面验收图。
- `apps/web/artifacts/market-mobile.png`：更新真实市场移动端验收图。
- `apps/web/artifacts/console-desktop.png`：更新密钥路由桌面验收图。
- `apps/web/artifacts/console-mobile.png`：更新密钥路由移动端验收图。
- `apps/web/artifacts/merchant-desktop.png`：更新商户报价和健康桌面验收图。
- `apps/web/artifacts/merchant-mobile.png`：更新商户报价和健康移动端验收图。
- `apps/web/artifacts/admin-desktop.png`：更新管理员页面桌面验收图。
- `apps/web/artifacts/admin-channels-desktop.png`：更新渠道健康桌面验收图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新渠道健康移动端验收图。
- `apps/web/artifacts/admin-models-desktop.png`：新增模型定价桌面验收图。
- `apps/web/artifacts/admin-models-mobile.png`：新增模型定价移动端验收图。
- `apps/web/artifacts/payment-config-desktop.png`：随最新前端构建更新支付配置桌面回归图。
- `apps/web/artifacts/route-modal-desktop.png`：更新真实策略桌面弹窗验收图。
- `apps/web/artifacts/route-modal-mobile.png`：更新真实策略移动端弹窗验收图。
- `compose.yaml`：向 API 和 Worker 传递 New API 服务账号变量。
- `deploy/tests/test_compose.py`：验证两个平台进程均接收服务账号变量。
- `deploy/scripts/verify-routing.sh`：新增临时密钥、公开市场和 OpenAI 错误线上契约。
- `docs/模型报价路由与健康.md`：记录业务边界、版本计价、路由、探活、接口、验收和回滚。
- `docs/部署运行手册.md`：补充服务账号配置和路由契约验收步骤。
- `progress.md`：追加第 5 阶段本地实现和验证记录。
- 回滚点：阶段标签 `phase-4-merchants`；第 5 阶段提交后可执行 `git revert <第5阶段提交>` 生成仓库回退改动，生产服务器在本轮记录时仍运行第 4 阶段发布 `192f31c`。

## 2026-07-22 - Task: 修正第 5 阶段基础部署验收契约

### What was done

- 将基础验收中的模型入口预期从“路由尚未开放”更新为“路由已开放但缺少 API 密钥”，与第 5 阶段真实行为一致。
- 增加部署测试，锁定未鉴权模型请求必须返回 `401/invalid_api_key`，避免后续再次使用旧的 `503/service_not_ready` 断言。

### Testing

- 新测试先因旧脚本仍期待 `503` 稳定失败，修正脚本后转为通过。
- 执行部署测试共 `7` 项，全部通过；`verify.sh` Shell 语法检查通过。
- 生产新版本服务本身保持健康，修复对象仅为尚未通过的验收脚本，不改变运行中 API、数据库或路由逻辑。

### Notes

- `deploy/scripts/verify.sh`：验证已启用模型入口的未鉴权错误结构。
- `deploy/tests/test_compose.py`：增加基础验收脚本契约回归测试。
- `progress.md`：追加本轮验收契约修正记录。
- 回滚点：提交 `c0b2fa0`；本轮修复提交可单独执行 `git revert <本轮修复提交>`，生产发布指针在记录时仍未切换。

## 2026-07-22 - Task: 完成第 5 阶段生产部署与路由契约验收

### What was done

- 在生产发布前备份平台库、New API 库、Compose、版本和修改前受限环境文件，并重新生成和校验统一 SHA-256 清单。
- 从服务器现有受限管理员密码文件向共享 `.env` 原子写入 New API 服务账号变量，环境文件和来源文件权限均保持 `600`，没有输出凭据正文。
- 上传并校验提交归档，执行第 3 份平台迁移，构建并启动六服务 Compose；验收脚本修正后使用最终提交 `6dd0565` 重新发布。
- 完成内网和公网基础验收、账号契约、商户契约和新增模型路由契约，再原子切换 `current` 到最终发布目录。
- 核对标准模型、报价、路由、探活和决策关键表、专用探活令牌登记、临时密钥清理、服务健康、端口隔离、原有域名和稳定窗口日志。

### Testing

- 发布前备份 `20260722T082512Z` 中平台库、New API 库、Compose、版本和修改前环境文件的 SHA-256 全部通过；环境副本权限为 `600`。
- 发布归档本地与服务器 SHA-256 一致；Compose 构建和启动退出码为 `0`，API 自动执行 `000003_routing.sql`。
- `verify.sh` 在 `http://127.0.0.1:28100` 和 `https://aggregate.lwylink.xyz` 均通过；`verify-accounts.sh`、`verify-merchants.sh` 和 `verify-routing.sh` 全部通过。
- `/api/version` 返回 `6dd0565`，`current` 指向 `/opt/modelroute-platform/releases/6dd0565`；8 张路由关键表存在，探活令牌元数据为 1 条，契约结束后有效临时路由绑定为 0 条。
- 网关、New API、平台 API、Worker、PostgreSQL 和 Redis 共六个服务运行，5 个健康检查均为 `healthy`；除网关回环端口外，内部服务没有宿主机发布端口。
- `aggregate.lwylink.xyz` 的 HTTP 返回 `308`、HTTPS 返回 `200`；`api.lwylink.xyz` 与 `sdk.lwylink.xyz` 均返回 `200`。
- 最终稳定窗口内 ModelRoute 网关没有 `499` 或 `5xx`，平台 API 和 Worker 没有 `panic`、`fatal` 或渠道探活异常。
- 同机 Caddy 全局日志的 `context canceled` 出现在 ModelRoute 无请求的时间段，证据指向其他域名流量，未将其误记为 ModelRoute 正常。

### Notes

- `docs/模型报价路由与健康.md`：补充第 5 阶段生产版本、迁移、契约、隔离、日志和备份验收事实。
- `docs/部署环境审计.md`：补充服务账号、路由迁移、探活令牌和最终生产隔离状态。
- `progress.md`：追加第 5 阶段生产部署、验收和回滚记录。
- 服务器 `/opt/modelroute-platform/releases/6dd0565`：保存第 5 阶段最终已验收运行发布。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T082512Z`：保存发布前两个数据库、Compose、版本和修改前环境文件备份。
- 回滚点：服务器将 `/opt/modelroute-platform/current` 原子切换到 `/opt/modelroute-platform/releases/192f31c` 并执行该目录的 `deploy/scripts/deploy.sh`；需要连数据库和环境文件回退时，使用上述发布前备份和 `restore.sh`，再恢复 `environment.before.env`。

## 2026-07-22 - Task: 实现第 6 阶段易支付充值与 New API 额度入账

### What was done

- 建立不可变支付配置版本、充值订单、回调事件和 New API 入账操作四类数据，美元额度、人民币实付、汇率和 New API 额度全部使用整数保存与计算。
- 实现易支付订单签名和 GET/POST 异步通知验签，逐项核对签名、PID、平台订单号、第三方交易号、人民币金额、支付方式和成功状态；浏览器返回页只查询当前用户订单。
- 支付回调使用订单行锁、回调摘要、第三方交易号和单订单入账键控制重复与并发通知，只创建一条 New API 入账操作。
- Worker 接入额度入账状态机：明确的 New API 错误进入有限重试，网络超时或处理中断进入结果未知并停止自动重试，管理员核对后才能确认或重新排队。
- 管理员支付页接入真实配置、订单、失败重试和结果未知处置接口；用户充值弹窗按美元创建锁定订单，控制台展示真实充值记录，并新增支付结果页。
- 清除旧充值演示订单和管理员概览中的虚假入账异常，统一本轮前端新增代码缩进，更新支付文档、部署验收脚本和桌面/移动端回归截图。

### Testing

- 使用本机 PostgreSQL 17 临时容器和 `MODELROUTE_TEST_DATABASE_URL` 执行 `go test ./...`，全部通过；真实 SQL 用例确认 8 个相同并发回调只生成 1 条回调事件和 1 条入账操作，两个并发 Worker 合计只领取 1 次。
- 使用同一真实数据库执行 `go test -race ./...`，全部通过；执行 `go vet ./...` 和 `go build ./cmd/...`，退出码均为 `0`。
- 执行 `npm.cmd run test:ui`，Vite 转换 `1609` 个模块，支付配置提交、失败入账重试、用户充值跳转和返回页只读契约通过。
- 执行 `npm.cmd run capture`，更新管理员支付、配置弹窗、控制台及关联回归截图；人工检查桌面和移动端支付信息、表格滚动与弹窗布局。
- 执行部署测试共 `8` 项，全部通过；8 个部署 Shell 脚本逐一通过 `sh -n` 语法检查。
- 执行 `git diff --check`、未合并文件检查和聊天中出现过的 SSH、易支付敏感字面值扫描，均无异常；生产支付契约保留到本阶段提交部署后执行。

### Notes

- `apps/api/cmd/worker/main.go`：每 10 秒领取一次待处理支付入账操作。
- `apps/api/internal/bootstrap/resources.go`：注入支付仓储和额度入账任务。
- `apps/api/internal/bootstrap/resources_test.go`：验证支付依赖已完整注入。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：补齐真实数据库路由测试的数据清理，避免与支付竞态套件互相污染。
- `apps/api/internal/httpapi/server.go`：注册用户、管理员、回调和支付订单接口。
- `apps/api/internal/httpapi/services.go`：增加支付仓储与入账周期依赖契约。
- `apps/api/internal/httpapi/payment_credit.go`：实现 New API 额度入账、重试退避和结果未知分类。
- `apps/api/internal/httpapi/payment_credit_test.go`：覆盖成功、明确失败、网络结果未知、过期处理中断和退避规则。
- `apps/api/internal/httpapi/payment_domain.go`：实现整数金额换算、易支付签名、验签和网关金额解析。
- `apps/api/internal/httpapi/payment_domain_test.go`：覆盖整数四舍五入、溢出、金额格式与 MD5 签名规则。
- `apps/api/internal/httpapi/payment_handlers.go`：实现支付配置、订单、异步通知、管理员重试和人工核对处置。
- `apps/api/internal/httpapi/payment_handlers_test.go`：覆盖权限、配置版本、订单锁价、签名篡改、返回页读取和管理员操作。
- `apps/api/internal/httpapi/payment_store.go`：实现配置、订单、回调和入账操作的 PostgreSQL 事务与并发领取。
- `apps/api/internal/httpapi/payment_store_postgres_test.go`：使用真实 PostgreSQL 验证并发回调、并发领取和结果未知重排。
- `apps/api/internal/httpapi/payment_store_test.go`：覆盖 SQL 配置版本、订单金额锁定和回调幂等契约。
- `apps/api/internal/httpapi/payment_types.go`：定义支付领域数据、命令和仓储接口。
- `apps/api/internal/migrations/migrate_test.go`：验证第 4 份支付迁移顺序和关键表。
- `apps/api/internal/migrations/sql/000004_payments.sql`：创建支付配置版本、订单、回调事件和入账操作表及唯一约束。
- `apps/web/src/Modals.jsx`：实现真实支付配置、用户充值和结果未知处置弹窗。
- `apps/web/src/data.js`：删除已由真实接口替代的充值演示订单。
- `apps/web/src/main.jsx`：注册支付返回页并处理返回后的站内导航。
- `apps/web/src/ui.jsx`：补齐支付与入账状态的视觉语义。
- `apps/web/src/views/AdminView.jsx`：移除硬编码支付异常和虚假待办数量。
- `apps/web/src/views/ConsoleView.jsx`：读取并展示当前用户真实充值订单。
- `apps/web/src/views/PaymentReturnView.jsx`：新增仅查询订单状态的支付返回页。
- `apps/web/src/views/admin/AdminPayment.jsx`：接入真实支付设置、订单、重试和人工处置接口。
- `apps/web/scripts/ui-smoke.mjs`：覆盖支付配置、充值、管理员处置和返回页只读行为。
- `apps/web/scripts/capture.mjs`：使用真实支付接口数据形状生成回归截图。
- `apps/web/artifacts/admin-desktop.png`：更新管理员支付入口桌面回归图。
- `apps/web/artifacts/admin-mobile.png`：更新管理员支付入口移动端回归图。
- `apps/web/artifacts/console-desktop.png`：新增充值记录的控制台桌面回归图。
- `apps/web/artifacts/console-mobile.png`：新增充值记录的控制台移动端回归图。
- `apps/web/artifacts/payment-config-desktop.png`：更新真实支付配置桌面弹窗回归图。
- `apps/web/artifacts/payment-config-mobile.png`：更新真实支付配置移动端弹窗回归图。
- `apps/web/artifacts/route-modal-desktop.png`：随最新前端构建更新路由弹窗桌面回归图。
- `apps/web/artifacts/route-modal-mobile.png`：随最新前端构建更新路由弹窗移动端回归图。
- `deploy/scripts/verify-payments.sh`：增加生产支付权限、返回页和无效回调契约检查。
- `deploy/tests/test_compose.py`：锁定支付验收脚本必须覆盖的安全边界。
- `docs/易支付充值.md`：记录支付口径、版本、幂等、人工核对、接口、验证和回滚。
- `docs/部署运行手册.md`：补充支付配置、发布与生产验收步骤。
- `progress.md`：追加第 6 阶段本地实现、验证、文件清单和回滚点。
- 回滚点：阶段标签 `phase-5-routing`；第 6 阶段提交后可执行 `git revert <第6阶段提交>` 生成代码回退改动。生产服务器在本轮记录时仍运行第 5 阶段发布 `6dd0565`，支付迁移尚未发布。

## 2026-07-22 - Task: 修正支付生产验收脚本可执行位

### What was done

- 将新增的支付生产验收脚本标记为 Git 可执行文件，使部署后可按运行手册直接执行。

### Testing

- 执行 `sh -n deploy/scripts/verify-payments.sh` 通过；`git ls-files -s` 显示脚本模式为 `100755`，`git diff --cached --check` 无异常。

### Notes

- `deploy/scripts/verify-payments.sh`：将文件模式从 `100644` 修正为 `100755`，脚本正文未变化。
- `progress.md`：追加本轮发布前修正记录。
- 回滚点：提交 `064b9ee`；本轮提交可单独执行 `git revert <本轮提交>` 恢复文件模式。

## 2026-07-22 - Task: 完成第 6 阶段生产部署与支付契约验收

### What was done

- 对 103 当前发布、磁盘、容器、端口和发布目录完成只读检查，在任何迁移前备份平台库、New API 库、Compose、版本和受限环境文件。
- 上传并校验 `5aa9fa406f47` 提交归档，从不可变发布目录构建并启动六服务，API 自动执行第 4 份支付迁移。
- 依次完成内网与公网基础验收、账号、商户、路由和支付契约，核对支付表与唯一约束后原子切换 `current`。
- 保持生产支付配置为空，没有写入聊天中出现过的旧 PID 或 KEY；清理服务器临时发布归档并保留已校验的发布前备份。

### Testing

- 发布前备份 `20260722T094226Z` 中两个数据库、Compose、版本和修改前环境文件全部通过 SHA-256 校验，文件权限均为 `600`。
- 本地与服务器发布归档 SHA-256 均为 `06625ad0ecb1f96e4513504fef9f5fd0936371cac05a78de695a18ad204edcf2`；Compose 构建和启动退出码为 `0`。
- `verify.sh` 在 `http://127.0.0.1:28100` 和 `https://aggregate.lwylink.xyz` 均通过；账号、商户、路由和支付四个专项契约全部通过。
- `/api/version` 返回 `5aa9fa406f47`，`current` 指向对应发布目录；平台迁移记录为 `4`，四张支付表存在，两个部分唯一索引和六个表级唯一约束存在。
- 支付配置、订单、回调事件和入账操作行数均为 `0`，有效临时路由绑定为 `0`；生产未引入测试支付数据。
- 网关、New API、平台 API、Worker、PostgreSQL 和 Redis 共六个服务运行，5 个健康检查均为 `healthy`；宿主机只监听 `127.0.0.1:28100`，内部端口未发布。
- 稳定窗口内 ModelRoute 网关 `499/5xx` 数量为 `0`，API、Worker 和 New API 日志没有 `panic`、`fatal` 或充值入账周期异常；三个现有域名均返回 `200`。

### Notes

- `docs/易支付充值.md`：补充生产版本、迁移、空配置、契约、稳定窗口和备份验收事实。
- `docs/部署环境审计.md`：更新支付迁移、生产凭据边界和当前运行版本。
- `progress.md`：追加第 6 阶段生产部署、验收、文件清单和回滚记录。
- 服务器 `/opt/modelroute-platform/releases/5aa9fa406f47`：保存第 6 阶段已验收运行发布。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T094226Z`：保存发布前两个数据库、Compose、版本和修改前环境文件备份。
- 回滚点：服务器将 `/opt/modelroute-platform/current` 原子切换到 `/opt/modelroute-platform/releases/6dd0565` 并执行该目录的 `deploy/scripts/deploy.sh`；需要连数据库回退时，使用上述发布前备份和 `restore.sh`，再恢复 `environment.before.env`。

## 2026-07-22 - Task: 实现第 7 阶段结算同步与人工提现

### What was done

- 建立结算同步水位、New API 原始日志来源、退款分配、对账队列、商户余额、只追加账本和人工提现数据结构，Worker 每 10 秒同步最终消费与退款日志。
- 以 New API 最终整数额度作为唯一计费事实，缓存命中不在平台重算；消费按整数拆分平台 10% 和商户剩余额度，部分退款按原消费拆分累计精确冲正。
- 实现 5 分钟重叠窗口、5 秒安全延迟、日志编号幂等和成功后推进水位；报价映射恢复后自动重试开放的对账项。
- 补齐异步任务退款的保守关联：精确标识缺失时只关联同归属的唯一无标识消费，候选不唯一或额度不足继续进入人工对账。
- 实现商户余额、标准模型名账本、提现申请和原子冻结；管理员可审核、驳回、确认线下打款和忽略对账异常，所有写操作检查同源并记录审计。
- 收款账户使用平台主密钥加密保存，商户只得到掩码，管理员接口才解密明文，密文字段不进入 JSON；商户账本不暴露内部报价别名。
- 商户中心和管理员提现页接入真实接口，删除提现演示数据、虚假数量和无实现导出入口，增加桌面与移动端回归截图及只读生产验收脚本。
- 新增中文结算文档并更新部署运行手册；本轮只完成本地实现和验证，生产仍运行第 6 阶段版本，待提交后先备份再发布。

### Testing

- 测试先行验证唯一候选退款、歧义候选、额度不足和商户账本模型名：现有实现分别失败为 `refund_origin_missing` 和内部报价别名，修改后专项 PostgreSQL 用例全部通过。
- 使用本机 PostgreSQL 17 容器执行 `go test ./...`，全部通过；真实 SQL 用例覆盖 8 个并发重叠日志只结算一次、两笔并发提现只冻结一笔、两次部分退款精确归零、映射恢复自动结算和账本禁止更新。
- 使用同一真实数据库执行 `go test -race ./...`，全部通过；`go vet ./...` 与 `go build ./cmd/...` 退出码均为 `0`。
- 执行 `npm.cmd run test:ui`，Vite 转换 `1608` 个模块，角色权限、真实账号、路由、报价、支付、结算和人工提现交互通过。
- 执行 `npm.cmd run capture`，生成 8 个业务视图、6 个专项管理视图和 6 个关键弹窗，脚本确认根页面无横向溢出；人工检查桌面提现弹窗与移动提现审核页布局。
- 执行部署测试共 `9` 项，全部通过；使用只读 Docker 挂载对 9 个部署 Shell 脚本逐一执行真实 `sh -n`，全部通过。
- `git diff --cached --check`、冲突标记、已出现的 SSH/易支付凭据字面值和私钥特征扫描均无异常；`verify-settlement.sh` 暂存模式为 `100755`。

### Notes

- `apps/api/cmd/worker/main.go`：把结算同步周期接入平台 Worker。
- `apps/api/internal/bootstrap/resources.go`：注入结算仓储与同步任务。
- `apps/api/internal/bootstrap/resources_test.go`：验证结算依赖完整注入。
- `apps/api/internal/httpapi/server.go`：注册商户结算、提现以及管理员提现和对账接口。
- `apps/api/internal/httpapi/services.go`：增加结算仓储与同步周期依赖契约。
- `apps/api/internal/httpapi/settlement_domain.go`：实现整数拆分、退款比例冲正、日志解析和同步窗口。
- `apps/api/internal/httpapi/settlement_domain_test.go`：覆盖整数拆分、部分退款、日志元数据和同步窗口。
- `apps/api/internal/httpapi/settlement_handlers.go`：实现结算、提现、管理员审核、打款和对账 HTTP 入口。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：覆盖角色权限、同源检查、输入验证、敏感字段和管理操作。
- `apps/api/internal/httpapi/settlement_reconciliation.go`：实现管理员对账列表和忽略处置。
- `apps/api/internal/httpapi/settlement_store.go`：实现日志幂等、消费结算、退款关联、对账重试和余额更新。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：以真实 PostgreSQL 覆盖并发、退款、提现、对账恢复和内部别名隐藏。
- `apps/api/internal/httpapi/settlement_sync.go`：实现 New API 日志分页同步和水位推进。
- `apps/api/internal/httpapi/settlement_sync_test.go`：覆盖分页、类型筛选、水位、失败记录和字段校验。
- `apps/api/internal/httpapi/settlement_types.go`：定义结算、账本、提现、对账数据与仓储接口。
- `apps/api/internal/httpapi/settlement_withdrawals.go`：实现商户余额、标准模型名账本和人工提现状态事务。
- `apps/api/internal/migrations/migrate_test.go`：验证第 5 份迁移顺序和关键表。
- `apps/api/internal/migrations/sql/000005_settlement.sql`：创建结算、余额、账本、提现和对账表、索引与只追加触发器。
- `apps/web/scripts/capture.mjs`：增加结算、提现和对账真实接口形状及专项截图。
- `apps/web/scripts/ui-smoke.mjs`：覆盖提现申请、管理员审核、打款、驳回、对账处置和敏感字段边界。
- `apps/web/src/Modals.jsx`：实现商户提现和管理员提现、对账操作弹窗。
- `apps/web/src/data.js`：删除提现演示数据。
- `apps/web/src/views/AdminView.jsx`：移除提现假数量并接入真实操作弹窗。
- `apps/web/src/views/MerchantView.jsx`：展示真实余额、标准模型账本、提现记录和申请入口。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：接入真实提现审核与结算对账队列。
- `apps/web/artifacts/admin-channels-desktop.png`：随最终构建更新管理员渠道桌面回归图。
- `apps/web/artifacts/admin-channels-mobile.png`：随最终构建更新管理员渠道移动回归图。
- `apps/web/artifacts/admin-desktop.png`：随最终构建更新管理员首页桌面回归图。
- `apps/web/artifacts/admin-mobile.png`：随最终构建更新管理员首页移动回归图。
- `apps/web/artifacts/admin-models-desktop.png`：随最终构建更新管理员模型桌面回归图。
- `apps/web/artifacts/merchant-desktop.png`：增加结算账本和提现记录的商户桌面回归图。
- `apps/web/artifacts/merchant-mobile.png`：增加结算账本和提现记录的商户移动回归图。
- `apps/web/artifacts/payment-config-desktop.png`：随最终构建更新支付配置桌面回归图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：新增提现审核与对账桌面回归图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：新增提现审核与对账移动回归图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：新增商户提现桌面弹窗回归图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：新增商户提现移动弹窗回归图。
- `deploy/scripts/verify-settlement.sh`：新增只读生产结算权限、列表、敏感字段和同源拦截验收。
- `deploy/tests/test_compose.py`：锁定结算验收脚本只读和权限边界。
- `docs/结算与人工提现.md`：记录账务口径、同步、退款关联、提现状态、接口和运维方法。
- `docs/部署运行手册.md`：增加第 5 份迁移和结算专项生产验收步骤。
- `progress.md`：追加第 7 阶段本地实现、验证、文件清单和回滚记录。
- 回滚点：阶段标签 `phase-6-payments`；第 7 阶段提交后可执行 `git revert <第7阶段提交>` 生成代码回退改动，生产发布前仍以 `/opt/modelroute-platform/releases/5aa9fa406f47` 为运行回滚点。

## 2026-07-22 - Task: 完成第 7 阶段生产部署与结算契约验收

### What was done

- 对 103 当前发布、磁盘、容器、宿主机端口、发布目录和既有备份完成只读检查，在任何迁移前备份平台库、New API 库、Compose、版本和受限环境文件。
- 上传并校验 `b147266e6464` 提交归档，验证归档路径边界后从不可变发布目录构建并启动六服务，平台 API 自动执行第 5 份结算迁移。
- 核对结算表、只追加触发器、Worker 同步水位和日志后，依次完成内网、公网、账号、商户、路由、支付和结算契约，再原子切换 `current`。
- 清理服务器和本机临时发布归档及临时口令助手，保留已校验的发布前备份和上一生产发布作为回滚点。

### Testing

- 发布前备份 `20260722T104834Z` 中两个数据库、Compose、版本和修改前环境文件全部通过 SHA-256 校验，文件权限均为 `600`。
- 本地与服务器发布归档 SHA-256 均为 `cb1f7a18ea6862114ac184c5db1d340f72692f1928ab57335d5bbdbaf24e4b8f`；归档条目路径检查、Compose 构建和启动退出码均为 `0`。
- `/api/version` 返回 `b147266e6464`，`current` 指向对应发布目录；平台迁移记录为 `1,2,3,4,5`，7 张结算表和只追加账本触发器存在。
- Worker 首次周期已推进同步水位且错误字段为空；发布验收时结算来源和开放对账项均为 `0`，未向生产写入测试结算账目。
- 内网与公网基础验收以及账号、商户、路由、支付、结算 5 个专项契约全部通过；结算专项保持只读，没有提交提现或处置真实对账项。
- 网关、New API、平台 API、Worker、PostgreSQL 和 Redis 共六个服务运行，5 个健康检查均为 `healthy`；宿主机只监听 `127.0.0.1:28100`，内部端口未发布。
- 三个现有域名均返回 `200`；路由契约预期产生一条无合格候选 `503`，后续新稳定窗口内网关 `499/5xx` 和应用异常均为 `0`。

### Notes

- `docs/结算与人工提现.md`：补充第 7 阶段生产版本、迁移、契约、稳定窗口和备份验收事实。
- `docs/部署环境审计.md`：更新结算迁移、同步水位、生产权限边界和当前运行版本。
- `progress.md`：追加第 7 阶段生产部署、验收、清理和回滚记录。
- 服务器 `/opt/modelroute-platform/releases/b147266e6464`：保存第 7 阶段已验收运行发布。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T104834Z`：保存发布前两个数据库、Compose、版本和修改前环境文件备份。
- 回滚点：服务器将 `/opt/modelroute-platform/current` 原子切换到 `/opt/modelroute-platform/releases/5aa9fa406f47` 并执行该目录的 `deploy/scripts/deploy.sh`；需要连数据库回退时，使用上述发布前备份和 `restore.sh`，再恢复 `environment.before.env`。

## 2026-07-22 - Task: 实现第 8 阶段运营管理与上线加固

### What was done

- 用自研页面接入用户真实用量趋势、缓存 Token、调用日志和兑换码充值，继续以 New API 最终日志、整数额度和原生兑换结果为准。
- 用自研管理员页面覆盖真实运营总览、全平台用量、SMTP 与注册登录设置、分组倍率、兑换码生命周期、系统任务和渠道运行操作；所有写操作继续经过管理员身份、同源校验和审计记录。
- 在所有用户与管理员用量响应中把内部报价别名还原为标准模型名，映射缺失时使用统一公共名称，避免内部别名进入浏览器。
- 增加只读运营验收、只读健康端点压测、临时数据库恢复演练和受控版本回滚工具，并为当前固定 New API 版本建立独立容器契约环境。
- 使用真实 New API 容器验证初始化、用户、令牌、渠道、模型映射、倍率、Chat、Responses、Embeddings、流式响应、缓存计费、退款日志和无渠道错误；候选环境结束后删除容器、网络和卷。
- 新增中文运营与升级验收文档并更新版本和部署手册；本轮只完成本地实现与验证，生产仍运行第 7 阶段版本 `b147266e6464`。

### Testing

- 使用本机 PostgreSQL 17 测试库执行 `go test -count=1 ./...` 和 `go test -race -count=1 ./...`，全部通过；执行 `go vet ./...` 与 `go build ./cmd/...`，退出码均为 `0`。
- 执行 `npm.cmd run test:ui`，Vite 转换 `1614` 个模块，用户用量、兑换码、管理员总览、设置、分组、兑换码、任务、渠道操作和内部别名隐藏契约全部通过。
- 执行 `npm.cmd run capture` 生成桌面与移动端回归截图；根页面无横向溢出，人工检查新增运营页面未发现文字遮挡。
- 执行 `python -m unittest discover -s deploy/tests -v` 共 `16` 项，全部通过；候选契约 Python 程序编译、主 Compose 和候选 Compose 解析、所有部署 Shell 脚本语法检查均通过。
- 执行真实 New API 候选容器契约通过；验证上游最高正式标签仍为 `v1.0.0-rc.21`，标签对象为 `8a784d7c8f8da9041b30c6c6ea4c3b3df0d683de`，源码提交为 `bde9b2f44887d34ec54799ae191d50f97914359e`。
- 生产只读验收、压力测试、备份恢复和版本回滚演练将在本阶段提交发布后执行，本条记录不提前声明生产可用。

### Notes

- `apps/api/internal/bootstrap/resources.go`：注入管理员运营总览仓储和内部报价别名解析依赖。
- `apps/api/internal/httpapi/routing_store.go`：增加内部报价别名到标准模型名的批量解析。
- `apps/api/internal/httpapi/server.go`：注册用量、运营设置、分组、兑换码、系统任务和渠道运行接口。
- `apps/api/internal/httpapi/services.go`：扩展运营管理、总览和别名解析依赖契约。
- `apps/api/internal/httpapi/operations_handlers.go`：实现设置、分组、兑换码、系统任务与渠道运行操作。
- `apps/api/internal/httpapi/operations_handlers_test.go`：覆盖运营接口权限、同源、整数额度、密码隐藏和别名隐藏。
- `apps/api/internal/httpapi/overview_handlers.go`：实现管理员运营总览响应和依赖健康状态。
- `apps/api/internal/httpapi/overview_store.go`：从平台账本、路由、审核与同步数据生成真实运营快照。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：验证运营快照使用最终整数账本和真实队列数据。
- `apps/api/internal/httpapi/usage_handlers.go`：代理并清洗用户与管理员 New API 用量和日志响应。
- `apps/web/scripts/capture.mjs`：增加第 8 阶段真实接口数据形状和运营页面截图场景。
- `apps/web/scripts/ui-smoke.mjs`：覆盖第 8 阶段页面、操作、权限与内部别名隐藏契约。
- `apps/web/src/data.js`：删除最后一批已由真实运营接口替代的演示数据。
- `apps/web/src/main.jsx`：传递账户刷新能力并接入新增真实页面。
- `apps/web/src/styles.css`：补充用量、兑换码和运营页面的响应式样式。
- `apps/web/src/ui.jsx`：补充运营状态所需的统一显示语义。
- `apps/web/src/views/AdminView.jsx`：接入真实运营总览及设置、分组、兑换码、任务和用量导航。
- `apps/web/src/views/ConsoleView.jsx`：展示真实用量、调用日志与兑换码充值入口。
- `apps/web/src/views/admin/AdminChannels.jsx`：展示渠道运行数据并提供测试和余额刷新操作。
- `apps/web/src/views/UsagePanels.jsx`：封装用户与管理员共用的真实用量趋势和日志表格。
- `apps/web/src/views/admin/AdminGroups.jsx`：实现整数基点分组倍率管理页面。
- `apps/web/src/views/admin/AdminRedemptions.jsx`：实现兑换码创建、编辑、启停、删除和清理页面。
- `apps/web/src/views/admin/AdminSettings.jsx`：实现 SMTP 与注册登录设置页面并保持密码只写不读。
- `apps/web/src/views/admin/AdminTasks.jsx`：实现系统任务列表、详情和日志清理任务页面。
- `apps/web/src/views/admin/AdminUsage.jsx`：实现全平台真实用量与调用日志页面。
- `apps/web/artifacts/admin-channels-desktop.png`：更新渠道运行操作桌面回归图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新渠道运行操作移动端回归图。
- `apps/web/artifacts/admin-desktop.png`：更新真实运营总览桌面回归图。
- `apps/web/artifacts/admin-mobile.png`：更新真实运营总览移动端回归图。
- `apps/web/artifacts/admin-models-desktop.png`：随最终页面构建更新模型管理桌面回归图。
- `apps/web/artifacts/admin-models-mobile.png`：随最终页面构建更新模型管理移动端回归图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：随最终页面构建更新提现管理桌面回归图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：随最终页面构建更新提现管理移动端回归图。
- `apps/web/artifacts/console-desktop.png`：更新用户用量和兑换码桌面回归图。
- `apps/web/artifacts/console-mobile.png`：更新用户用量和兑换码移动端回归图。
- `apps/web/artifacts/merchant-mobile.png`：随最终页面构建更新商户中心移动端回归图。
- `apps/web/artifacts/payment-config-desktop.png`：随最终页面构建更新支付配置桌面回归图。
- `apps/web/artifacts/route-modal-desktop.png`：随最终页面构建更新路由弹窗桌面回归图。
- `apps/web/artifacts/route-modal-mobile.png`：随最终页面构建更新路由弹窗移动端回归图。
- `apps/web/artifacts/admin-groups-desktop.png`：新增分组倍率桌面回归图。
- `apps/web/artifacts/admin-groups-mobile.png`：新增分组倍率移动端回归图。
- `apps/web/artifacts/admin-redemptions-desktop.png`：新增兑换码管理桌面回归图。
- `apps/web/artifacts/admin-redemptions-mobile.png`：新增兑换码管理移动端回归图。
- `apps/web/artifacts/admin-settings-desktop.png`：新增平台设置桌面回归图。
- `apps/web/artifacts/admin-settings-mobile.png`：新增平台设置移动端回归图。
- `apps/web/artifacts/admin-tasks-desktop.png`：新增系统任务桌面回归图。
- `apps/web/artifacts/admin-tasks-mobile.png`：新增系统任务移动端回归图。
- `apps/web/artifacts/admin-usage-desktop.png`：新增管理员用量桌面回归图。
- `apps/web/artifacts/admin-usage-mobile.png`：新增管理员用量移动端回归图。
- `deploy/newapi-contract/compose.yaml`：定义与生产隔离的 New API 候选契约环境。
- `deploy/newapi-contract/fake_upstream.py`：提供三类协议、流式和缓存用量固定上游响应。
- `deploy/newapi-contract/run_contract.py`：执行真实 New API 初始化、管理、转发、计费和错误契约。
- `deploy/scripts/load-test.sh`：增加只请求健康端点的有界并发压测。
- `deploy/scripts/rollback-release.sh`：增加确认变量、目录边界和失败恢复保护的版本回滚。
- `deploy/scripts/verify-backup-restore.sh`：在自动生成的临时数据库执行备份恢复演练。
- `deploy/scripts/verify-newapi-contract.sh`：启动、执行并清理独立 New API 候选契约环境。
- `deploy/scripts/verify-operations.sh`：增加第 8 阶段只读生产运营验收。
- `deploy/tests/test_release_tools.py`：锁定运营验收、压测、恢复、回滚和候选契约工具边界。
- `docs/New API版本与许可.md`：记录固定版本、源码提交、镜像摘要和候选契约结果。
- `docs/部署运行手册.md`：增加第 8 阶段发布、运营验收、压测、恢复演练和回滚步骤。
- `docs/运营管理与升级验收.md`：记录运营功能、权限边界、升级契约和发布工具用法。
- `progress.md`：追加第 8 阶段本地实现、验证、文件清单和回滚点。
- 回滚点：阶段标签 `phase-7-settlement`；第 8 阶段提交后可执行 `git revert <第8阶段提交>` 生成代码回退改动。生产服务器在本轮记录时仍运行第 7 阶段版本 `b147266e6464`。

## 2026-07-22 - Task: 修正第 8 阶段备份恢复演练迁移表名

### What was done

- 将备份恢复演练的迁移记录检查改为平台实际使用的 `platform_schema_migrations` 表，避免数据库已成功恢复后因检查错误而误报失败。
- 增加部署工具契约断言，锁定恢复演练必须检查平台迁移表真实名称，且不得再次使用错误的通用表名。
- 103 首次演练只恢复到自动生成的临时数据库并触发清理钩子，生产数据库和当前第 7 阶段服务未被修改；真实演练将在修复提交发布目录生成后重新执行。

### Testing

- 新增断言先在旧脚本上稳定失败，错误明确为缺少 `FROM platform_schema_migrations`；修改脚本后专项测试通过。
- 执行 `python -m unittest discover -s deploy/tests -v` 共 `16` 项，全部通过。
- 修正后的 `verify-backup-restore.sh` 通过 `sh -n` 语法检查，`git diff --check` 通过。

### Notes

- `deploy/scripts/verify-backup-restore.sh`：查询平台真实迁移记录表。
- `deploy/tests/test_release_tools.py`：增加正确迁移表名和错误表名不得出现的回归断言。
- `progress.md`：追加恢复演练问题、修正、验证和生产边界记录。
- 回滚点：第 8 阶段提交 `dd7d0db`；本轮修复提交可单独执行 `git revert <本轮修复提交>`，生产服务器仍运行 `/opt/modelroute-platform/releases/b147266e6464`。

## 2026-07-22 - Task: 修正就绪探测并发触发 New API 限流

### What was done

- 在真实回滚并恢复第 8 阶段后重跑压力测试，定位到 `/api/health/ready` 每次请求都会实时调用 New API `/api/status`，短时间并发触发上游状态接口 `429`，平台因此临时返回 `503`。
- 为平台就绪检查增加 1 秒共享结果缓存和互斥合并；同一秒的并发请求共用一次数据库、Redis 和 New API 探测，仍保留真实依赖失败响应。
- New API 源码、限流设置和模型调用链保持不变；修正仅作用于平台健康检查，避免发布验收和监控自身造成依赖限流。

### Testing

- 新增并发回归测试在旧实现上稳定记录数据库、Redis 和 New API 各探测 `50` 次并失败；修正后三类依赖各只探测 `1` 次并通过。
- 就绪依赖失败原有测试通过；并发专项执行 `go test -race` 通过。
- 使用本机 PostgreSQL 17 测试库执行 `go test -count=1 ./...` 和 `go test -race -count=1 ./...`，全部通过；`go vet ./...` 与 `go build ./cmd/...` 通过。
- 执行部署工具测试共 `16` 项，全部通过；`git diff --check` 通过。
- 生产重跑压力测试和稳定窗口验收将在本轮修复提交发布后执行，本条记录不提前声明问题已在线上消除。

### Notes

- `apps/api/internal/httpapi/server.go`：对就绪依赖探测增加 1 秒并发共享缓存。
- `apps/api/internal/httpapi/server_test.go`：增加 50 个并发就绪请求只触发一次依赖探测的竞态回归测试。
- `progress.md`：追加生产压测问题、根因、修正、验证和发布边界记录。
- 回滚点：生产版本 `5eece405533d`；本轮修复提交可单独执行 `git revert <本轮修复提交>`，正式回滚仍可使用发布前备份 `20260722T125523Z` 和第 7 阶段目录 `b147266e6464`。

## 2026-07-22 - Task: 完成第 8 阶段生产部署、恢复、回滚与运营验收

### What was done

- 对 103 当前发布、容器、宿主机端口、磁盘、发布目录、备份和受限文件权限完成只读审计，确认第 7 阶段运行正常且同机 Sub2API 未受影响。
- 在每次生产变更前分别创建备份 `20260722T125523Z` 和 `20260722T131751Z`，保存平台库、New API 库、解析后的 Compose、当前版本和受限环境副本，并统一生成 SHA-256 清单。
- 上传并双端校验最终提交归档，从不可变目录 `/opt/modelroute-platform/releases/46082cb062dc` 构建六服务；基础验收和全部专项契约通过后原子切换 `current`。
- 使用发布前备份在自动生成的两个临时数据库中完成真实恢复演练，检查平台与 New API 表和平台迁移记录后删除临时数据库，生产数据库未作为恢复目标。
- 使用受控回滚工具真实切回第 7 阶段 `b147266e6464`，核对内外网基础验收和线上版本后再恢复第 8 阶段，证明代码回滚和重新发布路径可执行。
- 在回切后的压力测试中发现就绪探测触发 New API 状态限流，按测试先行修正并通过新提交重新发布，没有直接修改服务器代码或移动已发布标签。
- 最终完成账号、商户、路由、支付、结算和运营专项契约、双轮只读压力测试、数据库状态、端口隔离、三个域名和 60 秒稳定窗口验收。

### Testing

- 发布归档本地与服务器 SHA-256 均为 `643284f7f3cc5110db1b5c41c46e01f24bafe98b61bb004ec899c17745f82d4d`；服务器归档路径检查、Compose 解析、构建和启动退出码均为 `0`。
- 两份备份的数据库、Compose、版本和环境副本均通过 SHA-256 校验且权限为 `600`；恢复演练得到平台库 `28` 张表、New API 库 `31` 张表，演练数据库残留为 `0`。
- 内网和公网 `verify.sh` 通过；账号、商户、路由、支付、结算和运营六组专项脚本全部通过，运营与结算验收保持只读。
- 受控回滚到 `b147266e6464` 后 `/api/version` 和 `current` 一致，内外网基础验收通过；恢复第 8 阶段后再次通过全部专项契约。
- 最终连续两轮执行 `200` 次、并发 `10` 的只读压力测试，均成功 `200/200`，P95 分别为 `0.004034` 秒和 `0.004217` 秒；验收窗口内 New API `/api/status` 的 `429` 数量为 `0`。
- `/api/version` 返回 `46082cb062dc`，`current` 指向对应发布目录；平台迁移为 `1,2,3,4,5`，结算同步水位为 `1784726543` 且错误为空。
- 支付配置、订单、回调和入账操作均为 `0`，有效临时路由绑定为 `0`；生产未写入支付配置或测试支付数据。
- ModelRoute 六服务运行，五项健康检查均为 `healthy`；宿主机只监听 `127.0.0.1:28100`，New API、PostgreSQL、Redis 和平台内部端口未发布。
- `aggregate.lwylink.xyz` 的 HTTP 返回 `308`、HTTPS 返回 `200`，`api.lwylink.xyz` 与 `sdk.lwylink.xyz` 均返回 `200`；60 秒稳定窗口内网关 `499/5xx`、平台 API、Worker 和 New API 状态限流数量均为 `0`。

### Notes

- `docs/运营管理与升级验收.md`：记录最终生产版本、备份恢复、专项契约、压力测试、回滚和稳定窗口证据。
- `docs/部署环境审计.md`：更新第 8 阶段当前发布、权限、数据、端口和同机服务隔离状态。
- `docs/部署运行手册.md`：补充就绪探测并发共享语义和 New API 状态限流核对要求。
- `progress.md`：追加第 8 阶段生产部署、恢复、回滚、验收和回滚点记录。
- 服务器 `/opt/modelroute-platform/releases/46082cb062dc`：保存第 8 阶段最终已验收运行发布。
- 服务器 `/opt/modelroute-platform/releases/5eece405533d`：保存最终修复前的第 8 阶段发布，可用于代码级回退核对。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T125523Z`：保存第 8 阶段首次发布前完整备份并已通过恢复演练。
- 服务器 `/opt/modelroute-platform/shared/backups/20260722T131751Z`：保存最终修复发布前完整备份。
- 回滚点：优先使用 `/opt/modelroute-platform/releases/5eece405533d` 做同迁移版本代码回退；需要回到第 7 阶段时使用 `/opt/modelroute-platform/releases/b147266e6464`，涉及数据恢复时使用对应发布前备份和 `restore.sh`。

## 2026-07-22 - Task: 轮换 103 SSH 运维凭据

### What was done

- 为 103 生成独立 Ed25519 运维密钥，追加到 root `authorized_keys` 前后均保持目录 `700`、文件 `600`，并完成纯密钥登录验证。
- 生成新的高强度随机 root 密码，通过受限临时文件和已验证密钥通道提交给服务器，执行完成后删除本机与服务器临时输入文件。
- 将新密码保存在工作区外的本机受限文件中，并把私钥与密码文件 ACL 收紧为仅当前 Windows 用户完全控制。
- 分别验证新密码登录成功、旧密码认证失败、专用密钥在轮换后仍可登录，并通过密钥再次读取生产就绪状态。

### Testing

- 专用 Ed25519 密钥指纹检查通过，服务器登记数量为 `1`；纯密钥登录在密码轮换前后均成功。
- 新密码强制使用密码认证登录成功；旧密码强制使用密码认证返回退出码 `255`。
- 本机私钥和密码文件 ACL 均只包含当前 Windows 用户；临时口令助手和密码传输文件残留为 `0`。
- 轮换后 `/api/health/ready` 返回 `200/ready`，生产版本、容器和业务配置未发生变化。

### Notes

- `docs/部署环境审计.md`：记录 SSH 密码轮换、密钥登录、受限文件位置和生产健康证据。
- `docs/部署运行手册.md`：增加 103 专用运维密钥、密码保管和后续轮换步骤。
- `progress.md`：追加 SSH 凭据轮换、验证、文件清单和回滚方法。
- `C:\Users\Administrator\.ssh\modelroute_103_ed25519`：新增 103 专用私钥，仅当前 Windows 用户可访问，不进入 Git。
- `C:\Users\Administrator\.ssh\modelroute_103_ed25519.pub`：新增对应公钥并登记到服务器 root `authorized_keys`。
- `C:\Users\Administrator\.ssh\modelroute_103_root_password.txt`：保存轮换后的 root 密码，仅当前 Windows 用户可访问，不进入 Git。
- 回滚方式：保持专用密钥可用，通过该密钥再次执行受控密码轮换；只有另一条管理通道已验证时，才从服务器 `authorized_keys` 删除本轮公钥并删除上述本机文件。

## 2026-07-22 - Task: 落实唯一管理员直属普通商户

### What was done

- 将 ModelRoute 管理员身份收口为 `NEW_API_ADMIN_USERNAME` 指定的唯一账号；其他 New API 管理角色进入平台后按普通用户处理，登录、二次验证、资料刷新和接口会话校验使用同一规则。
- 平台启动时核对唯一管理员账号的编号、用户名、状态和角色，并幂等创建其直属“ModelRoute 官方”真实商户资料；核对或创建失败时停止启动。
- 管理员与普通商户共用同一套商户身份、接口和操作，删除管理员缺少商户记录时的临时放行；管理员只额外拥有管理后台。
- 新增第 6 号迁移，允许平台自营商户不依赖普通申请记录，全库只保留一个平台自营商户，并把渠道审核关联到真实商户编号。
- `/api/me` 增加商户名称和平台自营标识，供后续同一商户工作区直接使用。

### Testing

- 新增测试先在旧实现上稳定失败，失败点为缺少第 6 号迁移、唯一管理员配置、平台商户创建方法和启动引导函数；实现后聚焦测试全部通过。
- 使用本机 PostgreSQL 17 测试库真实执行 `000006_platform_merchant.sql`，`go test -count=1 ./internal/httpapi` 通过，现有商户、路由、支付、结算集成场景保持兼容。
- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./cmd/...`，全部通过。
- 执行 `go test -race -count=1 ./internal/httpapi ./internal/bootstrap`，全部通过；`git diff --check` 将在提交前执行。

### Notes

- `apps/api/internal/bootstrap/resources.go`：注入唯一管理员用户名，并在服务启动时核验管理员和创建直属官方商户。
- `apps/api/internal/bootstrap/resources_test.go`：覆盖唯一管理员直属商户启动引导。
- `apps/api/internal/httpapi/auth_handlers.go`：统一归一化登录、二次验证、资料刷新和会话中的管理员角色。
- `apps/api/internal/httpapi/auth_test.go`：锁定非配置管理员用户名不得保留平台管理员权限。
- `apps/api/internal/httpapi/merchant_handlers.go`：删除管理员缺少真实商户资料时的特殊放行。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖管理员真实商户要求及 `/api/me` 商户字段。
- `apps/api/internal/httpapi/merchant_store.go`：实现官方商户幂等创建、真实身份读取和渠道审核商户关联。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖官方商户创建和审批关联真实商户编号。
- `apps/api/internal/httpapi/merchant_types.go`：扩展商户身份字段和仓储契约。
- `apps/api/internal/httpapi/services.go`：增加唯一管理员用户名服务配置。
- `apps/api/internal/migrations/migrate_test.go`：锁定第 6 号迁移顺序和关键约束。
- `apps/api/internal/migrations/sql/000006_platform_merchant.sql`：增加平台自营商户和渠道审核归属结构。
- `docs/账号与管理员接口.md`：记录唯一管理员角色边界和普通商户复用规则。
- `docs/商户与渠道审核接口.md`：记录管理员真实商户身份、相同商户接口和审核归属。
- `docs/部署运行手册.md`：更新唯一管理员配置和启动失败边界。
- `docs/superpowers/plans/2026-07-22-ModelRoute完整实施计划.md`：把原计划修正为唯一管理员直属普通商户。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：基线提交 `13a03b1`；本阶段提交后可执行 `git revert <本阶段提交>` 生成代码回退改动。第 6 号迁移包含数据库结构变更，生产尚未发布；未来发布后若需跨迁移回退，应使用同一发布前的平台库与 New API 库备份恢复。

## 2026-07-22 - Task: 接入公告中心与管理员公告管理

### What was done

- 新增普通用户、商户和管理员互斥受众的公告系统，支持草稿、定时发布、已发布、已下架、置顶、自动过期和幂等已读。
- 顶部铃铛接入真实公告列表和未读角标，打开详情时标记已读；移除无行为帮助按钮和头像下拉三角，头像点击直接进入控制台。
- 管理后台增加公告管理页，可新建、编辑、发布和下架纯文本公告，所有写操作记录公告对象审计。
- 新增第 7 号迁移和 PostgreSQL 可见性测试，定时生效与过期隐藏直接由查询时间判断，不新增 Worker 状态任务。

### Testing

- 后端测试先在旧实现上稳定失败，失败点为缺少第 7 号迁移、公告仓储、接口和服务注入；公告审计回归测试还确认旧辅助函数会把对象误记为用户，修正后全部转绿。
- 前端冒烟测试先在旧界面上稳定失败，失败点为仍存在无行为帮助按钮；实现后验证公告已读、头像跳转、公告管理和创建受众提交均通过。
- 使用本机 PostgreSQL 17 测试库执行 `go test -count=1 ./...`，真实覆盖定时前不可见、到时可见、过期隐藏、受众隔离和重复已读幂等。
- 执行 `go vet ./...`、`go build ./cmd/...` 和 `go test -race -count=1 ./internal/httpapi ./internal/bootstrap`，全部通过。
- 执行 `npm.cmd run test:ui` 和 `npm.cmd run capture`，桌面与移动端公告面板和公告管理页均生成有效截图，根页面无横向溢出；人工目检未发现文字遮挡或控件重叠。

### Notes

- `apps/api/internal/bootstrap/resources.go`：注入公告 SQL 仓储。
- `apps/api/internal/httpapi/announcement_handlers.go`：实现用户公告、已读和管理员公告写接口。
- `apps/api/internal/httpapi/announcement_handlers_test.go`：覆盖互斥受众、定时创建和公告审计对象。
- `apps/api/internal/httpapi/announcement_store.go`：实现公告可见性、已读幂等和管理状态写入。
- `apps/api/internal/httpapi/announcement_store_postgres_test.go`：覆盖真实 PostgreSQL 时间与已读行为。
- `apps/api/internal/httpapi/announcement_types.go`：定义公告领域类型、受众和仓储契约。
- `apps/api/internal/httpapi/server.go`：注册用户与管理员公告路由。
- `apps/api/internal/httpapi/services.go`：增加公告服务依赖。
- `apps/api/internal/migrations/migrate_test.go`：锁定第 7 号公告迁移。
- `apps/api/internal/migrations/sql/000007_announcements.sql`：创建公告和公告已读表。
- `apps/web/src/main.jsx`：接入公告面板、未读角标、头像跳转并删除无行为顶部控件。
- `apps/web/src/styles.css`：增加公告面板、公告管理和移动端样式。
- `apps/web/src/views/AdminView.jsx`：增加公告管理侧栏入口。
- `apps/web/src/views/admin/AdminAnnouncements.jsx`：实现公告列表、表单和状态操作。
- `apps/web/scripts/ui-smoke.mjs`：覆盖顶部公告、已读、头像跳转和管理员创建公告。
- `apps/web/scripts/capture.mjs`：增加桌面和移动端公告截图场景。
- `apps/web/artifacts/announcement-panel-desktop.png`：新增桌面公告面板回归图。
- `apps/web/artifacts/announcement-panel-mobile.png`：新增移动端公告面板回归图。
- `apps/web/artifacts/admin-announcements-desktop.png`：新增桌面公告管理回归图。
- `apps/web/artifacts/admin-announcements-mobile.png`：新增移动端公告管理回归图。
- `apps/web/artifacts/admin-channels-desktop.png`：更新顶部公告入口后的桌面回归图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新顶部公告入口后的移动端回归图。
- `apps/web/artifacts/admin-desktop.png`：更新管理员顶部栏桌面回归图。
- `apps/web/artifacts/admin-mobile.png`：更新管理员顶部栏移动端回归图。
- `apps/web/artifacts/admin-groups-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-groups-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-models-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-models-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-redemptions-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-redemptions-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-settings-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-settings-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-tasks-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-tasks-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-usage-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-usage-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：更新公告侧栏后的桌面回归图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：更新公告侧栏后的移动端回归图。
- `apps/web/artifacts/console-desktop.png`：更新头像与公告入口后的桌面回归图。
- `apps/web/artifacts/console-mobile.png`：更新移动端公告入口回归图。
- `apps/web/artifacts/market-desktop.png`：更新登录态顶部栏桌面回归图。
- `apps/web/artifacts/market-mobile.png`：更新登录态顶部栏移动端回归图。
- `apps/web/artifacts/merchant-desktop.png`：更新商户顶部栏桌面回归图。
- `apps/web/artifacts/merchant-mobile.png`：更新商户顶部栏移动端回归图。
- `apps/web/artifacts/payment-config-desktop.png`：更新支付弹窗顶部公告入口回归图。
- `apps/web/artifacts/route-modal-desktop.png`：更新路由弹窗顶部公告入口回归图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：更新提现弹窗顶部公告入口回归图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：更新提现弹窗移动端公告入口回归图。
- `docs/公告管理.md`：记录公告规则、接口、数据、页面和验证方式。
- `docs/部署运行手册.md`：把最低迁移验收更新到第 7 号公告迁移。
- `progress.md`：追加公告阶段实现、验证、文件清单和回滚点。
- 回滚点：上一阶段提交 `c24973f`；本阶段提交后可执行 `git revert <本阶段提交>` 生成代码回退改动。生产尚未执行第 7 号迁移；未来发布后跨迁移回退必须使用发布前双数据库备份。

## 2026-07-23 - Task: 升级绝对价格、价格阶梯与 New API 分层计费

### What was done

- 新增第 8 号迁移，为标准模型增加稳定版、Preview、限量和临近退役生命周期，为官方价格版本增加原币、独立核价汇率、核价来源和核价时间，并建立连续输入上下文价格阶梯。
- 将历史 Token 倍率换算为单档美元绝对价格，保留但停用历史固定计费模型；新模型只接受每百万 Token 输入、输出和缓存绝对价格。
- 实现美元和人民币微价的两步整数四舍五入、商户折扣最低边界、阶梯连续性与协议校验；稳定版默认启用，其他生命周期默认停用。
- 将报价同步从八组旧倍率配置切换为 New API 原生 `tiered_expr`，使用 `len/p/c/cr/cc/cc1h` 生成不可变表达式；同步失败时不更新渠道、不激活新报价。
- 模型目录返回原币阶梯，市场与商户报价返回折扣基点和折后美元阶梯；内部报价别名继续不返回浏览器。
- 更新固定 New API 容器契约，真实覆盖分层表达式、缓存命中、三类协议、流式响应和退款日志。

### Testing

- 测试先在旧实现上稳定失败，失败点为缺少第 8 号迁移、绝对价格类型、整数换算、生命周期状态、分层表达式和新同步键；实现后聚焦测试全部通过。
- 使用 PostgreSQL 17 执行完整迁移和全部数据库集成测试；模型目录、市场折后价和商户折后价断言通过。
- 在独立临时数据库先执行前 7 个迁移并写入旧 Token 与固定计费记录，再执行第 8 号迁移；旧 Token 得到单档 `$2.50/$7.50`，旧固定模型保持 `legacy_fixed` 且停用，临时数据库已删除。
- 执行 `go test -count=1 ./...`、`go vet ./...`、`go build ./cmd/...` 和 `git diff --check`，均通过。
- 在 New API 固定源码提交 `bde9b2f44887d34ec54799ae191d50f97914359e` 执行 `go test -count=1 ./pkg/billingexpr ./setting/billing_setting`，通过。
- 执行 `bash deploy/scripts/verify-newapi-contract.sh`，固定 `rc.21` 容器的初始化、账号、令牌、渠道、模型映射、分层表达式、Chat、Responses、Embeddings、流式、缓存日志、退款日志和无渠道错误契约全部通过。

### Notes

- `apps/api/go.mod`、`apps/api/go.sum`：引入与 New API `rc.21` 相同版本的 `expr-lang` 表达式编译器。
- `apps/api/internal/httpapi/absolute_pricing.go`：实现绝对价格校验、整数换算、折扣阶梯、生命周期初始状态和 `tiered_expr` 生成冒烟。
- `apps/api/internal/httpapi/absolute_pricing_test.go`：覆盖美元、人民币、四舍五入、阶梯边界、协议、厂商、生命周期和表达式。
- `apps/api/internal/httpapi/pricing_sync.go`：切换 New API 分层计费配置同步。
- `apps/api/internal/httpapi/pricing_sync_test.go`：覆盖现有配置保留、表达式生成、人民币 Embeddings 和失败不激活。
- `apps/api/internal/httpapi/routing_handlers.go`：接入生命周期与绝对价格 HTTP 契约。
- `apps/api/internal/httpapi/routing_handlers_test.go`：锁定管理员创建模型的新请求结构。
- `apps/api/internal/httpapi/routing_domain.go`、`apps/api/internal/httpapi/routing_domain_test.go`：删除本轮切换后不再使用的旧倍率缩放函数及测试。
- `apps/api/internal/httpapi/routing_store.go`：原子保存价格版本与阶梯，并返回模型和报价的分层价格。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖价格事务、别名装载和新价格发布。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：覆盖真实 PostgreSQL 模型目录与折后价格。
- `apps/api/internal/migrations/migrate_test.go`：锁定第 8 号迁移顺序和关键结构。
- `apps/api/internal/migrations/sql/000008_absolute_pricing.sql`：增加生命周期、绝对价格元数据、价格阶梯和旧数据迁移。
- `deploy/newapi-contract/run_contract.py`：把候选容器计费契约切换为分层表达式和分层日志断言。
- `deploy/scripts/verify-newapi-contract.sh`：更新候选容器契约结果说明。
- `deploy/tests/test_release_tools.py`：把候选契约静态断言更新为分层计费配置和日志字段。
- `docs/New API版本与许可.md`：更新候选版本分层计费覆盖范围。
- `docs/模型报价路由与健康.md`：更新绝对价格、阶梯、折扣和同步规则。
- `docs/绝对价格与分层计费.md`：新增绝对价格数据、换算、表达式、升级和验证说明。
- `docs/部署运行手册.md`：把最低迁移验收更新到第 8 号迁移。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：上一阶段提交 `b42bd57`；本阶段提交后可执行 `git revert <本阶段提交>` 生成代码回退改动。生产尚未执行第 8 号迁移或写入新表达式；未来发布后若跨迁移回退，必须恢复同一发布时间点的平台库与 New API 库备份。

## 2026-07-23 - Task: 统一商户工作区并接入绝对模型价格界面

### What was done

- 普通商户和唯一管理员统一使用同一个商户中心组件，增加经营概览、渠道管理、模型报价、健康状态、结算账本和提现记录六个侧栏入口；管理员只额外保留管理后台。
- 管理员模型表单删除倍率、固定计费和自由厂商输入，改为固定厂商与协议、生命周期、美元或人民币、独立核价汇率、核价来源、核价时间及每百万 Token 绝对价格。
- 支持连续上下文价格阶梯的新增和删除，自动生成下一档起点；Embeddings 只显示输入价格，文本协议支持输入、输出、缓存读取、5 分钟缓存写入和 1 小时缓存写入。
- 商户报价输入改为官方价折扣百分比，模型市场和商户报价页同时显示折扣及折后输入、输出美元单价；内部报价别名仍不进入浏览器。
- 更新桌面与移动端截图场景，新增商户报价工作区和管理员绝对价格弹窗专项截图；本阶段只修改外围前端，New API 源码、请求链路和计费链路保持不变。

### Testing

- 先扩展 `ui-smoke.mjs` 并在旧界面运行，稳定失败于“商户侧栏缺少经营概览”；实现后同一脚本验证唯一管理员和普通商户的六个入口一致、普通商户无管理后台、65% 折扣转换为 `6500` 基点、Embedding 单价请求体和两档连续价格请求体，最终通过。
- 执行 `npm.cmd run test:ui`，Vite 构建完成，角色导航、统一商户工作区、绝对价格、路由、支付、结算和人工提现交互全部通过。
- 执行 `npm.cmd run capture`，生成 1440x1000 桌面和 390x844 移动端截图；脚本确认根页面无横向溢出和运行时错误，人工检查商户侧栏、模型市场、模型列表与价格弹窗，未发现文字遮挡或控件重叠。
- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./cmd/...`，全部通过。
- 执行 16 个部署 Python 测试、`docker compose --env-file .env.example config --quiet` 和 `git diff --check`，全部通过。

### Notes

- `apps/web/src/api.js`：增加整数微价格和折后报价的统一显示格式。
- `apps/web/src/Modals.jsx`：把商户报价改为折扣百分比输入，并预览官方价与折后美元价。
- `apps/web/src/views/MarketView.jsx`：按折后首档价格排序并展示折后绝对价格和官方价折扣。
- `apps/web/src/views/MerchantView.jsx`：实现普通商户与管理员共用的六栏商户工作区。
- `apps/web/src/views/admin/AdminModels.jsx`：实现固定厂商、协议、生命周期和多阶梯绝对价格表单。
- `apps/web/src/styles.css`：增加厂商选择器、价格元数据和响应式阶梯编辑样式。
- `apps/web/scripts/ui-smoke.mjs`：覆盖统一商户权限、绝对价格提交、Embedding 和连续阶梯行为。
- `apps/web/scripts/capture.mjs`：改用绝对价格模拟数据并增加商户报价和模型价格弹窗截图。
- `docs/绝对价格与分层计费.md`：补充管理员价格界面和市场展示口径。
- `docs/模型报价路由与健康.md`：补充官方价折扣和统一商户工作区规则。
- `docs/账号与管理员接口.md`：补充唯一管理员与普通商户共用六栏页面的权限边界。
- `apps/web/artifacts/market-desktop.png`：更新桌面模型市场真实折后价格截图。
- `apps/web/artifacts/market-mobile.png`：更新移动端模型市场真实折后价格截图。
- `apps/web/artifacts/merchant-desktop.png`：更新桌面商户经营概览和侧栏截图。
- `apps/web/artifacts/merchant-mobile.png`：更新移动端商户经营概览和横向侧栏截图。
- `apps/web/artifacts/merchant-offers-desktop.png`：新增桌面商户模型报价工作区截图。
- `apps/web/artifacts/merchant-offers-mobile.png`：新增移动端商户模型报价工作区截图。
- `apps/web/artifacts/admin-models-desktop.png`：更新桌面管理员模型绝对价格列表截图。
- `apps/web/artifacts/admin-models-mobile.png`：更新移动端管理员模型绝对价格列表截图。
- `apps/web/artifacts/admin-model-price-desktop.png`：新增桌面管理员绝对价格弹窗截图。
- `apps/web/artifacts/admin-model-price-mobile.png`：新增移动端管理员绝对价格弹窗截图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：更新统一商户工作区背景下的桌面提现弹窗截图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：更新统一商户工作区背景下的移动端提现弹窗截图。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：基线提交 `83a0c01`；本阶段提交后可执行 `git revert <本阶段提交>` 生成代码回退改动。本阶段没有数据库迁移且尚未发布生产，不需要恢复数据库。

## 2026-07-23 - Task: 添加固定 New API 支持交集的版本化官方模型种子

### What was done

- 新增版本化官方模型种子 `2026-07-23-v1`，绑定 New API `v1.0.0-rc.21` 与源码提交 `bde9b2f44887d34ec54799ae191d50f97914359e`；首批收录 7 家厂商的 21 个模型，其中 19 个稳定版启用、2 个 Preview 停用。
- 实现种子结构、固定适配器支持交集、官方价格与阶梯校验；重复导入保持幂等，价格变化只追加新版本，模型身份冲突时终止。
- 新增一次性模型种子命令，通过 New API 原生登录读取配置中的有效唯一管理员，并以实际管理员编号创建模型和发布价格版本。
- 在 Docker 镜像中加入种子命令，并将 `model-seed` 作为 `tools` profile 的一次性服务；部署固定为先启动依赖、执行种子，再启动外围常驻服务。
- 为官方模型种子目录增加精确 Git 忽略例外，确保版本化 JSON 进入发布提交，同时不放开其他运行时 `data/` 目录。

### Testing

- 部署脚本回归测试先在缺少 `--profile tools` 的旧命令上稳定失败；修复后验证构建包含一次性服务，而最终常驻启动不启用 `tools` profile。
- 发布包回归测试先稳定捕获官方模型 JSON 被通用 `data/` 规则忽略；增加精确例外后转绿，Git 未跟踪清单已包含种子文件。
- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./cmd/...`，全部通过。
- 使用本机 PostgreSQL 17 独立测试库执行完整种子集成测试：首次创建 21 个模型和 21 个价格版本，重复执行保持 21/21；单模型调价由 1 个价格版本追加为 2 个，测试结束后清理种子数据。
- 对固定 New API 源码执行种子交集测试和 OpenAI、Anthropic、Gemini、DeepSeek、Qwen、Moonshot、Minimax 七个渠道包测试，固定提交与计划一致，全部通过。
- 执行 18 个部署 Python 测试、默认及 `tools` profile Compose 配置校验、API/Worker/种子镜像构建和镜像内三个可执行文件检查，全部通过；默认服务列表不含 `model-seed`。
- 执行 `gofmt` 差异检查、`git diff --check` 和变更敏感信息模式扫描，均未发现问题。
- 本轮实时核价链接复查受外部访问策略影响：DeepSeek、Kimi、MiniMax、Qwen 和 Claude 页面可返回 200，OpenAI 页面向命令行客户端返回 403，Gemini 当前网络不可达；平台不在生产自动抓取官网，确定性验收以版本化种子、固定源码交集和整数价格测试为准。

### Notes

- `.gitignore`：仅放行版本化官方模型种子 JSON，保留其他运行时数据目录忽略规则。
- `README.md`：增加官方模型种子模块入口。
- `apps/api/Dockerfile`：构建并复制一次性模型种子命令。
- `apps/api/cmd/seed-models/main.go`：实现配置加载、唯一管理员验证、迁移和种子执行入口。
- `apps/api/cmd/seed-models/main_test.go`：覆盖种子依赖配置和唯一管理员校验。
- `apps/api/internal/httpapi/absolute_pricing.go`：向受控种子模块开放标准模型复用校验入口。
- `apps/api/internal/modelseed/apply.go`：实现模型创建、幂等跳过、调价追加和版本元数据写入。
- `apps/api/internal/modelseed/apply_test.go`：覆盖首次导入、重复导入和调价追加。
- `apps/api/internal/modelseed/catalog.go`：实现嵌入式种子解析、摘要、固定版本和字段校验。
- `apps/api/internal/modelseed/catalog_test.go`：覆盖未知字段、支持清单、厂商协议、阶梯边界和人民币汇率。
- `apps/api/internal/modelseed/repository.go`：复用平台模型与价格仓储并事务写入种子元数据。
- `apps/api/internal/modelseed/repository_postgres_test.go`：覆盖真实 PostgreSQL 的完整种子幂等和价格版本追加。
- `apps/api/internal/modelseed/support_rc21.go`：固化七家渠道在 New API rc.21 中支持的首批模型交集。
- `apps/api/internal/modelseed/support_rc21_external_test.go`：逐模型核对固定 New API 渠道适配器源码。
- `apps/api/internal/modelseed/data/2026-07-23-v1.json`：记录 21 个模型、官方价格、阶梯、生命周期、来源和核价时间。
- `compose.yaml`：增加 `tools` profile 下的 `model-seed` 一次性服务。
- `deploy/scripts/deploy.sh`：加入依赖启动、模型种子和常驻服务的固定发布顺序。
- `deploy/tests/test_compose.py`：覆盖种子服务 profile、入口、依赖和非重启约束。
- `deploy/tests/test_deploy.py`：覆盖部署版本注入、profile 构建和六步调用顺序。
- `deploy/tests/test_release_tools.py`：锁定官方模型种子不得被 Git 忽略。
- `docs/官方模型种子.md`：记录当前目录、核价来源、未纳入原因、导入规则、执行和回滚方式。
- `docs/绝对价格与分层计费.md`：补充缓存写入口径和官方模型种子规则。
- `docs/部署运行手册.md`：补充一次性种子执行、生产验收和发布前核对项。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：上一阶段提交 `a97b38b`；本阶段提交后可执行 `git revert <本阶段提交>` 生成代码回退改动。生产尚未执行模型种子，本阶段回退不需要恢复数据库；未来生产已执行种子后若需恢复目录状态，必须恢复同一时间点的平台库与 New API 库备份。

## 2026-07-23 - Task: 修正唯一管理员启用后的商户生产契约

### What was done

- 修正商户验收脚本仍让唯一管理员提交普通商户申请的问题；管理员继续直接使用“ModelRoute 官方”商户，不创建管理员专用商户流程。
- 商户契约改为通过 103 内网 New API 原生接口创建一次性普通用户，由普通用户提交申请、管理员查询并驳回，完成后原生删除临时用户。
- 管理员、普通用户和 New API 管理会话使用三份独立 Cookie，凭据与临时渠道密钥只保存在权限受限的临时目录中。

### Testing

- 回归测试先在旧脚本上稳定失败，失败点为缺少普通用户与 New API 独立会话；实现后专项测试与 `bash -n` 语法检查通过。
- 新版本 `8526c3e6691b` 上线后，旧商户契约稳定返回 `409`；只读诊断确认唯一管理员角色为 `100`，且 `is_merchant=true`、`is_platform_merchant=true`，证明冲突来自验收脚本违反新业务规则。
- 将修正后的脚本放入 103 的临时发布目录，使用生产 Compose 与共享受限配置执行真实链路：普通用户申请、管理员列表、渠道密钥隐藏、管理员驳回和凭据清除全部通过。
- 契约结束后查询 New API 数据库，`mrc` 前缀临时用户残留为 `0`；临时脚本和临时发布目录已由 `trap` 清理，当前发布文件未被直接修改。

### Notes

- `deploy/scripts/verify-merchants.sh`：使用临时普通用户验证商户申请，并通过 New API 原生接口清理测试账号。
- `deploy/tests/test_release_tools.py`：锁定三会话、普通用户身份和临时账号原生创建删除约束。
- `docs/运营管理与升级验收.md`：补充商户契约的身份分工、内网依赖和清理规则。
- `progress.md`：追加本轮根因、实现、生产临时验证和回滚点。
- 回滚点：代码基线提交 `8526c3e`；本轮提交后可执行 `git revert <本轮提交>` 回退验收脚本。该修正不改数据库结构和业务接口，回退无需恢复数据库。

## 2026-07-23 - Task: 完成生产升级、唯一管理员轮换与全量验收

### What was done

- 将升级版本发布到 103 的 `/opt/modelroute-platform/releases/633a78e28c65`，执行第 6 至第 8 号迁移并导入版本化官方模型种子；New API 源码、镜像、模型请求和缓存计费链路保持原固定版本。
- 将全局唯一管理员登录名轮换为 `3180615598@qq.com`，清理全部平台会话并只重建平台 API 与 Worker；管理员继续使用与普通商户完全相同的商户中心，只额外显示管理后台。
- 新管理员凭据保存到服务器权限 `600` 的共享文件和本机受限文件，两个副本哈希一致；旧登录名已失效，临时明文恢复目录已删除，凭据正文未进入仓库和验收输出。
- 新增真实生产 Playwright 验收脚本，从受限凭据文件登录线上页面，覆盖桌面与移动端公告、头像、共用商户侧栏和绝对价格表单。

### Testing

- 内网基础、公网基础、账号、商户、路由、易支付、结算和运营管理八组生产契约全部通过。
- 公告生产权限矩阵通过：匿名读取和管理均为 `401`，普通用户读取为 `200`、管理为 `403`，管理员读取和管理均为 `200`；一次性普通用户通过 New API 原生接口删除，残留为 `0`。
- 平台迁移为 `6..8`；唯一平台商户数量为 `1`。模型种子版本为 `2026-07-23-v1`，共有 21 个模型、21 个当前绝对价格版本和 25 个连续阶梯，19 个启用、2 个 Preview 停用，阶梯异常数为 `0`。
- 连续两轮执行 `REQUESTS=200 CONCURRENCY=10 deploy/scripts/load-test.sh http://127.0.0.1:28100`，均成功 `200/200`；平均耗时为 `0.0019s`、`0.0018s`，P95 为 `0.004109s`、`0.003708s`。
- 同一稳定窗口内 400 个网关请求的 `499/5xx` 为 `0`，平台 API/Worker 错误为 `0`，New API `/api/status` 的 `429` 为 `0`；六个容器运行正常且重启次数为 `0`。
- 结算同步水位为 `1784744504` 且错误为空，开放对账项为 `0`；临时用户、临时数据库、契约渠道、待审核契约申请和活动契约令牌残留均为 `0`。
- 执行 `node scripts/production-smoke.mjs`，`1440x1000` 和 `390x844` 均通过真实生产 UI 检查；四张截图大于 15KB，并人工确认共用商户侧栏、公告、头像跳转、每百万 Token 价格和移动端布局没有遮挡。

### Notes

- `apps/web/scripts/production-smoke.mjs`：新增不记录凭据的真实生产桌面与移动端 Playwright 验收。
- `docs/运营管理与升级验收.md`：追加当前发布、迁移、模型种子、账号轮换、契约、压力和稳定窗口证据。
- `docs/部署运行手册.md`：增加受限凭据文件驱动的生产界面验收命令和边界。
- `progress.md`：追加本轮生产发布、管理员轮换、全量验收、文件清单和回滚点。
- 回滚点：生产代码仍为 `633a78e28c65`；本轮文档与验收脚本提交后可执行 `git revert <本轮提交>` 回退仓库改动。管理员轮换属于运行时凭据变更，不随 Git 回退；如需恢复账号状态，必须通过 New API 原生接口同步更新账号与受限配置，或恢复同一时间点的平台库、New API 库和配置备份。可用备份点为发布前 `20260722T174002Z` 与升级后 `20260722T175546Z`。

## 2026-07-23 - Task: 完成生产升级提交前全量复验

### What was done

- 对本轮生产验收脚本和中文记录执行最终提交前复验，并恢复本地模拟截图因动态时间产生的两处二进制噪声，提交范围保持为生产验收脚本和中文记录。

### Testing

- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./cmd/...`，全部通过。
- 执行 `npm.cmd run test:ui` 与 `npm.cmd run capture`，前端构建、角色导航、共用商户工作区、绝对价格和桌面/移动端截图检查通过。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py'`，19 个部署测试全部通过；执行 `docker compose --env-file .env.example config --quiet` 通过。
- 执行 `npm.cmd audit --audit-level=high`，结果为 `0` 个漏洞；执行 `node --check scripts/production-smoke.mjs` 通过。
- 再次执行真实生产 `node scripts/production-smoke.mjs`，桌面和移动端均通过共用商户侧栏、公告、头像、绝对价格、无横向溢出、无运行时错误和无 `5xx` 检查。
- 执行 `git diff --check`，未发现空白错误。

### Notes

- `progress.md`：追加最终提交前全量验证证据和提交范围说明。
- 回滚点：与上一条生产升级记录相同；本条只追加验证证据，可随本轮提交整体执行 `git revert <本轮提交>`。

## 2026-07-23 - Task: 建立 ModelRoute V2 空库数据基线

### What was done

- 将 8 份旧平台迁移替换为单一 V2 空库基线，不保留旧字段兼容分支。
- 建立单模型渠道、15 分钟检测凭证、商户模型唯一报价、渠道级内部别名、文本价格阶梯和图片价格矩阵。
- 将商户与路由倍率统一为 `0..50000` 基点，健康渠道默认不安排主动探活。
- 将支付结构简化为人民币分和 New API 整数额度，移除美元金额与汇率字段。

### Testing

- 先运行 `go test ./internal/migrations`，测试按预期因仍存在 8 份旧迁移而失败；替换 V2 基线后同一测试通过。
- 使用临时 `postgres:17-alpine` 容器执行完整基线 SQL，`ON_ERROR_STOP=1` 下成功创建 33 张业务表。
- 查询确认 `channel_health.next_probe_at` 允许为空；临时 PostgreSQL 容器已停止并自动清理。

### Notes

- `apps/api/internal/migrations/migrate_test.go`：锁定单一 V2 基线、关键表、倍率边界和已删除字段。
- `apps/api/internal/migrations/sql/000001_v2_baseline.sql`：提供平台 V2 全新空库结构。
- `apps/api/internal/migrations/sql/000001_foundation.sql` 至 `000008_absolute_pricing.sql`：删除旧迁移，避免重置环境继续携带旧业务结构。
- `docs/V2数据基线.md`：记录 V2 数据关系、健康、支付和回滚边界。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：代码基线提交 `480487f`；本阶段提交后可执行 `git revert <本阶段提交>` 回退仓库改动。103 尚未执行 V2 重置，本阶段回退不涉及生产数据恢复。

## 2026-07-23 - Task: 完成 V2 单模型渠道、报价、路由、健康、支付与模型种子后端

### What was done

- 商户渠道改为一个渠道绑定一个标准模型，新增 15 分钟 `hi` 自助检测凭证和管理员独立复检，自动保存厂商原生或 OpenAI 兼容适配器。
- 报价改为“商户 + 模型”唯一版本，同模型多渠道分别生成内部别名；倍率统一为 `0..50000` 基点，文本与图片表达式均支持零倍率。
- API 密钥创建后绑定独立路由策略，智能与固定商户路由区间统一为 `0.00x..5.00x`；公网增加图片生成并移除 Embeddings。
- 健康渠道只由真实调用更新，连续三次临时故障后按 `1/5/15/30/60/120` 分钟冷却，到期只探测一次；硬故障直接停用。
- 充值、余额、账本和运营金额统一为人民币显示口径，订单只保存人民币分和 New API 整数额度；外围分组倍率接口退出。
- 官方模型种子升级为 `2026-07-23-v2`，收录 23 个对话和图片模型，绑定固定 New API 源码提交并校验官方证据域名。

### Testing

- 使用本地 `postgres:17-alpine` 测试容器清空后执行 `go test -p 1 -count=1 ./...`，所有 Go 单元与 PostgreSQL 集成测试通过。
- 设置 `NEW_API_RC21_SOURCE_DIR` 后执行固定源码交集测试，源码 HEAD 与 `bde9b2f44887d34ec54799ae191d50f97914359e` 一致，23 个种子模型校验通过。
- 执行 `go vet ./...`、`go build ./...`、`gofmt -l` 和 `git diff --check`，均通过。
- 结构检查确认 V2 种子为 23 个模型、0 个 Embeddings、3 个图片模型；分组 GET/PUT 回归测试均锁定为 404。

### Notes

- `apps/api/internal/bootstrap/resources.go`：接入 New API 临时渠道检测服务。
- `apps/api/internal/httpapi/absolute_pricing.go`：实现无币种文本阶梯、图片矩阵和零至五倍表达式。
- `apps/api/internal/httpapi/absolute_pricing_test.go`：覆盖文本、缓存、图片和倍率边界。
- `apps/api/internal/httpapi/channel_detector.go`：新增临时渠道 `hi` 检测与自动适配器选择。
- `apps/api/internal/httpapi/channel_detector_test.go`：覆盖适配器回退、临时渠道清理和检测结果。
- `apps/api/internal/httpapi/health_runner.go`：将冷却探测扩展到文本和图片协议。
- `apps/api/internal/httpapi/health_runner_test.go`：锁定图片探测请求和单次恢复规则。
- `apps/api/internal/httpapi/merchant_handlers.go`：接入短期检测凭证、单模型渠道和管理员复检。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖凭证、单模型提交、复检和响应隐藏。
- `apps/api/internal/httpapi/merchant_store.go`：按商户与标准模型保存检测、渠道和唯一报价关系。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖渠道健康列表、商户隔离和单模型持久化。
- `apps/api/internal/httpapi/merchant_types.go`：将商户请求与响应收敛为标准模型和自动适配器字段。
- `apps/api/internal/httpapi/operations_handlers.go`：删除外围分组倍率处理器。
- `apps/api/internal/httpapi/operations_handlers_test.go`：锁定分组接口不再暴露。
- `apps/api/internal/httpapi/overview_store.go`：将运营金额换算为人民币口径。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：覆盖人民币运营汇总与整数结算。
- `apps/api/internal/httpapi/payment_domain.go`：改为人民币分直接换算 New API 额度。
- `apps/api/internal/httpapi/payment_domain_test.go`：覆盖人民币金额解析和额度边界。
- `apps/api/internal/httpapi/payment_handlers.go`：调整支付配置和订单 HTTP 契约。
- `apps/api/internal/httpapi/payment_handlers_test.go`：覆盖人民币订单和无汇率响应。
- `apps/api/internal/httpapi/payment_store.go`：持久化人民币金额、额度和精简支付设置。
- `apps/api/internal/httpapi/payment_store_postgres_test.go`：覆盖真实 PostgreSQL 支付幂等与人民币字段。
- `apps/api/internal/httpapi/payment_store_test.go`：更新支付仓储模拟契约。
- `apps/api/internal/httpapi/payment_types.go`：删除美元和汇率类型字段。
- `apps/api/internal/httpapi/pricing_sync.go`：按同模型多渠道同步文本或图片别名。
- `apps/api/internal/httpapi/pricing_sync_test.go`：覆盖多渠道整体激活、图片表达式和失败补偿。
- `apps/api/internal/httpapi/routing_domain.go`：实现零至五倍候选排序和六档冷却状态机。
- `apps/api/internal/httpapi/routing_domain_test.go`：覆盖倍率、冷却、立即恢复和硬故障。
- `apps/api/internal/httpapi/routing_handlers.go`：调整模型、报价与密钥策略接口。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖唯一报价、密钥策略和协议白名单。
- `apps/api/internal/httpapi/routing_proxy.go`：支持图片生成入口和渠道级健康观察。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖图片代理、首字节切换和内部别名隐藏。
- `apps/api/internal/httpapi/routing_store.go`：重写 V2 单模型报价、候选、冷却任务和人民币查询。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：覆盖真实数据库的唯一报价、多渠道和冷却筛选。
- `apps/api/internal/httpapi/routing_store_test.go`：更新路由仓储模拟与 SQL 契约。
- `apps/api/internal/httpapi/server.go`：注册渠道检测、模型删除和图片生成，移除 Embeddings。
- `apps/api/internal/httpapi/services.go`：增加渠道检测服务依赖。
- `apps/api/internal/httpapi/settlement_store.go`：按渠道级别名维持结算归属并使用人民币展示口径。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：覆盖 V2 别名结算、退款和公开模型名。
- `apps/api/internal/migrations/sql/000001_v2_baseline.sql`：补齐 V2 单模型渠道和报价约束。
- `apps/api/internal/modelseed/apply.go`：移除生命周期比较并支持文本或图片价格幂等导入。
- `apps/api/internal/modelseed/apply_test.go`：覆盖 V2 种子创建、重复和调价追加。
- `apps/api/internal/modelseed/catalog.go`：切换 V2 种子并校验厂商官方证据域名。
- `apps/api/internal/modelseed/catalog_test.go`：锁定 23 模型、三协议、关键模型和价格边界。
- `apps/api/internal/modelseed/data/2026-07-23-v1.json`：删除旧 21 模型、Embeddings 和汇率种子。
- `apps/api/internal/modelseed/data/2026-07-23-v2.json`：新增 23 个对话与图片模型及官方证据。
- `apps/api/internal/modelseed/repository_postgres_test.go`：更新 V2 种子真实数据库幂等数量。
- `apps/api/internal/modelseed/support_rc21.go`：固化 V2 模型与固定 New API 支持交集。
- `apps/api/internal/modelseed/support_rc21_external_test.go`：增加源码 HEAD 校验并核对默认清单。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：前一提交为 `e26e125`；执行 `git revert <本阶段提交>` 可回退后端代码。103 尚未重置，当前阶段回退不涉及生产数据；生产执行 V2 重置后须恢复同一备份点的平台库、New API 库、Redis 和受限配置。

## 2026-07-23 - Task: 完成 V2 三套工作区、密钥路由与价格管理界面

### What was done

- 用户控制台改为账户概览、API 密钥与路由、调用记录、充值中心和账户设置五项侧栏；密钥创建后可在同一行配置智能或固定路由。
- 商户中心保持与唯一管理员完全相同的六项侧栏，渠道提交只选择标准模型，报价按商户与模型唯一并支持 `0.00x..5.00x`。
- 管理后台删除分组倍率入口，模型管理只保留固定厂商、对话/Responses/图片协议、文本阶梯和图片质量尺寸价格矩阵。
- 余额、充值、账本、提现和运营报表统一显示人民币；模型价格只显示数值和计费单位，零倍率只显示“免费”。
- 更新桌面和移动端截图矩阵，新增图片价格弹窗截图并删除旧分组截图。

### Testing

- 执行 `npm.cmd run test:ui`，Vite 生产构建及三套侧栏、单模型渠道、零至五倍报价、密钥路由、人民币支付、结算和人工提现交互全部通过。
- 执行 `npm.cmd run build`，1614 个模块构建成功。
- 执行 `npm.cmd run capture`，桌面和移动端截图全部生成，根页面无横向溢出。
- 人工检查用户控制台、商户报价、文本价格和移动端图片矩阵截图，未见文字遮挡、控件重叠或异常裁切。

### Notes

- `apps/web/scripts/capture.mjs`：更新 V2 页面与弹窗截图场景。
- `apps/web/scripts/ui-smoke.mjs`：覆盖侧栏、单模型、倍率边界、人民币和旧字段退出。
- `apps/web/src/Modals.jsx`：重做密钥路由、渠道检测、唯一报价、充值与提现弹窗。
- `apps/web/src/api.js`：统一人民币额度格式化并接入 V2 接口字段。
- `apps/web/src/views/AdminView.jsx`：删除管理后台分组导航。
- `apps/web/src/views/ConsoleView.jsx`：实现用户控制台五项侧栏和密钥路由同页。
- `apps/web/src/views/MarketView.jsx`：显示无币种文本与图片价格和商户倍率。
- `apps/web/src/views/MerchantView.jsx`：实现单模型渠道、唯一报价和人民币商户数据。
- `apps/web/src/views/PaymentReturnView.jsx`：将充值结果统一为人民币。
- `apps/web/src/views/admin/AdminChannels.jsx`：删除渠道类型与多模型字段展示。
- `apps/web/src/views/admin/AdminGroups.jsx`：删除旧分组倍率页面。
- `apps/web/src/views/admin/AdminMerchants.jsx`：显示单模型渠道审核信息。
- `apps/web/src/views/admin/AdminModels.jsx`：实现模型删除、文本阶梯和图片价格矩阵。
- `apps/web/src/views/admin/AdminPayment.jsx`：删除汇率并维护人民币充值范围。
- `apps/web/src/views/admin/AdminRedemptions.jsx`：将兑换额度显示为人民币。
- `apps/web/artifacts/admin-announcements-desktop.png`：更新管理员公告桌面回归图。
- `apps/web/artifacts/admin-channels-desktop.png`：更新单模型渠道管理桌面回归图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新单模型渠道管理移动回归图。
- `apps/web/artifacts/admin-desktop.png`：更新管理后台侧栏桌面回归图。
- `apps/web/artifacts/admin-mobile.png`：更新管理后台侧栏移动回归图。
- `apps/web/artifacts/admin-groups-desktop.png`：删除旧分组倍率桌面回归图。
- `apps/web/artifacts/admin-groups-mobile.png`：删除旧分组倍率移动回归图。
- `apps/web/artifacts/admin-model-price-desktop.png`：更新文本阶梯价格桌面回归图。
- `apps/web/artifacts/admin-model-price-mobile.png`：更新文本阶梯价格移动回归图。
- `apps/web/artifacts/admin-image-price-desktop.png`：新增图片价格矩阵桌面回归图。
- `apps/web/artifacts/admin-image-price-mobile.png`：新增图片价格矩阵移动回归图。
- `apps/web/artifacts/admin-models-desktop.png`：更新标准模型管理桌面回归图。
- `apps/web/artifacts/admin-models-mobile.png`：更新标准模型管理移动回归图。
- `apps/web/artifacts/admin-redemptions-desktop.png`：更新人民币兑换码桌面回归图。
- `apps/web/artifacts/admin-redemptions-mobile.png`：更新人民币兑换码移动回归图。
- `apps/web/artifacts/admin-settings-desktop.png`：更新平台设置桌面回归图。
- `apps/web/artifacts/admin-settings-mobile.png`：更新平台设置移动回归图。
- `apps/web/artifacts/admin-tasks-desktop.png`：更新系统任务桌面回归图。
- `apps/web/artifacts/admin-usage-desktop.png`：更新人民币用量桌面回归图。
- `apps/web/artifacts/admin-usage-mobile.png`：更新人民币用量移动回归图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：更新人民币提现审核桌面回归图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：更新人民币提现审核移动回归图。
- `apps/web/artifacts/announcement-panel-desktop.png`：更新公告面板桌面回归图。
- `apps/web/artifacts/announcement-panel-mobile.png`：更新公告面板移动回归图。
- `apps/web/artifacts/console-desktop.png`：新增用户五项侧栏桌面回归状态。
- `apps/web/artifacts/console-mobile.png`：新增用户五项侧栏移动回归状态。
- `apps/web/artifacts/market-desktop.png`：更新 V2 模型市场桌面回归图。
- `apps/web/artifacts/market-mobile.png`：更新 V2 模型市场移动回归图。
- `apps/web/artifacts/merchant-desktop.png`：更新商户侧栏桌面回归图。
- `apps/web/artifacts/merchant-mobile.png`：更新商户侧栏移动回归图。
- `apps/web/artifacts/merchant-offers-desktop.png`：更新唯一模型报价桌面回归图。
- `apps/web/artifacts/merchant-offers-mobile.png`：更新唯一模型报价移动回归图。
- `apps/web/artifacts/payment-config-desktop.png`：更新人民币支付配置桌面回归图。
- `apps/web/artifacts/payment-config-mobile.png`：更新人民币支付配置移动回归图。
- `apps/web/artifacts/route-modal-desktop.png`：更新密钥路由桌面回归图。
- `apps/web/artifacts/route-modal-mobile.png`：更新密钥路由移动回归图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：更新人民币提现桌面回归图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：更新人民币提现移动回归图。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：前一提交为 `dbf128b`；执行 `git revert <本阶段提交>` 可回退 V2 前端与截图。该阶段不改生产数据。

## 2026-07-23 - Task: 建立 V2 受控备份重置工具并同步中文运行文档

### What was done

- 扩展生产备份为平台库、New API 库、Redis RDB、New API 文件、SMTP 设置、共享环境和管理员受限文件的完整回滚包。
- 新增 103 专用 V2 数据重置脚本，强制校验确认值、目标 IP、Git 哈希发布目录、共享数据真实路径和 `modelroute` Compose 项目。
- 重置流程固定先做 SHA-256 校验与双数据库临时恢复演练，再清理 ModelRoute 四个数据子目录，原生初始化唯一管理员、恢复 SMTP、导入 V2 种子并执行基础验收。
- 删除运营验收中的分组接口访问，更新全部现行中文文档为单模型、零至五倍、密钥路由、冷却探测、图片计费和人民币口径；旧计划与规范标记为历史文件。

### Testing

- 部署工具测试先稳定捕获旧种子路径、缺少 Redis/受限配置备份和缺少重置脚本；实现后执行 `python -m unittest discover -s deploy/tests -p 'test_*.py'`，22 个测试全部通过。
- 执行 `bash -n deploy/scripts/backup.sh`、`bash -n deploy/scripts/reset-v2-data.sh` 和 `python -m py_compile deploy/scripts/bootstrap-newapi.py`，语法检查通过。
- 执行 `docker compose --env-file .env.example --profile tools config --quiet`，Compose 配置校验通过。
- 中文文档关键口径检查确认 `2026-07-23-v2`、`0.00x..5.00x`、人民币额度、图片入口、渠道检测和受控重置命令均已写入现行文档。
- 只读检查 103 的真实发布结构，确认 ModelRoute 数据位于 `/opt/modelroute-platform/shared/data`、Compose 项目名为 `modelroute`，Sub2API 为独立项目。

### Notes

- `deploy/scripts/backup.sh`：增加 Redis、New API 文件、SMTP 与受限配置备份和完整性检查。
- `deploy/scripts/bootstrap-newapi.py`：通过 New API 原生接口初始化唯一管理员并恢复 SMTP 白名单设置。
- `deploy/scripts/reset-v2-data.sh`：实现 103 专用、显式确认、先备份演练后重置的 V2 发布流程。
- `deploy/scripts/verify-operations.sh`：删除已退出的分组接口只读验收。
- `deploy/tests/test_bootstrap_newapi.py`：覆盖 SMTP 设置备份白名单和未知字段拦截。
- `deploy/tests/test_release_tools.py`：锁定 V2 种子、运营接口、完整备份和重置保护条件。
- `docs/New API版本与许可.md`：说明 New API 原生契约与 ModelRoute 公网协议边界。
- `docs/V2数据基线.md`：补充受控重置脚本、固定路径和回滚边界。
- `docs/superpowers/plans/2026-07-22-ModelRoute完整实施计划.md`：标记旧实施计划已由 V2 取代。
- `docs/superpowers/specs/2026-07-22-API聚合平台设计规范.md`：标记旧多模型和周期探活规范为历史口径。
- `docs/商户与渠道审核接口.md`：重写自助检测、单模型提交、管理员复检和密钥生命周期。
- `docs/官方模型种子.md`：记录 V2 的 23 个模型、价格数字一比一和证据约束。
- `docs/易支付充值.md`：改为纯人民币订单和 `¥1 = 500000` 额度。
- `docs/模型报价路由与健康.md`：记录密钥策略、唯一报价、图片入口和仅冷却探测。
- `docs/绝对价格与分层计费.md`：记录无币种价格、零至五倍和文本/图片表达式。
- `docs/账号与管理员接口.md`：将账号余额说明统一为人民币。
- `docs/运营管理与升级验收.md`：删除分组功能并增加 V2 发布验收要求。
- `docs/部署环境审计.md`：更新外围分组删除后的功能边界。
- `docs/部署运行手册.md`：增加 V2 数据重置完整操作与验收顺序。
- `progress.md`：追加本阶段实现、验证、文件清单和回滚点。
- 回滚点：前一提交为 `ea0318a`；执行 `git revert <本阶段提交>` 可回退部署工具和中文文档。103 仍运行 V1 数据，当前阶段回退不涉及生产恢复；V2 重置后使用脚本输出的完整备份目录回滚。

## 2026-07-23 - Task: 完成 V2 生产发布前最终本地验收

### What was done

- 在三个 V2 功能提交后重新执行全量本地验收，确认最终 Git 状态对应的后端、前端、部署工具和固定 New API 契约一致。
- 清理隔离契约容器与网络，并恢复截图中仅由动态时间产生的两处二进制噪声，工作树保持干净。

### Testing

- 使用真实 PostgreSQL 测试容器执行 `go test -p 1 -count=1 ./...`，全部包通过；随后执行 `go vet ./...` 和 `go build ./...` 通过。
- 执行 `npm.cmd run test:ui`、`npm.cmd run build`、`npm.cmd run capture` 和 `npm.cmd audit --audit-level=high`，UI、构建、截图和依赖审计通过，漏洞数为 `0`。
- 执行 22 个部署 Python 测试、完整 `tools` profile Compose 配置校验和脚本语法检查，全部通过。
- 执行 `deploy/scripts/verify-newapi-contract.sh`，固定 New API `rc.21` 的初始化、账号、令牌、渠道、模型映射、分层表达式、Chat、Responses、原生 Embeddings、流式、缓存日志、退款和无渠道错误契约通过；临时容器与网络残留为 `0`。
- 执行 `git diff --check` 和敏感信息模式扫描，工作树无空白错误且未命中已知凭据。

### Notes

- `progress.md`：追加提交后完整本地验收、隔离契约清理和回滚点。
- 回滚点：代码与工具基线提交为 `7641ea9`；本条只追加验证证据，可随本轮提交执行 `git revert <本阶段提交>`。103 尚未执行 V2 重置。

## 2026-07-23 - Task: 修复 New API 长用户名初始化并支持中断续跑

### What was done

- 修复 New API `/api/setup` 只接受最多 12 字符用户名导致目标管理员邮箱初始化失败的问题；新库先创建受控临时根账号，再通过 New API 原生用户接口改为目标管理员用户名。
- 改名后强制使用目标用户名重新登录并校验根角色，SMTP 设置只通过最终管理员会话恢复。
- 增加 setup 已完成但尚未改名时的恢复路径，初始化脚本可从该中断点继续，不需要再次清理数据库。

### Testing

- 新增长用户名失败用例并先确认旧脚本因 17 字符用户名超过 setup 上限而失败；实现后执行 `python -m unittest deploy.tests.test_bootstrap_newapi -v`，3 个定向测试全部通过。
- 新增已初始化临时管理员的中断恢复用例并先确认旧流程无法继续；实现后该用例通过。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，24 个部署测试全部通过；执行 `python -m compileall -q deploy/scripts/bootstrap-newapi.py` 和 `git diff --check` 通过。
- 使用固定镜像 `calciumion/new-api:v1.0.0-rc.21` 启动隔离 PostgreSQL、Redis 和 New API，真实完成短名 setup、原生根账号改名和目标邮箱重新登录；测试容器、卷及网络随后全部清理。

### Notes

- `deploy/scripts/bootstrap-newapi.py`：增加短用户名初始化、原生根账号改名、最终登录校验和中断恢复。
- `deploy/tests/test_bootstrap_newapi.py`：覆盖长目标用户名和 setup 后中断两条回归路径。
- `docs/部署运行手册.md`：记录 New API 首次初始化的用户名限制、改名顺序和续跑方式。
- `progress.md`：追加本轮实现、测试证据、文件清单和回滚点。
- 回滚点：前一提交为 `9575ae8`；执行 `git revert <本阶段提交>` 可回退本轮初始化脚本。若 103 已完成根账号改名，代码回退不会自动改回数据库用户名；数据级回滚使用 `/opt/modelroute-platform/shared/backups/20260723T025901Z`。

## 2026-07-23 - Task: 校正 V2 生产验收工具的商户、路由和管理员契约

### What was done

- 将六组需管理员登录的生产验收脚本统一到唯一管理员 `3180615598@qq.com`，删除旧登录名默认值。
- 将路由验收的默认区间、固定路由边界和智能路由保存值统一为 V2 的 `0..50000` 基点。
- 将商户验收改为动态选择当前有效文本模型，先确认伪造上游的自助 `hi` 检测被拒绝，再使用绑定临时用户、单一模型、地址和密钥指纹的短期检测凭证完成申请、列表、驳回和敏感字段检查。
- 商户验收结束或中断时按临时用户编号清理平台检测凭证、申请、审核、审计和 New API 临时账号，避免在重置后的开发库留下契约测试业务记录。

### Testing

- 新增三组部署失败测试，并先确认旧脚本分别缺少 V2 单模型检测凭证、仍使用旧管理员登录名、仍断言 `1..10000` 路由范围；修改后定向测试全部通过。
- 执行六个生产验收脚本的 `bash -n`，语法检查全部通过。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，26 个部署测试全部通过。
- 执行 `git diff --check`，未发现空白错误。

### Notes

- `deploy/scripts/verify-accounts.sh`：更新唯一管理员默认登录名。
- `deploy/scripts/verify-merchants.sh`：改为 V2 单模型检测凭证、申请契约和可回收测试数据。
- `deploy/scripts/verify-routing.sh`：改为 `0..50000` 基点路由范围并更新管理员登录名。
- `deploy/scripts/verify-payments.sh`：更新唯一管理员默认登录名。
- `deploy/scripts/verify-settlement.sh`：更新唯一管理员默认登录名。
- `deploy/scripts/verify-operations.sh`：更新唯一管理员默认登录名。
- `deploy/tests/test_release_tools.py`：锁定 V2 商户、路由和唯一管理员生产验收契约。
- `docs/部署运行手册.md`：说明 V2 生产验收的临时凭证、清理边界和凭据来源。
- `progress.md`：追加本轮实现、测试证据、文件清单和回滚点。
- 回滚点：前一提交为 `5468bc9`；执行 `git revert <本阶段提交>` 可回退生产验收工具。本阶段不改平台业务结构，生产验证产生的临时记录由脚本按用户编号清理。

## 2026-07-23 - Task: 完成 V2 生产重置验收并校正生产界面检查

### What was done

- 在 103 完成 ModelRoute V2 开发数据全量重置，只重建平台库、New API 库和 Redis，并重新初始化唯一管理员、SMTP 设置和 23 个模型的 V2 种子。
- 完成账号、商户、路由、支付、结算和运营生产契约，清理全部验收临时用户、申请、检测凭证和 New API 临时渠道。
- 校正真实生产 Playwright 对模型管理页面的旧文案断言，使验收脚本匹配“标准模型与基础价格”和“发布模型价格”的现行界面。
- 更新部署审计为 V2 当前结构、备份、支付空配置与 SMTP 实发缺口，不再把 V1 的迁移和发布哈希描述为当前状态。

### Testing

- 重置前完整备份 `/opt/modelroute-platform/shared/backups/20260723T025901Z` 和重置后 V2 完整备份 `/opt/modelroute-platform/shared/backups/20260723T032444Z` 均通过 SHA-256；恢复演练确认平台库 `34` 张表、New API 库 `31` 张表，未覆盖生产库。
- 生产基础、账号、商户、路由、支付、结算和运营契约全部通过；测试申请、检测凭证、临时用户和 New API 临时渠道均为 `0`。
- 执行只读压力测试 `200` 个请求、并发 `10`，成功 `200`，平均耗时 `0.0017s`、P95 `0.004087s`；验收窗口内网关 `5xx`、New API `429`、API 与 Worker fatal 均为 `0`。
- `/v1/chat/completions`、`/v1/responses` 和 `/v1/images/generations` 无密钥返回 `401`，`/v1/embeddings` 返回 `404`；三个现有 HTTPS 域名均返回 `200`。
- 修复前真实生产 Playwright 在等待旧文案时稳定失败；修复后桌面 `1440x1000` 和移动 `390x844` 两个视口均通过，截图人工检查非空且无文字重叠。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，27 个部署测试全部通过；执行 `git diff --check` 通过。
- 只读复核生产 `current`、`/api/version`、容器状态和备份目录，确认 V2 首次验收版本为 `8257e228f367`，六个服务运行且五项健康检查正常。

### Notes

- `apps/web/scripts/production-smoke.mjs`：将模型管理生产验收断言更新为 V2 当前页面文案。
- `deploy/tests/test_release_tools.py`：增加生产 UI 验收脚本不得回退到旧模型价格文案的回归测试。
- `docs/部署环境审计.md`：以 V2 生产数据库、种子、协议、备份和配置现状替换旧 V1 当前状态。
- `progress.md`：追加本轮 V2 生产重置、完整验收、文件清单和回滚点。
- 回滚点：代码前一提交为 `8257e228f367`；执行 `git revert <本阶段提交>` 可回退生产界面验收脚本和文档。数据级回滚使用重置前完整备份 `/opt/modelroute-platform/shared/backups/20260723T025901Z`，V2 当前数据恢复使用 `/opt/modelroute-platform/shared/backups/20260723T032444Z`。
