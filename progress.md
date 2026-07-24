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

## 2026-07-23 - Task: 修复 hi 检测通过后渠道提交固定返回状态冲突

### What was done

- 根据生产截图、网关 409 记录和平台库检测记录定位渠道提交失败：检测凭证保存使用 HMAC-SHA256，提交领取使用 SHA-256，同一凭证的摘要必然不匹配。
- 将检测凭证保存与提交领取收敛到同一个 SHA-256 摘要函数；渠道密钥继续使用独立 HMAC-SHA256 指纹，不改变密钥保护边界。
- 增加检测响应随机凭证与数据库保存摘要必须一致的回归测试，防止两个阶段再次使用不同算法。

### Testing

- 生产只读核对确认管理员官方商户状态为 `active`，三条 `hi` 检测记录均在失败时有效且未消费，渠道审核记录为 `0`；网关记录四次 `POST /api/merchant/channels` 返回 `409`。
- 新回归测试在修复前稳定失败，输出同一凭证的保存摘要与提交期望摘要不同；修复后该测试与渠道申请事务测试均通过。
- 执行 `go test -p 1 -count=1 ./...`，全部 Go 包通过；执行 `go vet ./...` 和 `go build ./...` 通过。
- 执行 `git diff --check`，未发现空白错误。

### Notes

- `apps/api/internal/httpapi/merchant_handlers.go`：检测记录改用与提交事务一致的凭证摘要函数。
- `apps/api/internal/httpapi/merchant_store.go`：集中实现检测凭证 SHA-256 摘要并供领取逻辑复用。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：增加随机检测凭证保存摘要一致性回归测试。
- `docs/商户与渠道审核接口.md`：说明检测凭证摘要与渠道密钥指纹的用途边界。
- `progress.md`：追加生产故障证据、修复验证、文件清单和回滚点。
- 回滚点：前一提交为 `b0a72023c84d`；执行 `git revert <本阶段提交>` 可回退本轮渠道凭证摘要修复。本轮不改数据库结构，修复前生成的检测凭证需重新发送 `hi` 获取。

## 2026-07-23 - Task: 多模型渠道与运营界面升级

### What was done

- 增加多模型渠道同步、规范化匹配、审核模型绑定和正式渠道模型绑定；服务端在提交审核时重新同步上游模型，审批后创建包含全部匹配模型的 New API 渠道。
- 增加管理员逐模型 `hi` 检查、商户渠道再次同步、渠道启用/下架/归档和管理员渠道关键词/商户筛选接口。
- 路由和报价查询兼容多模型绑定；新增渠道调用日志入口；用户模型列表继续隐藏内部报价别名。
- 移除 MiniMax 模型种子、厂商白名单、厂商图标和前端选项；移除密钥查看、市场用于路由、管理员刷新余额和正式渠道行内测试按钮。
- 商户渠道弹窗改为三类上游协议和一键同步模型；管理员渠道页增加逐模型检查、搜索、商户筛选和状态操作。

### Testing

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `go build ./...` 通过。
- `npm.cmd run build` 通过。
- `npm.cmd run test:ui` 通过，桌面和移动端三套工作区冒烟流程完成。
- `python -m unittest discover -s deploy/tests -p 'test_*.py' -v` 通过，27 个部署与发布脚本测试通过。
- 多模型匹配、MiniMax 类型拒绝、上游模型列表解析和 `/v1/models` 代理测试通过。

### Notes

- `apps/api/internal/migrations/sql/000002_multimodel_channels.sql`：新增多模型渠道字段、审核绑定表、正式渠道绑定表和 MiniMax 历史归档。
- `apps/api/internal/httpapi/channel_models.go`、`channel_models_handlers.go`：实现模型匹配、同步、逐模型检查和渠道管理接口。
- `apps/api/internal/httpapi/merchant_store.go`、`merchant_types.go`、`merchant_handlers.go`：保存审核/正式模型绑定并支持多模型审批。
- `apps/api/internal/httpapi/routing_store.go`：报价创建和路由候选查询支持多模型渠道绑定。
- `apps/web/src/Modals.jsx`、`views/MerchantView.jsx`、`views/admin/AdminChannels.jsx`：更新渠道同步、审核、下架删除和筛选界面。
- `apps/web/src/views/ConsoleView.jsx`、`MarketView.jsx`、`AdminView.jsx`、`AdminUsage.jsx`：移除旧操作入口并接入渠道调用日志筛选。
- `apps/web/scripts/ui-smoke.mjs`：同步渠道调用日志新文案和已移除管理员按钮的回归断言。
- `deploy/scripts/reset-v2-data.sh`：开发库重置验收纳入多模型渠道迁移。
- `docs/多模型渠道运营升级.md`：记录业务流程、接口和数据边界。
- 回滚点：本阶段提交前的基线为 `918d1bb`；执行 `git revert <本阶段提交>` 可回退本轮代码、前端和迁移改动。开发库回滚需同时恢复本阶段迁移前的平台库与 New API 备份。

## 2026-07-23 - Task: 将 103 开发重置改为直接覆盖

### What was done

- 按最新部署决定删除 V2 重置流程中的新备份、校验和恢复演练，保留 103 地址、Git 发布目录、共享数据路径和 `modelroute` Compose 项目边界校验。
- 重置脚本改为先完成配置检查和镜像构建，再直接清空四个 ModelRoute 数据目录；重建后校验两份迁移、20 个有效模型和 MiniMax 为零。
- 多模型渠道、官方模型数量、厂商数量和部署验收文档统一到当前实现，界面冒烟输出不再使用“单模型渠道”旧文案。

### Testing

- 新增直接覆盖契约后先运行定向测试，确认旧脚本因仍调用备份流程而失败；修改后定向测试通过。
- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./...`，全部通过。
- 执行 `npm.cmd run build` 和 `npm.cmd run test:ui`，前端构建及三套工作区桌面/移动端烟熏通过。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，27 个部署测试全部通过；Compose 配置校验、Shell 语法检查和 `git diff --check` 通过。

### Notes

- `deploy/scripts/reset-v2-data.sh`：改为 103 定向直接覆盖并增加迁移、模型数和 MiniMax 清理验收。
- `deploy/tests/test_release_tools.py`：锁定重置流程不得调用备份或恢复脚本。
- `apps/web/scripts/ui-smoke.mjs`：将渠道冒烟结果文案更新为多模型渠道。
- `apps/web/src/views/admin/AdminMerchants.jsx`：移除商户审核页的单模型旧描述。
- `docs/V2数据基线.md`：记录多模型关系和直接重置边界。
- `docs/商户与渠道审核接口.md`：更新多模型同步、逐模型检查和渠道管理接口。
- `docs/官方模型种子.md`：更新为 20 个模型、6 家厂商并移除 MiniMax。
- `docs/模型报价路由与健康.md`：更新多模型渠道报价和路由关系。
- `docs/运营管理与升级验收.md`：更新模型目录验收数量。
- `docs/部署运行手册.md`：更新直接覆盖命令、两份迁移和 20 个模型验收步骤。
- `progress.md`：追加本轮直接覆盖策略、验证证据和文件清单。
- 回滚点：本阶段前的代码基线为 `382c3ad`；执行 `git revert <本阶段提交>` 可回退本轮脚本、文档和文案。103 尚未执行本轮直接重置。

## 2026-07-23 - Task: 修复带版本后缀的渠道模型同步

### What was done

- 生产专项验收复现商户填写 `.../v1` 后模型同步返回 `502`，确认固定 New API 会再次拼接 `/v1/models`，实际请求形成重复版本路径。
- 平台保留商户提交的基础地址，仅在调用 New API 时按上游类型去掉末尾 `/v1` 或 `/v1beta`；模型同步、管理员 `hi` 检查和正式渠道创建统一使用该规则。
- 保持 New API `v1.0.0-rc.21` 源码和容器镜像不变。

### Testing

- 先新增三类上游版本后缀测试并确认因规范化函数不存在而编译失败；实现后该测试与渠道适配器回退测试通过。
- 在 103 使用公开只读模型目录验证假设：带 `/v1` 时稳定返回 `502`，去掉重复后缀后经 New API 成功同步 342 条模型。
- 执行 `go test -count=1 ./...`、`go vet ./...` 和 `go build ./...`，全部通过。
- 执行 `python -m unittest discover -s deploy/tests -p 'test_*.py' -v`，27 个部署测试全部通过；`git diff --check` 通过。

### Notes

- `apps/api/internal/httpapi/channel_models.go`：新增按渠道类型转换 New API 基础地址的规则。
- `apps/api/internal/httpapi/channel_models_handlers.go`：模型同步调用使用 New API 专用基础地址。
- `apps/api/internal/httpapi/channel_detector.go`：临时 `hi` 检测统一使用版本后缀转换。
- `apps/api/internal/httpapi/merchant_handlers.go`：正式渠道创建统一使用版本后缀转换。
- `apps/api/internal/httpapi/channel_models_test.go`：覆盖 OpenAI 兼容、Anthropic 和 Gemini 的版本后缀边界。
- `docs/商户与渠道审核接口.md`：记录商户地址保存值与 New API 调用值的边界。
- `docs/多模型渠道运营升级.md`：补充三类上游版本后缀处理规则。
- `progress.md`：追加生产复现、根因、修复和测试证据。
- 回滚点：本阶段前的发布基线为 `138eee3`；执行 `git revert <本阶段提交>` 可回退本轮修复。103 当前仍运行 `138eee35b89e`，修复提交发布后再更新生产落点。

## 2026-07-23 - Task: 直接覆盖发布多模型渠道版本到 103

### What was done

- 将多模型渠道提交 `382c3ad`、直接覆盖提交 `138eee3` 和基础地址修复提交 `60f3100` 推送到 `origin/codex/full-platform`。
- 在 103 只停止 `modelroute` Compose 项目，直接清空其 PostgreSQL、Redis、New API 和日志数据目录，重建唯一管理员、官方商户、两份迁移和 20 个模型；未创建新备份或执行恢复演练。
- 将生产 `current` 原子切换到 `60f310080980`，101、102 对应的 Sub2API、SDK 服务、容器、端口和域名保持运行。

### Testing

- 本地完整执行 Go 测试、静态检查和构建，前端构建与 UI 烟熏、27 个部署测试、Compose 配置校验和 `git diff --check` 全部通过。
- 生产基础、账号、商户、路由、支付、结算和运营契约全部通过；两份迁移存在，模型数为 `20`，MiniMax 为 `0`，唯一管理员与唯一官方商户均为 `1`。
- 使用带 `/v1` 的公开只读上游经 New API 成功同步 342 条模型；渠道日志接口正常，临时密钥 `/v1/models` 返回 OpenAI 兼容列表且不包含内部别名。
- 生产 Playwright 在 `1440x1000` 与 `390x844` 通过，公告、头像跳转、共享商户侧栏和模型价格弹窗无运行时错误或根级横向溢出，四张截图均人工检查非空且无文字重叠。
- 最终只读压力测试请求 `200`、并发 `10`、成功 `200`、平均 `0.0019s`、P95 `0.004332s`；独立稳定窗口网关 `5xx=0`、New API `429=0`、平台 fatal/panic `0`。
- 平台申请、审核、渠道、报价和支付订单均为 `0`；New API 仅有唯一管理员、正式渠道为 `0`、活动测试密钥为 `0`。三个 HTTPS 域名均返回 `200`，ModelRoute 数据服务未发布宿主机端口。

### Notes

- `docs/部署环境审计.md`：更新当前提交、两份迁移、20 个模型、直接覆盖边界和生产验收证据。
- `progress.md`：追加 103 直接覆盖、修复发布、数据清理、隔离和验收记录。
- 生产发布目录：`/opt/modelroute-platform/releases/60f310080980`；代码回滚可把 `current` 切回 `/opt/modelroute-platform/releases/138eee35b89e` 后执行部署脚本。旧开发数据已直接覆盖，不存在本轮数据回滚点。

## 2026-07-23 - Task: 精简后台并统一用户、渠道与数据标识

### What was done

- 删除操作记录、系统任务、密钥查看、渠道余额刷新和正式渠道行内测试的页面、接口、存储及验收逻辑；模型调用、消费、退款、渠道调用日志和结算账本继续保留。
- 用户统一返回普通用户、商户、管理员三种账号类型；商户权限撤销与恢复并入用户管理，支持按账号类型筛选，不再显示独立商户权限列表。
- 管理端渠道审核记录与已审核渠道合并为同一生命周期表，统一显示审核中、已审核、不通过和已下架；商户和管理员均可下架或删除有权限的渠道。
- 商户模型设置支持按厂商再选型号、发布零至五倍价格以及取消当前价格；取消后对应模型退出市场和新路由候选。
- 控制台删除无行为按钮，账户设置改为账号资料与有效工作区；模型市场删除接入文档入口；主要数据表统一以数字 ID 作为首列。
- 前端不再展示内部核心名称、内部商户资料编号和内部渠道编号；运营总览保留全平台成交数据，渠道调用日志支持按商户用户 ID 与平台渠道 ID 筛选。
- 修复渠道模型同步弹窗因缺少状态组件导入而跳回页面的问题；修正截图脚本的旧价格版本断言和商户验收脚本对已删除操作记录表的残留清理语句。

### Testing

- `go test -count=1 ./...` 通过。
- `go vet ./...` 通过。
- `go build ./...` 通过。
- `npm.cmd run test:ui` 通过，覆盖统一渠道表、渠道同步不跳页、取消价格、账号类型、人民币支付和已删除入口。
- `npm.cmd run capture` 通过，桌面与移动端截图全部生成且根页面无横向溢出。
- `python -m unittest discover -s deploy/tests -p "test_*.py"` 通过，27 项部署测试成功。
- `docker compose --env-file .env.example --profile tools -f compose.yaml config` 通过。
- `git diff --check` 通过；静态扫描确认前端无操作记录、系统任务、刷新余额、查看密钥、分组倍率和接入文档入口，仓库未写入上游或支付密钥正文。
- 对提供的上游地址执行只读 `/v1/models` 验证，返回 OpenAI 兼容数据结构、20 个模型且无重复；密钥未写入仓库、文档或测试输出。

### Notes

- `apps/api/internal/bootstrap/resources.go`：移除操作记录存储初始化。
- `apps/api/internal/bootstrap/resources_test.go`：更新资源初始化断言。
- `apps/api/internal/httpapi/absolute_pricing.go`：统一价格记录的外部 ID 字段。
- `apps/api/internal/httpapi/account_handlers.go`：删除密钥查看接口和操作记录写入。
- `apps/api/internal/httpapi/account_test.go`：更新密钥与用户管理接口回归。
- `apps/api/internal/httpapi/announcement_handlers.go`：移除公告操作记录写入。
- `apps/api/internal/httpapi/announcement_handlers_test.go`：更新公告处理测试依赖。
- `apps/api/internal/httpapi/audit_store.go`：删除独立操作记录存储。
- `apps/api/internal/httpapi/audit_store_test.go`：删除独立操作记录存储测试。
- `apps/api/internal/httpapi/auth_handlers.go`：增加账号类型和用户列表商户状态映射。
- `apps/api/internal/httpapi/auth_test.go`：覆盖账号类型与筛选结果。
- `apps/api/internal/httpapi/channel_models_handlers.go`：清理渠道模型响应中的内部字段。
- `apps/api/internal/httpapi/health_runner.go`：统一健康任务使用的渠道标识。
- `apps/api/internal/httpapi/merchant_handlers.go`：合并商户权限入口并清理内部名称与操作记录写入。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖用户维度商户权限和渠道生命周期。
- `apps/api/internal/httpapi/merchant_store.go`：按用户筛选渠道并统一外部渠道 ID。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖渠道筛选与映射。
- `apps/api/internal/httpapi/merchant_types.go`：隐藏内部商户和渠道编号，精简仓储接口。
- `apps/api/internal/httpapi/operations_handlers.go`：删除系统任务、渠道余额和行内测试处理器。
- `apps/api/internal/httpapi/operations_handlers_test.go`：保留运营、渠道日志和兑换码测试并删除旧入口测试。
- `apps/api/internal/httpapi/payment_domain.go`：统一人民币额度字段。
- `apps/api/internal/httpapi/payment_handlers.go`：清理支付操作记录依赖。
- `apps/api/internal/httpapi/payment_handlers_test.go`：更新支付服务测试装配。
- `apps/api/internal/httpapi/pricing_sync.go`：统一价格同步渠道标识。
- `apps/api/internal/httpapi/routing_handlers.go`：增加取消价格并隐藏内部路由字段。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖取消价格和密钥路由响应。
- `apps/api/internal/httpapi/routing_proxy.go`：统一代理请求中的渠道 ID。
- `apps/api/internal/httpapi/routing_store.go`：增加价格停用和平台渠道映射。
- `apps/api/internal/httpapi/server.go`：注销旧接口并注册用户商户状态与取消价格接口。
- `apps/api/internal/httpapi/server_test.go`：确认已删除路由不再注册。
- `apps/api/internal/httpapi/services.go`：删除操作记录服务契约。
- `apps/api/internal/httpapi/settlement_handlers.go`：清理结算操作记录写入。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：更新结算处理器测试装配。
- `apps/api/internal/httpapi/settlement_reconciliation.go`：对账响应改用平台渠道 ID。
- `apps/api/internal/httpapi/settlement_sync.go`：结算同步统一渠道标识。
- `apps/api/internal/httpapi/settlement_types.go`：隐藏内部渠道编号。
- `apps/api/internal/httpapi/usage_handlers.go`：增加按商户用户与平台渠道筛选的渠道日志。
- `apps/api/internal/migrations/sql/000001_v2_baseline.sql`：从开发基线删除操作记录表。
- `apps/web/artifacts/admin-announcements-desktop.png`：更新桌面公告管理截图。
- `apps/web/artifacts/admin-announcements-mobile.png`：更新移动端公告管理截图。
- `apps/web/artifacts/admin-channels-desktop.png`：更新桌面统一渠道管理截图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新移动端统一渠道管理截图。
- `apps/web/artifacts/admin-desktop.png`：更新桌面运营总览截图。
- `apps/web/artifacts/admin-image-price-desktop.png`：更新桌面图片价格截图。
- `apps/web/artifacts/admin-mobile.png`：更新移动端运营总览截图。
- `apps/web/artifacts/admin-model-price-desktop.png`：更新桌面文本价格截图。
- `apps/web/artifacts/admin-model-price-mobile.png`：更新移动端文本价格截图。
- `apps/web/artifacts/admin-models-desktop.png`：更新桌面模型管理截图。
- `apps/web/artifacts/admin-models-mobile.png`：更新移动端模型管理截图。
- `apps/web/artifacts/admin-redemptions-desktop.png`：更新桌面兑换码截图。
- `apps/web/artifacts/admin-redemptions-mobile.png`：更新移动端兑换码截图。
- `apps/web/artifacts/admin-settings-desktop.png`：更新桌面平台设置截图。
- `apps/web/artifacts/admin-settings-mobile.png`：更新移动端平台设置截图。
- `apps/web/artifacts/admin-usage-desktop.png`：更新桌面渠道调用日志截图。
- `apps/web/artifacts/admin-usage-mobile.png`：更新移动端渠道调用日志截图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：更新桌面提现审核截图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：更新移动端提现审核截图。
- `apps/web/artifacts/announcement-panel-desktop.png`：更新桌面公告面板截图。
- `apps/web/artifacts/announcement-panel-mobile.png`：更新移动端公告面板截图。
- `apps/web/artifacts/console-desktop.png`：更新桌面控制台截图。
- `apps/web/artifacts/console-mobile.png`：更新移动端控制台截图。
- `apps/web/artifacts/market-desktop.png`：更新桌面模型市场截图。
- `apps/web/artifacts/market-mobile.png`：更新移动端模型市场截图。
- `apps/web/artifacts/merchant-desktop.png`：更新桌面商户中心截图。
- `apps/web/artifacts/merchant-mobile.png`：更新移动端商户中心截图。
- `apps/web/artifacts/merchant-offers-desktop.png`：更新桌面商户价格截图。
- `apps/web/artifacts/merchant-offers-mobile.png`：更新移动端商户价格截图。
- `apps/web/artifacts/payment-config-desktop.png`：更新桌面支付配置截图。
- `apps/web/artifacts/payment-config-mobile.png`：更新移动端支付配置截图。
- `apps/web/artifacts/route-modal-desktop.png`：更新桌面密钥路由弹窗截图。
- `apps/web/artifacts/route-modal-mobile.png`：更新移动端密钥路由弹窗截图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：更新桌面提现弹窗截图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：更新移动端提现弹窗截图。
- `apps/web/scripts/capture.mjs`：更新截图数据、旧入口和价格版本断言。
- `apps/web/scripts/ui-smoke.mjs`：覆盖账号类型、统一渠道、同步不跳页、取消价格和删除入口。
- `apps/web/src/Modals.jsx`：修复渠道同步弹窗并将价格选择拆为厂商与型号。
- `apps/web/src/styles.css`：补充账户设置和统一表格布局样式。
- `apps/web/src/views/AdminView.jsx`：删除操作记录和系统任务导航，调整渠道日志位置。
- `apps/web/src/views/ConsoleView.jsx`：重做侧栏控制台、密钥路由和账户设置。
- `apps/web/src/views/MarketView.jsx`：删除接入文档入口并增加 ID 首列。
- `apps/web/src/views/MerchantView.jsx`：统一渠道生命周期、取消价格和数据 ID 展示。
- `apps/web/src/views/UsagePanels.jsx`：保留用户调用消费日志并增加日志 ID。
- `apps/web/src/views/admin/AdminAnnouncements.jsx`：统一公告 ID 展示。
- `apps/web/src/views/admin/AdminAudit.jsx`：删除操作记录页面。
- `apps/web/src/views/admin/AdminChannels.jsx`：合并审核与正式渠道表并增加状态操作和筛选。
- `apps/web/src/views/admin/AdminMerchants.jsx`：只保留商户申请审核，移除独立权限表。
- `apps/web/src/views/admin/AdminModels.jsx`：统一模型和价格版本 ID 展示。
- `apps/web/src/views/admin/AdminPayment.jsx`：统一支付订单 ID 与人民币额度展示。
- `apps/web/src/views/admin/AdminRedemptions.jsx`：统一兑换码 ID 展示。
- `apps/web/src/views/admin/AdminSettings.jsx`：精简平台设置说明。
- `apps/web/src/views/admin/AdminTasks.jsx`：删除系统任务页面。
- `apps/web/src/views/admin/AdminUsage.jsx`：改为渠道调用日志并增加商户和渠道筛选。
- `apps/web/src/views/admin/AdminUsers.jsx`：增加用户 ID、账号类型筛选和商户权限操作。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：统一提现与对账渠道 ID 展示。
- `deploy/scripts/verify-accounts.sh`：移除密钥查看和操作记录验收。
- `deploy/scripts/verify-merchants.sh`：更新商户渠道验收并删除已移除表的清理语句。
- `deploy/scripts/verify-operations.sh`：改用运营总览与渠道日志并删除旧操作入口。
- `deploy/scripts/verify-routing.sh`：从密钥列表读取新密钥并更新路由断言。
- `deploy/tests/test_release_tools.py`：锁定旧接口、旧表和旧验收动作不得回归。
- `docs/公告管理.md`：更新公告状态留存与已读规则。
- `docs/商户与渠道审核接口.md`：更新用户维度商户权限和渠道记录边界。
- `docs/多模型渠道运营升级.md`：记录统一渠道生命周期与平台渠道 ID。
- `docs/账号与管理员接口.md`：记录账号类型、密钥复制和已删除接口。
- `docs/运营管理与升级验收.md`：记录精简后的运营总览与渠道日志范围。
- `progress.md`：追加本轮改动、测试、文件清单和回滚点。
- 回滚点：本轮前的本地基线为 `08eb7fedb3e0`；提交后执行 `git revert <本阶段提交>` 可回退本轮代码、文档与截图。当前改动未推送、未部署到 103，服务器运行状态未改变。

## 2026-07-23 - Task: 启动本地 UI 交互预览并修复多模型标签显示

### What was done

- 在 Git 忽略目录启动本地交互预览服务，使用真实 React 页面和内存演示接口，默认进入管理员兼商户身份，便于直接检查四套工作区和主要弹窗。
- 修复渠道表多模型标签无间距导致模型名连成一串的问题，增加独立标签、稳定宽度和长名称换行样式。

### Testing

- `http://127.0.0.1:57723/` 返回 `200`，`/api/me` 返回管理员账号类型。
- Playwright 实浏览器验证管理后台、渠道管理页、3 条生命周期数据和截图，运行时错误为 `0`，页面宽度 `1440`、滚动宽度 `1440`。
- `npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面和移动端截图重新生成。

### Notes

- `apps/web/src/styles.css`：增加多模型标签的间距、边框和换行样式。
- `apps/web/artifacts/admin-channels-desktop.png`：更新多模型渠道标签的桌面截图。
- `apps/web/artifacts/admin-channels-mobile.png`：更新多模型渠道标签的移动端截图。
- `data/ui-preview/vite.config.mjs`：Git 忽略的本地演示接口和内存数据，不进入正式构建与服务器发布。
- `data/ui-preview/verify.mjs`：Git 忽略的本地浏览器渲染检查脚本。
- `data/ui-preview/current.png`：Git 忽略的本地渠道管理截图。
- 回滚点：撤销本轮未提交的 `apps/web/src/styles.css` 与截图改动即可恢复上一提交 `acaa5edd9c32`；本地预览服务可通过结束监听 `57723` 的 Node 进程关闭，未连接和修改服务器。

## 2026-07-23 - Task: 报价可用渠道数与市场可见性

### What was done

- 市场报价只统计启用且健康状态为“健康/降级”的可用渠道，并在可用渠道数为零时直接隐藏报价。
- 商户报价接口增加 `channel_count`，商户中心和模型市场统一显示“可用渠道”数量；前端增加零数量保护，避免异常数据再次展示。
- 本地演示接口同步模拟渠道下架后的市场空列表与恢复后的报价数量，未连接生产服务。

### Testing

- PostgreSQL 集成回归：冷却渠道不返回市场报价，恢复为降级后返回报价且 `channel_count=1`；商户报价数量同步为 `1`。
- `go test -count=1 ./...`、`go vet ./...`、`go build ./...` 通过。
- `npm.cmd run test:ui`、`npm.cmd run capture` 通过；本地 `http://127.0.0.1:57723/` 的浏览器检查无运行时错误、无横向溢出。
- 本地预览接口实测：下架渠道后 `/api/offers` 返回空数组，重新启用后恢复报价；部署测试 27 项通过，Compose 配置校验通过。

### Notes

- `apps/api/internal/httpapi/routing_store.go`：增加商户可用渠道数字段，市场和商户查询按健康可路由状态统计。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：增加冷却隐藏、降级恢复和商户数量回归。
- `apps/web/src/views/MarketView.jsx`：显示可用渠道数量并过滤零数量报价。
- `apps/web/src/views/MerchantView.jsx`：报价表改为显示可用渠道数量。
- `docs/模型报价路由与健康.md`：记录 `channel_count` 与市场隐藏规则。
- `data/ui-preview/vite.config.mjs`：Git 忽略的本地演示接口加入渠道下架后的市场过滤。
- `apps/web/artifacts/market-*.png`、`apps/web/artifacts/merchant-offers-*.png`：更新本地界面截图；同次截图命令生成的其他界面截图保持现状。
- 回滚点：本轮提交未推送、未上传、未部署；使用 `git revert <本轮提交哈希>` 可撤销本轮代码、文档和截图，停止监听 `57723` 的 Node 进程可关闭本地预览。

## 2026-07-23 - Task: 本地默认上游模型获取与支付配置预览

### What was done

- 渠道弹窗打开时读取本地预览默认上游地址和协议，已配置默认上游时无需重复填写密钥即可获取模型。
- 将按钮统一为“一键获取模型”；同步结果留在原弹窗内，不再触发页面跳转或关闭，手动补充提交密钥后仍保留同步结果。
- 本地预览服务端使用受限进程环境请求真实上游模型列表，并按本地模型目录返回已收录与平台未收录结果；支付设置只显示网关、商户号掩码和已配置状态。
- 增加真实浏览器回归，覆盖默认地址填充、真实模型同步、弹窗不跳转和无横向溢出。

### Testing

- 实测 `GET /api/merchant/channel-defaults` 返回已配置的默认地址和协议，未返回密钥。
- 实测 `POST /api/merchant/channel-models/sync` 通过真实上游返回 20 个模型，其中 2 个匹配本地目录，其余标记为平台未收录。
- 实测管理支付设置返回网关、商户号掩码和 `key_configured=true`，响应不含明文密钥。
- `node data/ui-preview/verify.mjs` 通过，浏览器运行时错误为 0，页面宽度与滚动宽度均为 1440。
- `npm.cmd run test:ui` 通过；`npm.cmd run capture` 通过；`node --check data/ui-preview/vite.config.mjs`、`node --check data/ui-preview/verify.mjs` 和 `git diff --check` 通过。

### Notes

- `apps/web/src/Modals.jsx`：读取默认上游配置，修复获取模型按钮文案、校验和同步后弹窗状态。
- `apps/web/scripts/ui-smoke.mjs`：增加默认上游、无重复输入和同步不跳转回归。
- `docs/部署运行手册.md`：补充仅限本地预览的环境变量和敏感字段边界。
- `apps/web/artifacts/admin-settings-desktop.png`：更新支付设置桌面截图。
- `apps/web/artifacts/admin-settings-mobile.png`：更新支付设置移动端截图。
- `data/ui-preview/vite.config.mjs`：本地忽略目录中的真实上游同步和支付配置预览逻辑，不进入正式构建。
- `data/ui-preview/verify.mjs`：本地忽略目录中的真实浏览器回归脚本，不进入 Git。
- 凭据只存在于当前本地预览进程环境，未写入 Git、文档、进度日志、截图或接口响应；本轮未推送、未上传、未部署服务器。
- 回滚点：提交后执行 `git revert <本轮提交哈希>` 回退跟踪文件；停止监听 `57723` 的 Node 进程即可关闭本地预览。

## 2026-07-23 - Task: 更新 OpenAI 5.6 与 Anthropic 经典模型目录

### What was done

- 保留 OpenAI `gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna` 三个文本模型，并让本地模型定价页完整显示三项及其上下文阶梯价格。
- Anthropic 目录调整为 `claude-opus-4-8`、`claude-opus-4-7`、`claude-opus-4-6`、`claude-sonnet-5`，移除旧的 Sonnet 4.6 与 Haiku 4.5 条目。
- 同步更新固定模型支持清单、模型种子校验、官方模型种子说明、部署验收数量和本地截图数据；模型调用链、路由和计费结构未改动。

### Testing

- 先以模型数量和名称回归锁定旧种子失败，再完成种子和支持清单更新。
- `go test -count=1 ./...`、`go vet ./...`、`go build ./...` 通过。
- `npm.cmd run test:ui`、`npm.cmd run capture` 通过。
- 本地预览 `/api/models` 返回 9 个演示模型，包含 3 个 OpenAI 5.6 变体和 4 个 Anthropic 经典模型；真实上游同步返回 20 个模型，其中 4 个匹配当前目录。
- `node data/ui-preview/verify.mjs` 通过，浏览器运行时错误为 0，页面宽度和滚动宽度均为 1440；`git diff --check` 和 Node 语法检查通过。

### Notes

- `apps/api/internal/modelseed/data/2026-07-23-v2.json`：更新 Anthropic 模型集合和对应整数价格，种子总数调整为 21。
- `apps/api/internal/modelseed/support_rc21.go`：允许新的 Anthropic 模型标识进入固定支持清单。
- `apps/api/internal/modelseed/support_rc21_external_test.go`：将新增模型纳入固定源码交集测试的开发种子例外集合。
- `apps/api/internal/modelseed/catalog_test.go`：锁定 21 个模型、OpenAI 5.6 三变体和 Anthropic 四项，并禁止旧条目回归。
- `apps/web/scripts/capture.mjs`：更新渠道审核截图数据中的 Anthropic 模型。
- `apps/web/artifacts/admin-channels-desktop.png`：更新渠道管理桌面截图。
- `apps/web/artifacts/admin-settings-desktop.png`：更新平台设置桌面截图。
- `apps/web/artifacts/admin-settings-mobile.png`：更新平台设置移动端截图。
- `docs/官方模型种子.md`、`docs/运营管理与升级验收.md`、`docs/部署环境审计.md`、`docs/部署运行手册.md`：同步模型目录和验收数量。
- `data/ui-preview/vite.config.mjs`：Git 忽略的本地预览模型目录已同步更新，不进入正式提交。
- 回滚点：本轮提交前的 `e893b6a`；执行 `git revert <本轮提交哈希>` 可回退跟踪文件和截图，模型种子回退后需重新执行种子导入。

## 2026-07-23 - Task: 本地预览展示完整模型种子

### What was done

- 修复本地预览只显示 9 个演示模型的问题，改为启动时读取完整 `2026-07-23-v2` 模型种子。
- 管理后台“模型与定价”页面现在展示完整 21 个模型，包含所有 OpenAI、Anthropic、Gemini、DeepSeek、Qwen 和 Moonshot 条目；渠道同步仍使用相同目录匹配。

### Testing

- 本地 `GET /api/models` 返回 21 个模型，厂商和型号清单与种子一致。
- `node data/ui-preview/verify.mjs` 通过，真实浏览器确认模型定价页显示 21 行、关键 OpenAI 5.6 和 Claude 型号均可见；渠道弹窗、默认上游和一键获取模型回归仍通过。
- `node --check data/ui-preview/vite.config.mjs` 通过，页面运行时错误为 0，根页面无横向溢出。

### Notes

- `data/ui-preview/vite.config.mjs`：从版本化模型种子动态生成完整本地预览模型列表，文件位于 Git 忽略目录。
- `data/ui-preview/verify.mjs`：增加模型定价页 21 行和关键型号的真实浏览器断言，文件位于 Git 忽略目录。
- `docs/部署运行手册.md`：说明本地模型页面读取完整种子而非精简演示数据。
- 凭据仍只存在当前本地预览进程环境，未写入 Git、文档、进度日志或接口响应；本轮未上传服务器。
- 回滚点：停止监听 `57723` 的 Node 进程可关闭预览；恢复 `data/ui-preview` 到本轮前本地状态即可撤销本地预览变更。

## 2026-07-23 - Task: 修复账户设置移动端侧栏布局

### What was done

- 修复窄屏进入“账户设置”时控制台侧栏横向滚动导致前置入口被截断的问题。
- 控制台侧栏在移动端改为两列换行，五个入口同时可见；桌面端和管理后台侧栏布局保持不变。

### Testing

- `npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面和移动端截图已重新生成。
- `node data/ui-preview/account-settings-check.mjs` 通过：桌面与移动端文案完整，移动端 `scrollWidth` 等于视口宽度。
- `node data/ui-preview/verify.mjs` 通过：模型定价页 21 行、运行时错误为 0、根页面无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/styles.css`：为移动端控制台侧栏增加两列换行布局，避免账户设置入口触发横向滚动定位。
- `apps/web/artifacts/admin-settings-desktop.png`：按最新本地构建更新桌面验收截图。
- `apps/web/artifacts/admin-settings-mobile.png`：按最新本地构建更新移动端验收截图。
- `apps/web/artifacts/console-mobile.png`：按最新本地构建更新控制台移动端验收截图。
- `apps/web/artifacts/route-modal-mobile.png`：按最新本地构建更新路由弹窗移动端验收截图。
- `docs/部署运行手册.md`：补充窄屏控制台侧栏验收要求。
- 回滚点：本轮仅修改前端样式、验收截图和文档；执行 `git revert <本轮提交哈希>` 可回退，未上传或部署服务器。

## 2026-07-23 - Task: 移除调用记录页重复用量趋势

### What was done

- 删除控制台“调用记录”页重复显示的用量趋势，只保留调用日志明细。
- 账户概览页的用量趋势和 `/api/usage` 数据读取保持不变。

### Testing

- `npm.cmd run test:ui` 通过。
- 真实浏览器进入控制台“调用记录”页，确认页面不再出现“用量趋势”，调用日志区域仍正常显示。
- `git diff --check` 通过。

### Notes

- `apps/web/src/views/ConsoleView.jsx`：调用记录分支仅渲染 `UsageLogTable`。
- `docs/运营管理与升级验收.md`：同步账户概览与调用记录页的展示边界。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复调用记录页趋势区域；本轮未上传或部署服务器。

## 2026-07-23 - Task: 商户报价改为显示调用统计

### What was done

- 删除商户模型报价表中的“当前价格”，改为显示调用次数、Token 用量和最后调用时间。
- 调用次数与 Token 用量按“商户 + 模型”关联的最终消费日志累计，调整倍率后统计不清零；千以内不带单位，之后依次使用小写 `k/m/b`，最后调用使用整数分钟、小时或天的相对时间。
- 结算同步保留最终日志中的输入、输出和总 Token 元数据，继续以核心服务最终日志为事实来源，不改计费和分账逻辑。

### Testing

- PostgreSQL 集成测试通过：真实结算来源汇总为 `1` 次调用、`2.6B` Token，返回对应最后调用时间，并在调整倍率生成新报价版本后保持统计不变。
- `go test -p=1 -count=1 ./...`、`go vet ./...`、`go build ./...` 通过；PostgreSQL 集成包串行使用同一测试库。
- `npm.cmd run test:ui` 通过，确认商户报价显示 `32.4k`、`2.6b`、`10分钟前`，且不再出现“当前价格”。
- `npm.cmd run capture` 通过，桌面和移动端商户报价截图已更新，根页面无横向溢出。
- `node data/ui-preview/verify.mjs` 通过，浏览器运行时错误为 0。
- `git diff --check` 通过。

### Notes

- `apps/api/internal/httpapi/routing_store.go`：商户报价响应增加调用次数、Token 用量和最后调用时间，并按商户与模型聚合已结算消费来源。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：增加商户调用统计的真实 PostgreSQL 回归。
- `apps/api/internal/httpapi/settlement_domain.go`：保留最终日志中的 Token 元数据供报价统计使用。
- `apps/api/internal/httpapi/settlement_domain_test.go`：增加输入、输出 Token 元数据解析回归。
- `apps/web/src/views/MerchantView.jsx`：模型报价表改为三项调用统计和整数相对时间。
- `apps/web/src/styles.css`：增加商户报价调用统计布局。
- `apps/web/scripts/ui-smoke.mjs`、`apps/web/scripts/capture.mjs`：更新交互与截图验收契约。
- `apps/web/artifacts/merchant-offers-desktop.png`、`apps/web/artifacts/merchant-offers-mobile.png`：更新商户报价桌面和移动端截图。
- `docs/模型报价路由与健康.md`：记录商户报价统计字段及来源。
- `data/ui-preview/vite.config.mjs`、`data/ui-preview/verify.mjs`：更新 Git 忽略的本地预览统计数据与浏览器断言。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复价格展示和旧接口；本轮未上传或部署服务器。

## 2026-07-23 - Task: 调整商户报价统计列与单位大小写

### What was done

- 将调用次数、Token 用量、最后调用从报价行内的统计块拆为三个独立表格列，列名直接显示在表头。
- 调用量紧凑单位统一为大写 `K/M/B`，千以内保持无单位；相对时间继续使用整数分钟、小时或天。
- 同步前端烟熏断言、截图验收和模型报价说明，移除不再使用的统计块样式与图标导入。

### Testing

- `npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面与移动端截图确认独立列和 `32.4K`、`2.6B`、`10分钟前`。
- `node data/ui-preview/verify.mjs` 通过，模型行数 `21`、渠道行数 `3`、运行时错误 `0`、根页面无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/views/MerchantView.jsx`：报价表拆分统计列并统一大写紧凑单位。
- `apps/web/src/styles.css`：删除已无引用的统计块样式。
- `apps/web/scripts/ui-smoke.mjs`、`apps/web/scripts/capture.mjs`、`data/ui-preview/verify.mjs`：更新大小写和独立列验收断言。
- `docs/模型报价路由与健康.md`：记录统计独立列及 `K/M/B` 展示规则。
- `apps/web/artifacts/merchant-offers-desktop.png`、`apps/web/artifacts/merchant-offers-mobile.png`：更新本地验收截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可撤销本轮表格展示调整；本轮未上传或部署服务器。

## 2026-07-23 - Task: 同步模型市场调用统计列

### What was done

- 模型市场删除“当前价格”列，增加调用次数、Token 用量和最后调用三个独立列，展示规则与商户报价一致。
- 公开市场报价接口按商户与标准模型读取已结算消费来源，返回 `call_count`、`token_usage` 和 `last_called_at`，不改变原有计费链路。
- 本地预览、桌面/移动端截图和市场浏览器断言同步更新；本地默认上游地址只用于模拟同步，不保存密钥。

### Testing

- `go test -p=1 -count=1 ./internal/httpapi` 通过，包含市场统计 PostgreSQL 回归。
- `go vet ./...`、`go build ./...` 待本轮后续命令完成后记录。
- `npm.cmd run test:ui` 通过，市场独立列名和统计展示断言通过。
- `npm.cmd run capture` 通过，模型市场桌面和移动端截图更新，根页面无横向溢出。
- `node data/ui-preview/verify.mjs` 通过，模型行数 `21`、渠道行数 `3`、运行时错误 `0`。

### Notes

- `apps/api/internal/httpapi/routing_store.go`：市场报价增加消费统计字段并通过报价版本关联结算来源。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：增加市场调用次数、Token 用量和最后调用时间回归。
- `apps/web/src/api.js`：抽取共享的 `K/M/B` 和相对时间格式化函数。
- `apps/web/src/views/MerchantView.jsx`、`apps/web/src/views/MarketView.jsx`：统一商户报价与模型市场统计列展示。
- `apps/web/scripts/ui-smoke.mjs`、`apps/web/scripts/capture.mjs`、`data/ui-preview/verify.mjs`：同步市场模拟数据和列名断言。
- `docs/模型报价路由与健康.md`：同步模型市场统计字段说明。
- `data/ui-preview/vite.config.mjs`：本地预览默认地址和动态相对时间，仅为忽略目录运行夹具。
- 回滚点：执行 `git revert <本轮提交哈希>` 可撤销模型市场统计展示和接口字段；本轮未上传或部署服务器。

## 2026-07-23 - Task: 补充模型市场构建验证

### What was done

- 完成模型市场统计改动后的 Go 静态检查和构建验证。

### Testing

- `go vet ./...` 通过。
- `go build ./...` 通过。

### Notes

- 本条为上一条模型市场任务的验证补充，不增加业务代码。
- 回滚点：沿用上一条模型市场任务的提交回滚点。

## 2026-07-23 - Task: 移除模型市场可用渠道重复文案

### What was done

- 删除模型市场每行数量下方重复的“可用渠道”小字，仅保留表头列名和渠道数量。

### Testing

- `npm.cmd run test:ui` 通过，断言页面只保留一个“可用渠道”文本。
- `npm.cmd run capture` 通过，桌面与移动端模型市场截图已更新。
- `node data/ui-preview/verify.mjs` 通过，运行时错误为 `0`，根页面无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/views/MarketView.jsx`：移除渠道数量行内重复说明。
- `apps/web/scripts/ui-smoke.mjs`、`data/ui-preview/verify.mjs`：增加重复文案回归断言。
- `apps/web/artifacts/market-desktop.png`：更新桌面端市场截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复行内重复文案；本轮未上传或部署服务器。

## 2026-07-23 - Task: 调整模型市场统计列字重

### What was done

- 将模型市场中的调用次数、Token 用量、最后调用和可用渠道数量改为普通字重，保持表头和其他重点状态样式不变。

### Testing

- `npm.cmd run test:ui` 通过，浏览器字重断言通过。
- `npm.cmd run capture` 通过，桌面与移动端截图已更新。
- `node data/ui-preview/verify.mjs` 通过，运行时错误为 `0`、根页面无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/views/MarketView.jsx`：移除四个统计单元格的粗体标签。
- `apps/web/scripts/ui-smoke.mjs`、`data/ui-preview/verify.mjs`：增加四列普通字重回归断言。
- `apps/web/artifacts/market-desktop.png`、`apps/web/artifacts/market-mobile.png`：更新验收截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复原字重；本轮未上传或部署服务器。

## 2026-07-23 - Task: 将 OpenAI 测试地址归入管理员商户渠道

### What was done

- 将本地预览管理员“ModelRoute 官方”商户的已导入渠道设为 `OpenAI 测试渠道`，地址显示为 `https://api.lwylink.xyz`，只保留密钥指纹。
- 移除新增商户渠道弹窗的默认上游地址和默认密钥状态，新增渠道必须主动填写后再同步模型。
- 同步管理员商户渠道夹具、模型报价渠道名称和新增渠道流程测试；密钥正文未写入代码、页面或日志。

### Testing

- `npm.cmd run test:ui` 通过，确认管理员渠道地址展示、新增渠道空白表单和显式填写同步流程。
- `npm.cmd run capture` 通过，桌面与移动端截图生成完成。
- `node data/ui-preview/verify.mjs` 通过，模型行数 `21`、渠道行数 `3`、运行时错误为 `0`。
- `git diff --check` 通过。

### Notes

- `apps/web/src/Modals.jsx`：移除新增渠道表单对默认渠道配置接口的读取和自动填充。
- `apps/web/scripts/ui-smoke.mjs`、`apps/web/scripts/capture.mjs`：同步管理员测试渠道夹具和显式填写测试。
- `data/ui-preview/vite.config.mjs`：更新忽略目录中的管理员测试渠道与本地同步夹具。
- `docs/模型报价路由与健康.md`：记录管理员专用测试渠道边界。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复默认地址行为和旧渠道夹具；本轮未上传或部署服务器。

## 2026-07-23 - Task: 优化用户控制台账户设置布局

### What was done

- 将账户设置从横向大块资料排布调整为身份摘要、工作区操作和账号资料的清晰分区。
- 右侧登录账号、邮箱和用户类型改为独立行高，避免内容挤在一起；移动端改为上下排列并保持相同信息层级。
- 商户入口、商户申请和审核提示继续沿用原有权限与交互，不改变账号接口和业务数据。

### Testing

- `npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面与移动端截图已更新。
- `node data/ui-preview/account-settings-check.mjs` 通过，桌面和移动端无横向溢出，账号资料文案完整。
- `node data/ui-preview/verify.mjs` 通过，模型目录 21 行、渠道 3 行、运行时错误为 0。
- `git diff --check` 通过。

### Notes

- `apps/web/src/views/ConsoleView.jsx`：重排账户设置 JSX，保留身份、工作区入口和申请状态逻辑。
- `apps/web/src/styles.css`：新增账户设置双栏与移动端堆叠样式，增加右侧资料行高。
- `docs/账号与管理员接口.md`：补充账户设置布局和审核提示展示规则。
- `data/ui-preview/account-settings-desktop.png`、`data/ui-preview/account-settings-mobile.png`：更新本地验收截图（忽略目录）。
- `apps/web/artifacts/admin-settings-desktop.png`：按本地构建更新截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退本轮账户设置代码、文档和截图；本轮仅本地修改，未上传服务器。

## 2026-07-23 - Task: 统一列表操作菜单

### What was done

- 将用户密钥、商户渠道与报价、管理员公告、渠道、商户审核、模型、支付订单、兑换码、用户、提现和对账列表的多按钮操作统一收纳到三点下拉菜单。
- 菜单支持危险操作标色、禁用状态、点击外部关闭、Esc 关闭和窗口尺寸变化关闭；固定定位避免被表格滚动容器裁切。
- 更新桌面与移动端截图脚本及浏览器回归，确保具体操作仍可从菜单触发。

### Testing

- `npm.cmd run test:ui` 通过，覆盖密钥路由、渠道审核/恢复、报价取消、模型发布/删除、兑换码、支付入账、提现和对账操作。
- `npm.cmd run capture` 通过，桌面和移动端页面截图生成完成，根页面无横向溢出。
- `node data/ui-preview/verify.mjs` 通过，模型目录 21 行、渠道 3 行、运行时错误为 0。
- `node data/ui-preview/account-settings-check.mjs` 通过，桌面和移动端控制台布局无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/ui.jsx`：新增可复用 `ActionMenu` 组件。
- `apps/web/src/styles.css`：新增三点触发器、固定定位菜单、禁用和危险动作样式。
- `apps/web/src/views/ConsoleView.jsx`、`apps/web/src/views/MerchantView.jsx`：收纳密钥、渠道和报价操作。
- `apps/web/src/views/admin/*.jsx`：收纳公告、渠道、商户审核、模型、支付、兑换码、用户、提现和对账操作。
- `apps/web/scripts/ui-smoke.mjs`、`apps/web/scripts/capture.mjs`：更新菜单交互回归和截图流程。
- `docs/账号与管理员接口.md`：补充列表操作菜单行为说明。
- `apps/web/artifacts/*.png`：按本地构建更新受影响的桌面和移动端截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退统一菜单代码、测试、文档和截图；本轮仅本地修改，未上传服务器。

## 2026-07-23 - Task: 调整顶部用户身份摘要

### What was done

- 右上角显示账号昵称，头像自动取昵称中的第一个汉字、字母或数字，英文字母统一转为大写显示。
- 身份文案继续显示管理员、商户或普通用户；身份摘要改为静态展示，移除点击跳转控制台行为。
- 更新生产界面验收脚本，确认桌面头像首字符、身份文案和非按钮语义；移动端隐藏规则保持原样。

### Testing

- `npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面与移动端截图生成完成。
- `node data/ui-preview/verify.mjs` 通过，运行时错误为 0、根页面无横向溢出。
- `node data/ui-preview/account-settings-check.mjs` 通过，控制台桌面和移动端布局无横向溢出。
- `git diff --check` 通过。

### Notes

- `apps/web/src/main.jsx`：新增头像首字符提取逻辑，将可点击按钮改为静态身份摘要。
- `apps/web/src/styles.css`：同步身份摘要和移动端隐藏样式，移除 hover 点击反馈。
- `apps/web/scripts/production-smoke.mjs`：更新头像静态展示和模型操作菜单验收。
- `docs/账号与管理员接口.md`：补充顶部昵称、首字符头像和身份文案规则。
- `progress.md`：追加本轮验证和回滚记录。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退顶部身份摘要代码、测试和文档；本轮仅本地修改，未上传服务器。

## 2026-07-23 - Task: 为高记录量列表增加分页

### What was done

- 新增统一分页控件，默认每页 20 条，可切换 50 或 200 条，并提供首页、上一页、下一页和尾页。
- 模型市场在筛选和排序结果上分页；用户调用记录、充值记录、管理员用户、渠道调用日志和管理员充值订单按接口页码重新读取。
- 搜索、类型、商户和渠道筛选变化后自动回到第一页，移动端分页栏支持换行且不产生横向溢出。

### Testing

- `npm.cmd run test:ui` 通过，覆盖模型市场、用户调用记录、管理员渠道日志、用户列表和充值订单分页控件。
- `npm.cmd run capture` 通过，桌面和移动端截图生成完成，根页面无横向溢出。
- `node data/ui-preview/verify.mjs` 通过，运行时错误为 0。
- `node data/ui-preview/account-settings-check.mjs` 通过，控制台桌面和移动端布局无横向溢出。
- `node --check apps/web/scripts/ui-smoke.mjs; node --check apps/web/scripts/production-smoke.mjs` 通过。
- `git diff --check` 通过。

### Notes

- `apps/web/src/ui.jsx`：新增可复用分页组件及首页、尾页图标按钮。
- `apps/web/src/styles.css`：新增桌面和移动端分页布局样式。
- `apps/web/src/views/MarketView.jsx`：增加模型市场筛选结果分页。
- `apps/web/src/views/UsagePanels.jsx`：为调用日志表接入分页元数据和事件。
- `apps/web/src/views/ConsoleView.jsx`：为用户调用记录和充值记录接入服务端分页。
- `apps/web/src/views/admin/AdminUsers.jsx`：为用户管理接入关键词、类型筛选后的服务端分页。
- `apps/web/src/views/admin/AdminUsage.jsx`：为渠道调用日志接入商户、渠道筛选后的服务端分页。
- `apps/web/src/views/admin/AdminPayment.jsx`：为充值订单接入服务端分页。
- `apps/web/scripts/ui-smoke.mjs`：增加分页栏、条数选项和四个跳页按钮回归断言。
- `docs/账号与管理员接口.md`：记录分页参数和适用页面。
- `apps/web/artifacts/*.png`：按本地构建更新受影响的桌面和移动端截图。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退分页组件、页面接入、测试、文档和截图；本轮仅本地修改，未上传服务器。

## 2026-07-23 - Task: 补充分页跳页回归

### What was done

- 将用户管理冒烟数据扩展为多页场景，补充下一页到第 2/3 页以及首页返回第 1/3 页的自动化断言。

### Testing

- `npm.cmd run test:ui` 通过，实际完成用户列表下一页和首页跳转验证。
- `node --check apps/web/scripts/ui-smoke.mjs; node --check apps/web/scripts/production-smoke.mjs` 通过。
- `git diff --check` 通过。

### Notes

- `apps/web/scripts/ui-smoke.mjs`：增加多页用户列表分页跳转回归数据和断言。
- `progress.md`：追加本次回归验证记录。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退分页跳页回归测试；本轮仅本地修改，未上传服务器。

## 2026-07-23 - Task: 完整项目审计与开发阶段简化

### What was done

- 删除后端、前端和测试中的零引用函数、组件、导入与辅助代码，统一内部错误文案，并使用 `staticcheck` 把 Go 全包静态检查收敛到零告警。
- 删除旧仪表盘、分组编辑、旧健康事件和旧支付列表遗留的无引用样式；移除已经删除的系统任务截图和三份被 V2 规范替代的早期计划文档。
- 将前端包名、页面标题、描述和 README 从原型口径统一为正式 ModelRoute 平台，并清除已经不存在的根目录旧原型忽略规则。
- 修正公告头像交互文档和 V2 迁移清单，移除当前运营文档内已经失效的 V1 验收数据；保留内网核心 Embeddings 契约作为固定核心版本完整性校验，公网仍不注册该接口。
- 对后端路由、前端调用、部署验收、依赖、跟踪凭据、模型证据链接和本地预览进程做了交叉检查；数据库中多模型渠道的代表模型字段涉及结构迁移，本轮未混入普通清理。

### Testing

- `staticcheck ./...`、`go test ./...`、`go vet ./...`、`go build ./cmd/api ./cmd/worker ./cmd/seed-models` 全部通过。
- `go mod tidy -diff` 无输出，Go 模块文件无需变更。
- `npm.cmd ls --all --depth=0` 通过，前端直接依赖完整；`npm.cmd run test:ui` 通过。
- `npm.cmd run capture` 通过，桌面与移动端页面无横向溢出、运行时错误为 0。
- `node data/ui-preview/verify.mjs` 通过，本地 `127.0.0.1:57723` 返回正式页面标题，模型 21 条、渠道 3 条、页面宽度与滚动宽度均为 1440。
- `python -m unittest discover -s deploy/tests -p 'test_*.py'` 共 27 项通过。
- `docker compose --env-file .env.example --profile tools config --quiet` 与 `git diff --check` 通过。
- Git 跟踪内容凭据模式扫描仅命中示例占位值和隔离契约测试口令，未发现生产上游、支付或服务器凭据。

### Notes

- `.dockerignore`：删除不存在的根目录旧原型排除项。
- `.gitignore`：删除不存在的根目录旧原型忽略项。
- `README.md`：将前端描述更新为正式三角色统一界面。
- `apps/api/cmd/seed-models/main.go`：统一核心服务管理员校验错误文案。
- `apps/api/internal/bootstrap/resources.go`：统一启动期核心服务管理员校验错误文案。
- `apps/api/internal/dependencies/checks.go`：统一核心服务健康检查错误文案。
- `apps/api/internal/httpapi/account_handlers.go`：删除零引用查询参数辅助函数。
- `apps/api/internal/httpapi/pricing_sync_test.go`：删除零引用 JSON 测试辅助函数。
- `apps/api/internal/httpapi/routing_handlers.go`：删除零引用报价别名生成函数及导入。
- `apps/api/internal/httpapi/server.go`：删除未注册、零引用的部署中占位响应。
- `apps/api/internal/httpapi/usage_handlers.go`：删除零引用日志清理包装函数。
- `apps/api/internal/newapi/client.go`：统一核心服务客户端错误文案。
- `apps/api/internal/newapi/service_account.go`：统一核心服务账号和探活令牌错误文案。
- `apps/web/index.html`：使用正式平台标题和页面描述。
- `apps/web/package.json`：将前端包名更新为 `modelroute-web`。
- `apps/web/package-lock.json`：同步正式前端包名。
- `apps/web/src/api.js`：删除零引用报价格式化函数。
- `apps/web/src/styles.css`：删除旧页面和零引用组件样式及响应式规则。
- `apps/web/src/ui.jsx`：删除零引用分段控件、外链占位组件及图标导入。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：删除零引用按钮导入。
- `apps/web/artifacts/admin-tasks-desktop.png`：删除已下线系统任务桌面截图。
- `apps/web/artifacts/admin-tasks-mobile.png`：删除已下线系统任务移动端截图。
- `apps/web/artifacts/admin-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/admin-redemptions-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/admin-settings-desktop.png`：按清理后的前端构建刷新桌面验收图。
- `apps/web/artifacts/admin-settings-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/admin-usage-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/announcement-panel-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/console-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/market-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `apps/web/artifacts/payment-config-mobile.png`：按清理后的前端构建刷新移动端验收图。
- `docs/superpowers/plans/2026-07-22-ModelRoute完整实施计划.md`：删除被当前 V2 文档替代的旧计划。
- `docs/superpowers/plans/2026-07-22-UI原型实施计划.md`：删除已经完成且失效的原型计划。
- `docs/superpowers/specs/2026-07-22-API聚合平台设计规范.md`：删除与当前 V2 行为冲突的早期规范。
- `docs/公告管理.md`：同步头像静态身份摘要的当前交互。
- `docs/运营管理与升级验收.md`：删除 V1 历史验收段落并修正 V2 迁移清单。
- `progress.md`：追加本轮审计、验证和回滚记录。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复本轮全部清理、文档和截图；本轮只更新本地仓库与本地预览，未上传服务器。

## 2026-07-23 - Task: 统一调用记录并简化商户模型与财务界面

### What was done

- 用户、商户和管理员调用记录统一改读 `settlement_sources` 成功消费事实，默认不限制明细时间范围，并按模型、商户用户 ID、平台渠道 ID 和 `20/50/200` 页大小查询。
- 用户页面显示承载商户，商户页面显示渠道和 90% 自动入账金额，管理员页面显示消费用户、商户和渠道；三种视角共用调用记录表格，不展示退款行、状态列或 Token 合计。
- 修复商户调用记录的范围覆盖问题：即使请求手工传入其他商户用户 ID，查询仍强制使用当前登录商户。
- 商户渠道生命周期增加审核中撤销和驳回记录删除；正式渠道继续支持同步模型、下架、恢复和删除，并保持跨商户权限隔离。
- 商户“模型报价”和独立“健康状态”合并为“上架模型”，支持修改模型价格、上架、下架、删除和按需展开渠道健康；删除后保留历史调用关系。
- 删除产品层商户账本、单次退款、人工结算和人工对账入口及无用读取代码；成功调用继续自动完成商户 90%、平台 10% 的内部幂等入账，核心服务失败冲正、余额、提现和内部账本继续保留。
- 更新桌面与移动端截图验收，新增用户调用记录、账户设置、商户调用记录和渠道健康展开页面；本轮只完成本地开发、验证和提交，不发布服务器。

### Testing

- `go test ./internal/httpapi -run TestMerchantUsageUsesSessionMerchantAndReturnsNetIncome -count=1`：先复现商户筛选覆盖失败，修复后通过。
- `go test ./... -count=1`、`go vet ./...`、`go build ./...`、`staticcheck ./...` 全部通过。
- `npm.cmd run test:ui` 通过，覆盖三套侧栏、统一调用筛选、渠道审核撤销/删除、模型价格修改、上下架、删除、渠道健康展开、分页和权限隐藏。
- `npm.cmd run capture` 通过，桌面与移动端截图均非空，根页面无横向溢出；人工查看账户设置、用户调用记录、商户调用记录和渠道健康展开未发现内容重叠。
- `python -m unittest discover -s deploy/tests -p 'test_*.py'` 共 27 项通过。
- `docker compose config --quiet` 通过；本地没有加载生产环境文件，因此仅输出缺少生产变量的预期警告。
- 变更 Go 文件 `gofmt -d` 无差异；三个前端脚本 `node --check`、`git diff --check` 和前端旧产品术语扫描通过。
- Git 跟踪内容凭据扫描未命中聊天中提供的支付密钥、上游密钥或服务器密码。

### Notes

- `apps/api/internal/bootstrap/resources.go`：把结算存储注册为统一调用记录读取器。
- `apps/api/internal/httpapi/channel_models_handlers.go`：增加商户撤销或删除渠道审核记录处理器。
- `apps/api/internal/httpapi/merchant_handlers.go`：将通用分页上限调整为 200 条。
- `apps/api/internal/httpapi/merchant_store.go`：实现渠道审核记录的归属校验、状态限制和删除事务。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖审核撤销、驳回删除、处理中冲突和跨商户隔离。
- `apps/api/internal/httpapi/merchant_types.go`：增加渠道审核记录管理接口。
- `apps/api/internal/httpapi/operations_handlers_test.go`：把管理员渠道日志测试改为统一调用事实查询。
- `apps/api/internal/httpapi/pricing_sync.go`：支持价格同步后按调用方要求保持下架或激活。
- `apps/api/internal/httpapi/pricing_sync_test.go`：覆盖激活和保持下架两种价格同步结果。
- `apps/api/internal/httpapi/routing_handlers.go`：实现上架模型修改价格、上架、下架和删除处理流程。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖模型生命周期、同步失败和下架后调价。
- `apps/api/internal/httpapi/routing_store.go`：实现模型最新状态、删除标记、上下架同步和成功调用统计。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖删除隐藏、下架保留、可用渠道和历史统计。
- `apps/api/internal/httpapi/server.go`：注册新商户调用记录和模型生命周期接口，移除旧账本、健康和对账路由。
- `apps/api/internal/httpapi/services.go`：增加统一调用记录读取依赖。
- `apps/api/internal/httpapi/settlement_handlers.go`：删除商户账本和管理员人工对账产品处理器。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：确认旧账本和人工对账路由不再开放，并保留提现回归。
- `apps/api/internal/httpapi/settlement_reconciliation.go`：删除无产品入口的人工对账读取和忽略实现。
- `apps/api/internal/httpapi/settlement_types.go`：从产品服务接口移除账本列表和人工对账类型。
- `apps/api/internal/httpapi/usage_handlers.go`：实现用户、商户和管理员统一调用记录接口，并强制商户会话隔离。
- `apps/api/internal/httpapi/usage_records.go`：新增基于成功消费事实的共用查询、筛选、分页和字段映射。
- `apps/api/internal/httpapi/usage_records_test.go`：覆盖无默认保留窗口、三种查询范围、筛选、字段和越权参数。
- `apps/api/internal/migrations/migrate_test.go`：将迁移验收更新到第 3 份迁移。
- `apps/api/internal/migrations/sql/000003_offer_deleted_status.sql`：为模型价格版本增加 `deleted` 状态。
- `apps/web/src/Modals.jsx`：统一“设置模型价格”文案并保留 `0.00x..5.00x` 输入。
- `apps/web/src/styles.css`：增加调用筛选、Token 明细和渠道健康展开的桌面与移动端样式。
- `apps/web/src/ui.jsx`：移除已下线退款和结算状态的界面色彩映射。
- `apps/web/src/views/AdminView.jsx`：从运营待办和计数中移除人工对账。
- `apps/web/src/views/ConsoleView.jsx`：增加用户调用记录模型与商户筛选并接入服务端分页。
- `apps/web/src/views/MerchantView.jsx`：合并上架模型与健康，接入审核撤销、模型生命周期和商户调用记录。
- `apps/web/src/views/UsagePanels.jsx`：重做三种视角共用的调用记录表格和独立 Token 字段。
- `apps/web/src/views/admin/AdminUsage.jsx`：改为按商户、渠道和模型筛选统一调用记录。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：删除人工对账队列，只保留人工提现审核。
- `apps/web/scripts/ui-smoke.mjs`：更新 mock 契约并增加统一调用、渠道审核和模型生命周期回归。
- `apps/web/scripts/capture.mjs`：移除旧接口 mock，增加账户设置、三类调用记录和渠道健康展开截图。
- `apps/web/scripts/production-smoke.mjs`：同步商户五项侧栏和“上架模型”验收口径。
- `apps/web/artifacts/admin-announcements-{desktop,mobile}.png`、`admin-channels-{desktop,mobile}.png`、`admin-{desktop,mobile}.png`：刷新公告、渠道和管理概览截图。
- `apps/web/artifacts/admin-image-price-desktop.png`、`admin-model-price-desktop.png`、`admin-models-{desktop,mobile}.png`：刷新模型和价格截图。
- `apps/web/artifacts/admin-redemptions-{desktop,mobile}.png`、`admin-settings-{desktop,mobile}.png`、`admin-usage-{desktop,mobile}.png`、`admin-withdrawals-{desktop,mobile}.png`：刷新运营功能截图。
- `apps/web/artifacts/announcement-panel-{desktop,mobile}.png`、`console-{desktop,mobile}.png`、`market-{desktop,mobile}.png`、`merchant-{desktop,mobile}.png`：刷新三工作区与公告截图。
- `apps/web/artifacts/merchant-offers-{desktop,mobile}.png`、`payment-config-desktop.png`、`withdrawal-modal-{desktop,mobile}.png`：刷新上架模型、支付和提现截图。
- `apps/web/artifacts/console-settings-{desktop,mobile}.png`、`console-usage-{desktop,mobile}.png`、`merchant-channel-health-{desktop,mobile}.png`、`merchant-usage-{desktop,mobile}.png`：新增账户设置、用户调用、渠道健康和商户调用截图。
- `docs/统一调用记录.md`：新增统一消费事实、权限视角、永久明细和内部入账边界说明。
- `docs/账号与管理员接口.md`：同步商户五项侧栏和商户调用记录分页。
- `docs/模型报价路由与健康.md`：同步“上架模型”、模型生命周期和商户调用接口。
- `docs/运营管理与升级验收.md`：同步永久调用明细、运营筛选和已删除产品入口。
- `docs/部署运行手册.md`：将开发重置后的平台迁移清单更新为三份迁移。
- `docs/商户与渠道审核接口.md`：补充审核撤销、驳回删除和统一调用事实。
- `docs/结算与人工提现.md`：明确成功调用自动入账、失败自动退回和无人工结算页面。
- `progress.md`：追加本轮实现、测试、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退统一调用记录、渠道审核、上架模型、界面简化、文档和截图；本轮未上传服务器。

## 2026-07-23 - Task: 完成独立路由、缓存指标与支付宝提现收尾

### What was done

- 将路由策略改为可被多把 API 密钥复用的独立资源，创建密钥前必须先创建并绑定路由；支持智能路由、固定商户和均衡、价格、成功率、响应速度、缓存命中率五种加权优先方式。
- 路由采用先商户、后渠道的两级加权抽取，候选权重范围固定为 `1..4`；首字节前失败时对剩余渠道无放回重试，避免多渠道商户天然获得额外首选流量。
- 智能路由增加可选最低缓存命中率，文本模型样本不足或低于阈值时退出候选，达到阈值即可参与；图片模型忽略该条件。
- 增加最近 7 天缓存指标小时桶、滚动快照和 Worker 每分钟刷新，市场、商户上架模型、渠道详情及路由候选统一使用同一指标口径。
- 完成商户名称格式、长度和唯一性校验，并在商户中心显示商户名称；修复账户设置中商户申请模型目录引用失效的问题。
- 用户、商户和管理员调用记录继续保留服务端请求编号，但三个前端主表均隐藏该列。
- 提现固定为支付宝，最低 `¥50` 且只接受人民币整数；申请冻结完整金额，数据库以整数生成 3% 税费和实际到账金额，驳回完整释放，确认打款后扣除完整冻结金额。
- 商户提现页增加余额门槛、差额提示、税费试算和申请后刷新；管理员提现页增加状态筛选、刷新、`20/50/200` 分页及实际支付宝打款提示。
- 刷新桌面与 390px 移动端验收截图。本轮仅更新本地仓库和本地预览，不发布服务器。

### Testing

- `go test ./... -count=1`：Go 全量测试通过。
- `go vet ./...`、`staticcheck ./...`、`go build ./...`：静态检查与构建通过。
- 设置 `MODELROUTE_TEST_DATABASE_URL` 后执行 `go test ./internal/httpapi -count=1`：临时 PostgreSQL 集成测试通过，覆盖 `49/50/51` 元边界、整数校验、`¥100 -> ¥3 -> ¥97`、并发冻结、管理员即时可见、驳回释放和确认打款。
- 设置 `MODELROUTE_TEST_DATABASE_URL` 后执行 `go test ./internal/modelseed -count=1`：真实数据库模型种子测试通过。
- `npm.cmd run test:ui`：前端 UI 烟雾测试通过，包含小数提现不发送请求、独立路由、缓存阈值、固定商户分页、隐藏请求编号和管理员提现操作。
- `npm.cmd run capture`、`npm.cmd run build`：桌面与 390px 移动端截图刷新成功，前端生产构建通过，页面根节点无横向溢出。
- 人工检查 `route-modal`、`withdrawal-modal` 和 `admin-withdrawals` 桌面/移动截图：弹窗滚动、固定底部操作、税费明细及表格容器布局正常。
- `docker compose config --quiet`：Compose 配置解析通过；未加载本地生产环境文件时仅输出环境变量缺省提示。
- `git diff --check`：补丁空白与冲突标记检查通过。
- 工作区敏感值扫描：未发现对话中提供的上游密钥、支付密钥、完整商户号或服务器密码进入代码、文档及日志；仓库原有部署文档和重置脚本中的服务器地址保持原样，本轮未新增或扩散。

### Notes

- `apps/api/cmd/worker/main.go`：接入最近 7 天路由指标刷新任务。
- `apps/api/cmd/worker/main_test.go`：验证 Worker 指标刷新调度。
- `apps/api/internal/bootstrap/resources.go`：初始化路由指标所需资源。
- `apps/api/internal/bootstrap/resources_test.go`：同步启动资源测试。
- `apps/api/internal/httpapi/account_handlers.go`：实现独立路由绑定的密钥创建、查询、重绑和失败补偿。
- `apps/api/internal/httpapi/account_test.go`：覆盖无路由创建、路由归属、重绑和补偿清理。
- `apps/api/internal/httpapi/merchant_handlers.go`：为商户申请和上架模型返回名称及缓存指标信息。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖商户名称与缓存指标响应。
- `apps/api/internal/httpapi/merchant_store.go`：实现商户名称唯一校验和上架模型缓存快照查询。
- `apps/api/internal/httpapi/merchant_store_test.go`：覆盖名称占用、释放及缓存字段扫描。
- `apps/api/internal/httpapi/merchant_types.go`：补充商户名称和缓存指标响应字段。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：同步 PostgreSQL 运营数据测试基线。
- `apps/api/internal/httpapi/routing_domain.go`：实现两级加权抽取、五种优先方式和最低缓存阈值过滤。
- `apps/api/internal/httpapi/routing_domain_test.go`：覆盖权重上限、两级抽取、阈值边界、图片豁免和无放回顺序。
- `apps/api/internal/httpapi/routing_handlers.go`：提供独立路由增删改查、商户搜索和密钥路由绑定接口。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖路由参数、权限、固定商户和缓存阈值接口。
- `apps/api/internal/httpapi/routing_proxy.go`：将独立路由策略和缓存阈值接入模型请求选择。
- `apps/api/internal/httpapi/routing_proxy_test.go`：验证代理使用绑定路由并保留首字节重试边界。
- `apps/api/internal/httpapi/routing_store.go`：实现独立路由、密钥绑定、路由商户分页、缓存快照和加权候选数据访问。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：覆盖真实数据库路由归属、绑定和候选读取。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖路由仓储 SQL、缓存阈值和跨用户操作。
- `apps/api/internal/httpapi/server.go`：注册独立路由、密钥重绑和路由商户接口。
- `apps/api/internal/httpapi/services.go`：为服务集合补充路由指标依赖。
- `apps/api/internal/httpapi/settlement_domain.go`：定义人民币提现整数门槛和 3% 税费整数拆分。
- `apps/api/internal/httpapi/settlement_domain_test.go`：覆盖 `49/50/51/100` 元及小数提现边界。
- `apps/api/internal/httpapi/settlement_handlers.go`：固定支付宝请求结构、校验金额和账户，并增加管理员状态筛选。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：覆盖支付宝加密、旧收款类型拦截、金额错误和管理员审核链路。
- `apps/api/internal/httpapi/settlement_metrics_test.go`：覆盖七天缓存小时桶与滚动快照汇总。
- `apps/api/internal/httpapi/settlement_store.go`：写入成功调用的缓存指标桶并提供刷新入口。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：覆盖并发提现冻结、管理员即时可见、驳回释放和打款扣除。
- `apps/api/internal/httpapi/settlement_types.go`：为提现记录增加税费和实际到账字段。
- `apps/api/internal/httpapi/settlement_withdrawals.go`：实现支付宝整数提现事务、分页筛选及税费字段扫描。
- `apps/api/internal/migrations/migrate_test.go`：校验商户名称、独立路由、缓存指标和提现约束迁移。
- `apps/api/internal/migrations/sql/000001_v2_baseline.sql`：将开发基线提现收紧为支付宝、最低 50 元和人民币整数，并生成税费字段。
- `apps/api/internal/migrations/sql/000004_merchant_names.sql`：增加商户名称规范化和唯一占用结构。
- `apps/api/internal/migrations/sql/000005_independent_routes.sql`：建立独立路由、固定商户和密钥绑定结构。
- `apps/api/internal/migrations/sql/000006_route_metrics.sql`：建立缓存指标小时桶和最近 7 天快照结构。
- `apps/web/artifacts/admin-settings-desktop.png`：刷新管理员设置桌面验收图。
- `apps/web/artifacts/admin-settings-mobile.png`：刷新管理员设置移动端验收图。
- `apps/web/artifacts/admin-usage-desktop.png`：刷新隐藏请求编号后的管理员调用记录桌面图。
- `apps/web/artifacts/admin-usage-mobile.png`：刷新隐藏请求编号后的管理员调用记录移动端图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：刷新税费、筛选和分页后的管理员提现桌面图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：刷新税费、筛选和分页后的管理员提现移动端图。
- `apps/web/artifacts/console-desktop.png`：刷新独立路由侧栏后的控制台桌面图。
- `apps/web/artifacts/console-mobile.png`：刷新独立路由侧栏后的控制台移动端图。
- `apps/web/artifacts/console-keys-desktop.png`：新增密钥与独立路由绑定桌面验收图。
- `apps/web/artifacts/console-keys-mobile.png`：新增密钥与独立路由绑定移动端验收图。
- `apps/web/artifacts/console-settings-desktop.png`：刷新商户申请修复后的账户设置桌面图。
- `apps/web/artifacts/console-settings-mobile.png`：刷新商户申请修复后的账户设置移动端图。
- `apps/web/artifacts/console-usage-desktop.png`：刷新隐藏请求编号后的用户调用记录桌面图。
- `apps/web/artifacts/console-usage-mobile.png`：刷新隐藏请求编号后的用户调用记录移动端图。
- `apps/web/artifacts/market-desktop.png`：刷新缓存命中率后的模型市场桌面图。
- `apps/web/artifacts/market-mobile.png`：刷新缓存命中率后的模型市场移动端图。
- `apps/web/artifacts/merchant-channel-health-desktop.png`：刷新渠道缓存指标桌面图。
- `apps/web/artifacts/merchant-channel-health-mobile.png`：刷新渠道缓存指标移动端图。
- `apps/web/artifacts/merchant-desktop.png`：刷新商户名称标签后的商户中心桌面图。
- `apps/web/artifacts/merchant-mobile.png`：刷新商户名称标签后的商户中心移动端图。
- `apps/web/artifacts/merchant-offers-desktop.png`：刷新上架模型缓存命中率桌面图。
- `apps/web/artifacts/merchant-offers-mobile.png`：刷新上架模型缓存命中率移动端图。
- `apps/web/artifacts/merchant-usage-desktop.png`：刷新隐藏请求编号后的商户调用记录桌面图。
- `apps/web/artifacts/merchant-usage-mobile.png`：刷新隐藏请求编号后的商户调用记录移动端图。
- `apps/web/artifacts/route-modal-desktop.png`：新增最低缓存阈值和固定商户选择桌面验收图。
- `apps/web/artifacts/route-modal-mobile.png`：新增最低缓存阈值和固定商户选择移动端验收图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：刷新支付宝、3% 税费和整数金额桌面验收图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：刷新支付宝、3% 税费和整数金额移动端验收图。
- `apps/web/scripts/capture.mjs`：更新路由、缓存、提现和隐藏请求编号截图场景。
- `apps/web/scripts/ui-smoke.mjs`：覆盖独立路由、缓存阈值、提现整数边界和管理员审核 UI。
- `apps/web/src/Modals.jsx`：重做路由、密钥和提现弹窗，并修复小数提现前端校验。
- `apps/web/src/styles.css`：补充路由选择器、缓存阈值、提现明细和移动端布局样式。
- `apps/web/src/views/AdminView.jsx`：进入管理后台及提现动作后刷新待办统计。
- `apps/web/src/views/ConsoleView.jsx`：拆分路由管理与 API 密钥页面并修复商户申请模型目录引用。
- `apps/web/src/views/MarketView.jsx`：增加最近 7 天缓存命中率列。
- `apps/web/src/views/MerchantView.jsx`：显示商户名称、上架模型缓存指标及提现门槛和明细。
- `apps/web/src/views/UsagePanels.jsx`：隐藏用户、商户和管理员调用记录的请求编号列。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：增加提现状态筛选、刷新、分页、税费和实际打款列。
- `docs/V2数据基线.md`：记录独立路由、缓存快照和支付宝整数提现基线。
- `docs/模型报价路由与健康.md`：同步两级加权路由、最低缓存阈值和七天缓存指标。
- `docs/结算与人工提现.md`：同步 50 元门槛、支付宝、3% 税费及冻结释放规则。
- `docs/统一调用记录.md`：明确请求编号仅在服务端保留，三个前端主表隐藏。
- `docs/账号与管理员接口.md`：同步独立路由、密钥绑定和管理员提现分页。
- `progress.md`：追加本轮实现、验证、逐文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可整体回退独立路由、缓存指标、商户名称、支付宝提现、前端界面、文档和截图；本轮未上传服务器。

## 2026-07-23 - Task: 刷新本地交互预览夹具

### What was done

- 将 `57723` 本地预览的模拟接口同步为独立路由、路由商户分页、密钥绑定、缓存指标和支付宝提现当前结构。
- 将本地商户可提现余额调整到可操作范围，使商户提现与管理员审核界面可直接试用；该夹具位于 Git 忽略目录，不包含真实密钥。

### Testing

- `Invoke-WebRequest http://127.0.0.1:57723/`：返回 `200` 且页面包含 `ModelRoute`。
- Playwright 实际点击验证：路由列表、固定商户选择、密钥绑定、商户 `¥100 / ¥3 / ¥97` 提现明细和管理员提现列表均正常显示，业务接口无 `4xx/5xx`。
- 端口检查：仅一个 Vite 进程监听 `127.0.0.1:57723`。

### Notes

- `data/ui-preview/vite.config.mjs`：同步当前前端所需的本地模拟数据与接口；该运行夹具被 `.gitignore` 排除，不进入提交。
- `progress.md`：追加本地预览验证与运行边界记录。
- 回滚方式：停止 `57723` 预览进程并删除被忽略的 `data/ui-preview/vite.config.mjs`；该操作不影响 Git 跟踪代码和已完成提交。

## 2026-07-24 - Task: 动态读取核心额度单位并统一人民币口径

### What was done

- 平台启动时从核心状态读取实际 `quota_per_unit`，以“核心页面余额 1.00 对应平台 ¥1”作为唯一对外口径，移除充值、提现、兑换码和页面展示中写死的 `500000`。
- 充值订单继续使用人民币分和核心整数额度记账；提现门槛、人民币整数校验、3% 税费及实际到账金额统一使用启动时锁定的额度单位计算。
- 浏览器从平台版本接口获取额度单位并集中格式化金额，支付返回页、商户余额、管理员额度调整和兑换码额度均复用同一换算逻辑。
- V2 开发迁移取消数据库中写死的提现整数倍约束，金额合法性由持有运行时额度单位的事务入口校验。

### Testing

- `go test ./...`：通过，全部 Go 包测试无失败。
- `go vet ./...`：通过，无静态检查问题。
- `go build ./...`：通过，API、Worker 和模型种子命令均可构建。
- `npm.cmd run test:quota`：通过，前端运行时额度单位换算专项测试通过。
- `npm.cmd run test:ui`：通过，前端构建及 UI 冒烟测试通过。
- `npm.cmd run build`：通过，生产前端构建成功。
- `docker compose config --quiet`：通过；未注入本地生产环境变量时仅输出变量缺失提示。

### Notes

- `apps/api/internal/bootstrap/resources.go`：启动资源初始化时读取并注入核心额度单位。
- `apps/api/internal/bootstrap/resources_test.go`：覆盖额度单位初始化及配置传递。
- `apps/api/internal/httpapi/auth_test.go`：测试服务夹具注入有效额度单位。
- `apps/api/internal/httpapi/payment_domain.go`：充值额度按运行时单位和人民币分进行整数换算。
- `apps/api/internal/httpapi/payment_domain_test.go`：覆盖非默认单位、溢出及不可精确表示人民币分的单位。
- `apps/api/internal/httpapi/payment_handlers.go`：创建充值订单时使用服务启动锁定的额度单位。
- `apps/api/internal/httpapi/quota_unit.go`：新增核心状态额度单位读取与合法性校验。
- `apps/api/internal/httpapi/quota_unit_test.go`：覆盖状态读取、非默认单位和非法单位。
- `apps/api/internal/httpapi/server.go`：版本接口向浏览器发布当前额度单位。
- `apps/api/internal/httpapi/server_test.go`：覆盖版本接口额度单位响应。
- `apps/api/internal/httpapi/services.go`：服务依赖保存启动时锁定的额度单位。
- `apps/api/internal/httpapi/settlement_domain.go`：提现金额、门槛和税费拆分使用运行时单位。
- `apps/api/internal/httpapi/settlement_domain_test.go`：覆盖默认与非默认额度单位的提现边界。
- `apps/api/internal/httpapi/settlement_handlers.go`：商户提现入口传递当前额度单位。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：更新提现处理测试夹具。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：更新 PostgreSQL 提现事务测试输入。
- `apps/api/internal/httpapi/settlement_types.go`：提现事务命令携带额度单位。
- `apps/api/internal/httpapi/settlement_withdrawals.go`：事务层按当前单位校验整数提现。
- `apps/api/internal/migrations/migrate_test.go`：覆盖动态单位迁移约束。
- `apps/api/internal/migrations/sql/000007_dynamic_quota_unit.sql`：移除写死额度整数倍的数据库约束并保留正数约束。
- `apps/web/package.json`：增加前端额度单位专项测试命令。
- `apps/web/scripts/capture.mjs`：截图夹具返回非默认额度单位以验证动态展示。
- `apps/web/scripts/quota-unit-test.mjs`：新增前端动态换算专项测试。
- `apps/web/scripts/ui-smoke.mjs`：UI 冒烟夹具和金额断言改用运行时单位。
- `apps/web/src/Modals.jsx`：额度调整、充值和提现弹窗统一按运行时单位转换。
- `apps/web/src/api.js`：集中加载、保存、读取和格式化运行时额度单位。
- `apps/web/src/main.jsx`：应用启动阶段先初始化额度单位。
- `apps/web/src/styles.css`：增加额度单位初始化失败提示样式。
- `apps/web/src/views/MerchantView.jsx`：商户余额及提现输入使用动态单位。
- `apps/web/src/views/PaymentReturnView.jsx`：支付返回页使用统一额度格式化。
- `apps/web/src/views/admin/AdminRedemptions.jsx`：兑换码人民币额度使用动态单位换算。
- `docs/V2数据基线.md`：同步动态额度单位的数据口径。
- `docs/易支付充值.md`：同步充值金额与核心整数额度换算规则。
- `docs/模型报价路由与健康.md`：补充路由文档中的当前额度展示口径。
- `docs/结算与人工提现.md`：同步提现门槛、税费与动态单位规则。
- `docs/账号与管理员接口.md`：同步用户余额和管理员额度调整口径。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可整体回退动态额度单位改造；本轮未发布服务器。

## 2026-07-24 - Task: 删除固定路由并改为单一智能路由

### What was done

- 删除固定路由模式、固定商户名单、商户选择接口和相关前端交互，路由只保留名称、优先方式、倍率区间、最低缓存阈值及密钥绑定。
- 将失败切换改为按商户分轮排列候选，每轮每个商户最多尝试一条渠道；文本请求在首字节前遍历合格渠道，图片请求仅在明确可重试响应时切换。
- 缓存样本不足的新商户按中性分参与智能路由；只有样本充足且低于用户阈值时才退出候选。
- V2 最终路由迁移不再创建模式字段、固定商户关联表和路由决策模式字段，旧字段及旧接口由回归测试确认不再接受。

### Testing

- `go test ./... -count=1`：通过，全部 Go 包测试无失败。
- `go vet ./...`：通过，无静态检查问题。
- `go build ./cmd/api ./cmd/worker`：通过，API 与 Worker 可构建。
- 临时 PostgreSQL 17 执行 `go test ./internal/httpapi -run TestSQLRoutingStoreAgainstPostgres -count=1 -v`：通过，最终迁移、路由创建、候选解析和决策写入均完成真实数据库验证；临时容器已停止。
- `npm.cmd run test:ui`：通过，前端生产构建和单一智能路由交互冒烟测试通过。
- `npm.cmd run build`：通过，生产前端构建成功。
- `docker compose --env-file .env.example config --quiet`：通过，Compose 配置有效。
- `git diff --check`：通过，无空白错误。

### Notes

- `apps/api/internal/httpapi/routing_domain.go`：调整缓存阈值口径并按商户分轮生成候选顺序。
- `apps/api/internal/httpapi/routing_domain_test.go`：覆盖商户分轮和缓存低样本中性参与。
- `apps/api/internal/httpapi/routing_handlers.go`：删除固定路由字段校验和商户选择接口。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖单一路由请求及旧字段、旧接口拒绝行为。
- `apps/api/internal/httpapi/routing_proxy.go`：升级算法版本并限制图片连接结果不明时的重试。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖文本遍历和图片重试边界。
- `apps/api/internal/httpapi/routing_store.go`：移除固定商户查询、写入和过滤逻辑。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：使用单一路由结构验证 PostgreSQL 查询与决策写入。
- `apps/api/internal/httpapi/routing_store_test.go`：更新路由存储夹具和断言。
- `apps/api/internal/httpapi/server.go`：取消注册路由商户选择接口。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：同步最终路由表清理清单。
- `apps/api/internal/migrations/migrate_test.go`：锁定最终迁移不含固定路由字段和关联表。
- `apps/api/internal/migrations/sql/000005_independent_routes.sql`：移除模式、固定商户表和决策模式字段。
- `apps/web/scripts/capture.mjs`：更新单一路由截图场景。
- `apps/web/scripts/ui-smoke.mjs`：覆盖旧字段不再提交和固定路由界面已删除。
- `apps/web/src/Modals.jsx`：删除路由模式与固定商户选择器。
- `apps/web/src/styles.css`：清理固定商户选择器遗留样式。
- `apps/web/src/views/ConsoleView.jsx`：移除路由商户列表加载逻辑。
- `docs/V2数据基线.md`：记录不含固定路由字段和关联表的最终基线。
- `docs/模型报价路由与健康.md`：同步单一智能路由、缓存低样本和分轮重试规则。
- `docs/账号与管理员接口.md`：删除旧路由商户接口说明。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可整体恢复固定路由实现；本轮未发布服务器。

## 2026-07-24 - Task: 删除平台模型协议配置并复用核心协议处理

### What was done

- 标准模型目录删除接口协议和无实际用途的模型适配提示，只保留模型名称、厂商、状态与价格版本；文本或图片类型由价格结构唯一确定。
- 文本模型统一允许 Chat Completions 与 Responses 入口进入同一条智能路由，平台只替换模型别名并保留原始请求入口，具体上游转发继续交给核心服务。
- 模型市场删除协议筛选和协议列，商户渠道、上架模型、管理员渠道审核及模型管理删除协议和内部适配器展示。
- 管理员创建模型改为选择文本 Token 或图片按张的计费类型；检测接口只接收价格类型，创建接口不再提交协议字段。
- 模型种子、本地预览夹具和商户部署验收脚本同步使用价格类型，不再读取或生成模型协议字段。

### Testing

- `go test ./... -count=1`：通过，全部 Go 包测试无失败。
- `go vet ./...`：通过，无静态检查问题。
- `go build ./cmd/api ./cmd/worker`：通过，API 与 Worker 可构建。
- 全新临时 PostgreSQL 17 执行 `go test ./internal/httpapi -run TestSQLRoutingStoreAgainstPostgres -count=1 -v`：通过，V2 空库迁移、模型创建、报价与路由查询完成真实数据库验证；临时容器已删除。
- `npm.cmd run test:quota`：通过，动态额度单位专项测试正常。
- `npm.cmd run test:ui`：通过，创建与检测请求不含协议字段，模型市场、商户渠道和管理员模型表均无协议列。
- `npm.cmd run build`：通过，生产前端构建成功。
- `npm.cmd run capture`：通过，桌面与移动端截图无横向溢出；人工查看模型市场、商户上架模型、管理员模型和移动渠道页均无重叠。
- `python -m unittest discover -s deploy/tests -p 'test_*.py'`：通过，共 `27` 个部署测试。
- `docker compose --env-file .env.example config --quiet`：通过，Compose 配置有效。
- `git diff --check`：通过，无空白错误。

### Notes

- `apps/api/internal/httpapi/absolute_pricing.go`：按价格内容推导文本或图片类型并删除协议参数。
- `apps/api/internal/httpapi/absolute_pricing_test.go`：覆盖无协议文本和图片价格校验。
- `apps/api/internal/httpapi/channel_detector.go`：渠道检测能力改用价格类型。
- `apps/api/internal/httpapi/channel_detector_test.go`：更新价格类型检测夹具。
- `apps/api/internal/httpapi/channel_models.go`：上游模型匹配响应删除协议字段。
- `apps/api/internal/httpapi/health_runner.go`：恢复探测按价格类型选择请求入口。
- `apps/api/internal/httpapi/health_runner_test.go`：更新价格类型探测断言。
- `apps/api/internal/httpapi/merchant_handlers.go`：管理员复检使用价格类型构造模型。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：更新商户检测和审批夹具。
- `apps/api/internal/httpapi/merchant_store.go`：商户渠道查询和检测存储改为价格类型。
- `apps/api/internal/httpapi/merchant_store_test.go`：更新商户存储 SQL 断言。
- `apps/api/internal/httpapi/merchant_types.go`：商户审核内部字段由协议调整为价格类型。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：更新真实数据库模型插入结构。
- `apps/api/internal/httpapi/pricing_sync.go`：报价表达式生成按价格类型分支。
- `apps/api/internal/httpapi/pricing_sync_test.go`：更新文本和图片同步测试。
- `apps/api/internal/httpapi/routing_domain.go`：路由选择按价格类型应用缓存边界。
- `apps/api/internal/httpapi/routing_domain_test.go`：更新图片路由选择夹具。
- `apps/api/internal/httpapi/routing_handlers.go`：模型创建和检测接口删除协议输入。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖模型接口不返回协议字段及检测价格类型。
- `apps/api/internal/httpapi/routing_proxy.go`：文本模型共用 Chat 与 Responses 入口，图片模型保持图片入口。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖同一文本路由的两个入口和图片入口保护。
- `apps/api/internal/httpapi/routing_store.go`：标准模型、市场、报价和路由 SQL 删除协议与模型适配提示。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：用文本价格类型完成空库路由集成验证。
- `apps/api/internal/httpapi/routing_store_test.go`：更新模型创建与查询 SQL 夹具。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：更新历史调用关联的模型结构。
- `apps/api/internal/migrations/migrate_test.go`：锁定 V2 基线不再包含模型协议和适配提示列。
- `apps/api/internal/migrations/sql/000001_v2_baseline.sql`：删除标准模型协议和适配提示列。
- `apps/api/internal/modelseed/apply.go`：种子导入按厂商和价格类型比对。
- `apps/api/internal/modelseed/apply_test.go`：更新种子导入测试夹具。
- `apps/api/internal/modelseed/catalog.go`：种子模型结构删除协议字段。
- `apps/api/internal/modelseed/catalog_test.go`：覆盖无协议种子和价格类型推导。
- `apps/api/internal/modelseed/data/2026-07-23-v2.json`：删除全部模型协议配置。
- `apps/web/scripts/capture.mjs`：截图夹具删除模型协议字段。
- `apps/web/scripts/ui-smoke.mjs`：增加协议列、筛选和请求字段删除断言。
- `apps/web/src/Modals.jsx`：渠道检测结果不再显示内部适配器。
- `apps/web/src/views/MarketView.jsx`：删除协议筛选和协议列。
- `apps/web/src/views/MerchantView.jsx`：删除渠道协议列和上架模型协议副标题。
- `apps/web/src/views/admin/AdminChannels.jsx`：管理员渠道列表不再显示内部适配器。
- `apps/web/src/views/admin/AdminMerchants.jsx`：商户审核列表删除检测适配器列。
- `apps/web/src/views/admin/AdminModels.jsx`：模型表单改为计费类型并删除协议列与协议请求字段。
- `apps/web/artifacts/admin-announcements-desktop.png`：刷新当前管理员公告桌面验收图。
- `apps/web/artifacts/admin-channels-desktop.png`：刷新无内部适配器的管理员渠道桌面验收图。
- `apps/web/artifacts/admin-channels-mobile.png`：刷新无内部适配器的管理员渠道移动验收图。
- `apps/web/artifacts/admin-desktop.png`：刷新当前管理后台桌面验收图。
- `apps/web/artifacts/admin-image-price-desktop.png`：刷新图片计费类型桌面验收图。
- `apps/web/artifacts/admin-mobile.png`：刷新当前管理后台移动验收图。
- `apps/web/artifacts/admin-model-price-desktop.png`：刷新文本计费类型桌面验收图。
- `apps/web/artifacts/admin-models-desktop.png`：刷新无协议列的管理员模型桌面验收图。
- `apps/web/artifacts/admin-models-mobile.png`：刷新无协议列的管理员模型移动验收图。
- `apps/web/artifacts/admin-redemptions-desktop.png`：刷新当前兑换码管理桌面验收图。
- `apps/web/artifacts/admin-settings-desktop.png`：刷新当前平台设置桌面验收图。
- `apps/web/artifacts/admin-settings-mobile.png`：刷新当前平台设置移动验收图。
- `apps/web/artifacts/admin-usage-desktop.png`：刷新当前管理员调用记录桌面验收图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：刷新当前提现审核桌面验收图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：刷新当前提现审核移动验收图。
- `apps/web/artifacts/announcement-panel-desktop.png`：刷新当前公告面板桌面验收图。
- `apps/web/artifacts/announcement-panel-mobile.png`：刷新当前公告面板移动验收图。
- `apps/web/artifacts/console-desktop.png`：刷新当前控制台桌面验收图。
- `apps/web/artifacts/console-keys-desktop.png`：刷新当前密钥页面桌面验收图。
- `apps/web/artifacts/console-mobile.png`：刷新当前控制台移动验收图。
- `apps/web/artifacts/console-settings-desktop.png`：刷新当前账户设置桌面验收图。
- `apps/web/artifacts/console-usage-desktop.png`：刷新当前用户调用记录桌面验收图。
- `apps/web/artifacts/market-desktop.png`：刷新无协议筛选和协议列的市场桌面验收图。
- `apps/web/artifacts/market-mobile.png`：刷新无协议筛选和协议列的市场移动验收图。
- `apps/web/artifacts/merchant-channel-health-desktop.png`：刷新当前渠道健康详情桌面验收图。
- `apps/web/artifacts/merchant-desktop.png`：刷新当前商户中心桌面验收图。
- `apps/web/artifacts/merchant-mobile.png`：刷新当前商户中心移动验收图。
- `apps/web/artifacts/merchant-offers-desktop.png`：刷新无协议副标题的上架模型桌面验收图。
- `apps/web/artifacts/merchant-offers-mobile.png`：刷新无协议副标题的上架模型移动验收图。
- `apps/web/artifacts/merchant-usage-desktop.png`：刷新当前商户调用记录桌面验收图。
- `apps/web/artifacts/payment-config-desktop.png`：刷新当前支付设置桌面验收图。
- `apps/web/artifacts/route-modal-desktop.png`：刷新当前智能路由桌面验收图。
- `apps/web/artifacts/route-modal-mobile.png`：刷新当前智能路由移动验收图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：刷新当前提现弹窗桌面验收图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：刷新当前提现弹窗移动验收图。
- `deploy/scripts/verify-merchants.sh`：商户验收按价格类型选择文本模型。
- `docs/V2数据基线.md`：记录标准模型不保存协议和适配提示。
- `docs/商户与渠道审核接口.md`：明确内部适配结果只供服务端使用。
- `docs/官方模型种子.md`：将协议目录改为文本或图片价格类型目录。
- `docs/模型报价路由与健康.md`：记录入口透传和核心协议处理边界。
- `docs/绝对价格与分层计费.md`：同步无协议模型目录的计费口径。
- `docs/账号与管理员接口.md`：删除模型市场协议筛选说明。
- `data/ui-preview/vite.config.mjs`：本地预览夹具删除模型协议和模型适配字段；该文件位于 Git 忽略目录。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可整体恢复平台模型协议字段、界面和验收脚本；本轮未发布服务器。本地预览夹具可重新从当前种子生成。

## 2026-07-24 - Task: 校准本地预览路由数据

### What was done

- 本地预览路由数据删除已废弃的固定路由模式、商户名单和商户选择接口，与当前单一智能路由接口保持一致。

### Testing

- 请求 `GET /api/routing-policies`：通过，返回数据不含 `mode` 和 `merchant_ids`。
- 请求 `GET /api/routing-merchants`：通过，旧接口返回 `404`。
- Playwright 直接访问 `http://127.0.0.1:57723/`：通过，模型市场无协议控件，路由页无固定路由和商户选择，无横向溢出或页面运行错误。

### Notes

- `data/ui-preview/vite.config.mjs`：清理旧固定路由预览数据和接口；该文件位于 Git 忽略目录，仅用于当前本地预览。
- `progress.md`：追加本地预览校准和浏览器验收证据。
- 回滚方式：将 `data/ui-preview/vite.config.mjs` 的路由样例重置为当前提交的预览配置；本轮未改动服务器。

## 2026-07-24 - Task: 补齐最低缓存命中率的样本过滤规则

### What was done

- 修正智能路由的最低缓存命中率筛选：启用阈值后，文本模型商户必须同时满足样本充足且命中率达标；样本不足和低于阈值均退出候选。
- 图片模型继续忽略缓存阈值；未启用阈值时，样本不足的商户仍按中性分参与普通加权路由。
- 路由弹窗明确提示样本不足会被阈值过滤，与服务端行为保持一致。

### Testing

- `go test ./internal/httpapi -run TestSmartCacheFloorRejectsInsufficientSample -count=1`：修正前稳定失败，错误为样本不足候选仍返回可用。
- `go test ./internal/httpapi -run 'TestSmartCacheFloorRejectsInsufficientSample|TestCachePriorityTreatsLowSampleAsNeutral' -count=1`：通过，阈值过滤与未开启阈值时的中性权重均正确。
- `npm.cmd run test:ui`：通过，路由弹窗显示新筛选口径，前端生产构建和交互冒烟无失败。

### Notes

- `apps/api/internal/httpapi/routing_domain.go`：最低缓存阈值启用时增加商户样本就绪条件。
- `apps/api/internal/httpapi/routing_domain_test.go`：将样本不足用例改为应被阈值过滤。
- `apps/web/src/Modals.jsx`：更新最低缓存命中率的界面说明。
- `apps/web/scripts/ui-smoke.mjs`：增加路由阈值规则文案回归。
- `docs/模型报价路由与健康.md`：同步样本不足退出候选的业务口径。
- `progress.md`：追加本轮实现、红绿验证和回滚信息。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复样本不足候选按中性分参与的旧行为；本轮不涉及数据库结构，未发布服务器。

## 2026-07-24 - Task: 支持管理员配置提现税率

### What was done

- 管理员“支付与充值”增加提现税率设置，允许输入 `0.00%..100.00%` 且最多两位小数；服务端使用整数基点保存并随支付配置生成新版本。
- 商户提交提现时读取当前配置并把税率锁定到申请记录，历史申请不随管理员后续调税变化；税费和实际到账继续由数据库整数生成列计算。
- 商户提现弹窗动态读取并展示当前税率、税费和预计到账，配置读取完成前禁止提交；商户和管理员提现列表删除固定 `3%` 文案。
- 修正上一轮遗留的 PostgreSQL 测试夹具：删除已废弃路由 `mode` 字段，并消除 Windows 与 Docker 时钟偏差造成的支付任务领取波动。
- 刷新支付配置、商户提现和管理员提现审核的桌面与移动端验收图；本地 `57723` 预览同步使用可配置税率。

### Testing

- `npm.cmd run test:ui` 红灯一：修改测试为 `7.50%` 后稳定失败，提现弹窗仍显示固定 `¥3.00`；实现动态税率后通过。
- `npm.cmd run test:ui` 红灯二：增加固定 `税费（3%）` 禁止出现断言后稳定失败；清理商户与管理员固定税率文案后通过。
- `go test ./... -count=1`：通过，全部 Go 单元与契约测试无失败。
- `go vet ./...`、`go build ./...`：通过。
- 临时 PostgreSQL 17 执行 `go test ./internal/httpapi -count=1`：通过，8 份迁移、支付并发、提现冻结与税率锁定等完整数据库集成测试无失败；临时容器已关闭。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`、`npm.cmd run build`：通过。
- `npm.cmd run capture`：通过，桌面和移动端页面无根级横向溢出；人工检查税率输入、提现摘要和管理员审核表无文字重叠。
- `python -m unittest discover -s deploy/tests -p 'test_*.py'`：27 项通过。
- `docker compose config --quiet`：通过；未加载本地生产环境变量时仅输出预期默认空值警告。
- Playwright 直接访问 `http://127.0.0.1:57723/`：通过，当前税率显示 `3.00%`，编辑框值为 `3`，页面宽度与滚动宽度均为 `1440`，运行错误为 `0`。
- `git diff --check`：通过。

### Notes

- `apps/api/internal/httpapi/payment_handlers.go`：支付配置读写接口增加提现税率并校验整数基点范围。
- `apps/api/internal/httpapi/payment_handlers_test.go`：覆盖税率保存、响应和越界拒绝。
- `apps/api/internal/httpapi/payment_store.go`：支付配置仓储增加税率列的写入、查询和扫描。
- `apps/api/internal/httpapi/payment_store_test.go`：同步支付配置 SQL 与税率扫描契约。
- `apps/api/internal/httpapi/payment_store_postgres_test.go`：使用确定性未来领取时间并增强失败诊断，消除跨环境时钟波动。
- `apps/api/internal/httpapi/payment_types.go`：支付配置及创建命令增加税率基点字段。
- `apps/api/internal/httpapi/settlement_domain.go`：提现拆分改为使用运行时税率基点。
- `apps/api/internal/httpapi/settlement_domain_test.go`：覆盖 `7.50%` 拆分和税率范围边界。
- `apps/api/internal/httpapi/settlement_handlers.go`：提现申请读取当前支付配置并传递锁定税率。
- `apps/api/internal/httpapi/settlement_handlers_test.go`：覆盖当前税率读取、锁定和响应金额。
- `apps/api/internal/httpapi/settlement_store_postgres_test.go`：真实提现事务显式使用锁定税率。
- `apps/api/internal/httpapi/settlement_types.go`：提现记录和创建命令增加锁定税率字段。
- `apps/api/internal/httpapi/settlement_withdrawals.go`：提现仓储写入、查询并返回申请税率。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：移除已废弃路由模式字段，恢复当前数据库基线的完整集成测试。
- `apps/api/internal/httpapi/routing_store_test.go`：同步最低缓存阈值对样本不足候选的既定过滤契约。
- `apps/api/internal/migrations/migrate_test.go`：锁定第 8 份迁移及税率列、约束和生成公式。
- `apps/api/internal/migrations/sql/000008_withdrawal_tax_setting.sql`：增加支付配置和提现申请税率，并按记录税率生成税费与实际到账。
- `apps/web/src/Modals.jsx`：支付配置增加税率输入，提现弹窗动态读取并计算当前税率。
- `apps/web/src/views/MerchantView.jsx`：商户提现列表删除固定税率表头。
- `apps/web/src/views/admin/AdminPayment.jsx`：当前支付配置展示提现税率。
- `apps/web/src/views/admin/AdminWithdrawals.jsx`：管理员审核说明与表头改为申请时锁定税率口径。
- `apps/web/scripts/ui-smoke.mjs`：覆盖税率加载、提交、动态金额和固定文案清理。
- `apps/web/scripts/capture.mjs`：截图夹具增加公共与管理员税率配置。
- `apps/web/artifacts/payment-config-desktop.png`：更新管理员税率输入桌面验收图。
- `apps/web/artifacts/payment-config-mobile.png`：更新管理员税率输入移动验收图。
- `apps/web/artifacts/withdrawal-modal-desktop.png`：更新动态税费桌面验收图。
- `apps/web/artifacts/withdrawal-modal-mobile.png`：更新动态税费移动验收图。
- `apps/web/artifacts/admin-withdrawals-desktop.png`：更新申请时锁定税率的管理员桌面验收图。
- `apps/web/artifacts/admin-withdrawals-mobile.png`：更新申请时锁定税率的管理员移动验收图。
- `docs/V2数据基线.md`：将固定税率基线改为申请时锁定税率。
- `docs/易支付充值.md`：记录税率配置范围、整数基点和配置版本边界。
- `docs/结算与人工提现.md`：记录税率锁定、整数公式和历史申请不重算规则。
- `data/ui-preview/vite.config.mjs`：本地预览接口增加可编辑税率及动态提现金额；该文件位于 Git 忽略目录。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可恢复固定 `3%` 的代码、界面与测试；开发库需要精确回退时，切换到上一提交后按现有 V2 重置流程重建。本轮未发布服务器。

## 2026-07-24 - Task: 拆分商户申请并增加厂商渠道与路由范围预览

### What was done

- 商户申请缩减为商户名称、商户介绍和网站地址，管理员可从待审核状态直接通过或驳回；商户通过后再独立提交渠道。
- 新增商户资料读取与介绍修改接口，商户中心增加“商户设置”，名称和网站只读、介绍可编辑。
- 商户渠道对外统一使用六厂商 `provider`，页面不展示协议或内部适配器；模型同步按厂商隔离，Anthropic 与 Gemini 支持原生适配失败后回退兼容适配。
- 路由创建和编辑增加按模型实时范围预览，统计与真实路由共用模型、报价、渠道、商户、健康、倍率和缓存阈值条件。
- 公网模型请求外围只负责读取公开模型名、选择渠道和替换内部别名，请求协议与参数校验继续交给核心服务；`/v1/models` 继续按密钥路由返回公开模型。
- 增加连续迁移保存商户介绍和渠道厂商；按开发阶段约定，厂商迁移直接清理旧渠道及其级联开发数据，不保留旧渠道字段兼容。
- 校准 Git 忽略的 `57723` 本地预览夹具，使路由预览、六厂商渠道、直接商户审批和商户设置可以实际点击验收；未发布服务器。

### Testing

- `go test ./internal/migrations -run TestEmbeddedMigrationIsV2Baseline -count=1` 红灯：先稳定复现重复 `000008` 导致迁移编号不连续；重编号为 `000009/000010` 后通过。
- `npm.cmd run test:ui` 红灯：先稳定复现路由弹窗缺少“预览模型”；实现页面、接口桩和交互后通过。
- `go test ./... -count=1`：通过，全部 Go 单元与契约测试无失败。
- `go vet ./...`、`go build ./cmd/api`、`go build ./cmd/worker`：通过。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`、`npm.cmd run build`：通过；UI 烟雾覆盖三字段申请、管理员直接审批、六厂商渠道、商户设置、路由预览及桌面和移动布局。
- `docker compose config --quiet`：通过；仅输出本机未加载生产环境变量的预期空值警告。
- 直接请求 `http://127.0.0.1:57723/api/routing-policies/preview`、`/api/merchant/profile` 和 `/api/merchant/channels`：通过，分别返回 `1/1` 范围统计、官方商户资料和 OpenAI 厂商。
- Playwright 直接访问 `http://127.0.0.1:57723/`：通过，桌面与移动端页面级横向溢出均为 `0`，页面运行错误为 `0`，路由弹窗显示可用商户 `1`、可用渠道 `1`。
- 人工检查 `modelroute-route-preview-desktop.png` 与 `modelroute-merchant-settings-desktop.png`：统计块、表单、侧栏和操作区无文字重叠、截断或异常留白。
- `git diff --check`：通过。

### Notes

- `apps/api/internal/httpapi/channel_models.go`：增加六厂商规范化、厂商模型隔离和内部适配顺序。
- `apps/api/internal/httpapi/channel_models_handlers.go`：模型同步请求改用 `provider` 并限制正式商户调用。
- `apps/api/internal/httpapi/channel_models_test.go`：覆盖六厂商、跨厂商过滤和原生适配回退。
- `apps/api/internal/httpapi/merchant_handlers.go`：拆分商户申请与渠道提交，增加商户资料接口并删除旧单模型提交入口。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖三字段申请、直接审批、资料修改和商户权限。
- `apps/api/internal/httpapi/merchant_store.go`：持久化商户介绍、渠道厂商及直接申请审批事务。
- `apps/api/internal/httpapi/merchant_store_test.go`：同步商户资料与厂商渠道 SQL 契约。
- `apps/api/internal/httpapi/merchant_types.go`：公共商户与渠道结构增加介绍和厂商，隐藏内部适配字段。
- `apps/api/internal/httpapi/routing_domain.go`：删除外围请求协议与图片参数校验入口。
- `apps/api/internal/httpapi/routing_handlers.go`：增加路由范围预览接口和输入校验。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖预览鉴权、输入边界和响应。
- `apps/api/internal/httpapi/routing_proxy.go`：模型代理只保留公开模型替换、路由和核心响应透传。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖流式响应、核心错误和跨入口透传。
- `apps/api/internal/httpapi/routing_store.go`：实现与真实候选条件一致的商户和渠道去重统计。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖倍率、缓存、健康、下架和统计去重。
- `apps/api/internal/httpapi/server.go`：注册商户资料与路由预览路由，移除商户申请验证路由。
- `apps/api/internal/migrations/migrate_test.go`：锁定十份连续迁移、商户介绍、六厂商和开发数据清理约束。
- `apps/api/internal/migrations/sql/000009_merchant_profile_description.sql`：增加商户介绍字段。
- `apps/api/internal/migrations/sql/000010_channel_provider.sql`：清理旧渠道开发数据并增加六厂商约束。
- `apps/web/scripts/ui-smoke.mjs`：覆盖新申请、审批、厂商渠道、商户设置和路由预览的桌面及移动交互。
- `apps/web/src/Modals.jsx`：简化商户申请，渠道改选厂商，路由弹窗增加防抖范围预览。
- `apps/web/src/styles.css`：增加路由统计块、商户设置和长介绍表格样式。
- `apps/web/src/views/ConsoleView.jsx`：为路由弹窗提供模型目录并移除申请阶段渠道参数。
- `apps/web/src/views/MerchantView.jsx`：增加厂商列和商户设置页面，接入资料读取与保存。
- `apps/web/src/views/admin/AdminMerchants.jsx`：管理员商户审核改为资料核对后直接通过或驳回。
- `docs/V2数据基线.md`：记录六厂商渠道和实时路由预览数据边界。
- `docs/商户与渠道审核接口.md`：记录三字段申请、商户资料、厂商同步与审批接口。
- `docs/模型报价路由与健康.md`：记录核心协议处理边界、范围预览和四个公网入口。
- `data/ui-preview/vite.config.mjs`：校准本地预览的新接口和数据；该文件被 Git 忽略，仅服务当前 `57723` 验收。
- `progress.md`：追加本轮实现、验证、文件清单和回滚边界。
- 回滚方式：执行 `git revert <本轮提交哈希>` 恢复代码、迁移、页面、测试和文档；若本地开发库已经执行 `000010`，旧渠道数据已按约定清理，需切回上一提交后通过开发数据重置流程重建。本地预览夹具可恢复当前提交对应的预览配置。本轮未发布服务器。

## 2026-07-24 - Task: 修复渠道审核并改用模型详情弹窗

### What was done

- 修复管理员渠道“验证”与“通过”链路：验证按顺序检查全部已匹配模型，允许部分成功；通过时只导入检查成功的型号，并取消正式渠道创建后的重复测试。
- 文本模型继续发送 `hi`，图片模型使用图片检测入口；全部模型失败时审核恢复为待审核，审批和单模型检查返回脱敏后的实际原因。
- 平台启动时幂等开启唯一管理员的未定价模型测试权限，同时保留其原有通知、额度阈值和 IP 日志设置，不影响普通用户与正式计费。
- 管理后台和商户中心的渠道列表统一改为“查看模型（数量）”按钮；共用弹窗展示模型、匹配状态、检查状态、耗时和错误，管理员可单模型复检，商户只读。
- 管理员正式渠道接口补齐实际导入模型清单；`verified` 状态明确显示为“已验证”，批量验证提示成功和失败数量。

### Testing

- 后端目标红灯覆盖图片检测入口、部分成功、全部失败、只导入成功型号、无成功型号禁止通过、模型提升 SQL 过滤和管理员设置保留；实现后全部通过。
- `go test ./...`：通过，全部 Go 单元与契约测试无失败。
- `go vet ./...`、`go build ./cmd/api ./cmd/worker`：通过。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`、`npm.cmd run build`：通过；UI 烟雾覆盖模型名称默认隐藏、两类弹窗、单模型检查、状态文案和验证数量提示。
- `npm.cmd run capture`：通过，管理后台与商户中心的模型弹窗均生成桌面和移动端截图，根页面无横向溢出；人工检查无文字重叠或列表异常增高。
- `python -m unittest discover -s deploy/tests -p "test_*.py"`：27 项通过。
- `docker compose --env-file .env.example --profile tools config --quiet`、`git diff --check`：通过。

### Notes

- `apps/api/internal/bootstrap/resources.go`：管理员商户初始化时合并并幂等开启未定价模型测试权限。
- `apps/api/internal/bootstrap/resources_test.go`：覆盖设置保留和重复启动不重复写入。
- `apps/api/internal/httpapi/channel_detector.go`：图片模型检测增加图片入口参数。
- `apps/api/internal/httpapi/channel_detector_test.go`：覆盖图片检测入口。
- `apps/api/internal/httpapi/channel_models_handlers.go`：单模型检查失败返回脱敏后的真实原因。
- `apps/api/internal/httpapi/merchant_handlers.go`：实现批量验证、成功型号审批、正式渠道模型响应和错误脱敏。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：覆盖部分成功、全部失败、审批过滤、无成功型号阻止、错误脱敏和管理员正式渠道模型清单。
- `apps/api/internal/httpapi/merchant_store.go`：正式渠道只提升已匹配且检查成功的模型绑定。
- `apps/api/internal/httpapi/merchant_store_test.go`：锁定成功模型提升 SQL 条件。
- `apps/api/internal/newapi/client.go`：读取唯一管理员现有设置。
- `apps/web/src/ChannelModelsModal.jsx`：新增管理端与商户端共用的渠道模型弹窗。
- `apps/web/src/views/admin/AdminChannels.jsx`：模型列改为数量按钮，接入弹窗、复检、已验证文案和验证统计提示。
- `apps/web/src/views/MerchantView.jsx`：模型列改为数量按钮并接入只读弹窗。
- `apps/web/src/styles.css`：增加弹窗内部滚动和错误换行样式。
- `apps/web/scripts/ui-smoke.mjs`：增加渠道模型隐藏、弹窗、复检和状态回归测试。
- `apps/web/scripts/capture.mjs`：增加管理端与商户端模型弹窗桌面和移动截图。
- `apps/web/artifacts/admin-channels-desktop.png`、`apps/web/artifacts/admin-channels-mobile.png`：更新管理员渠道列表验收图。
- `apps/web/artifacts/admin-channel-models-desktop.png`、`apps/web/artifacts/admin-channel-models-mobile.png`：新增管理员模型弹窗验收图。
- `apps/web/artifacts/merchant-channel-models-desktop.png`、`apps/web/artifacts/merchant-channel-models-mobile.png`：新增商户模型弹窗验收图。
- `docs/商户与渠道审核接口.md`：同步批量验证、成功型号审批、弹窗和管理员内部测试设置。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：执行 `git revert <本轮提交哈希>` 可回退本轮代码、界面、测试、文档和验收图；本轮没有数据库迁移，不需要恢复或重置现有渠道数据。

## 2026-07-24 - Task: 修复批量渠道验证的网关超时

### What was done

- 生产候选版本实际验证 20 个模型时复现 Nginx 60 秒 `504`；确认旧同步请求会随浏览器连接取消并把审核记录遗留在 `verifying`。
- 批量验证改为立即返回 `202`，由平台后台在 15 分钟时限内继续顺序检查；完成后统一写回“已验证”或“待审核”，不再依赖浏览器和网关连接存活。
- 管理后台在收到“验证已开始”后轮询审核记录，完成时根据逐模型结果提示成功和失败数量；失败时展示已脱敏的实际原因。

### Testing

- `go test ./internal/httpapi -run 'TestAdminVerifiesEveryMatchedModelAndAllowsPartialSuccess|TestAdminVerificationRestoresPendingWhenEveryMatchedModelFails' -count=1`：通过；阻塞首个模型时 HTTP 请求仍立即返回，后台释放后按顺序完成，全部失败时恢复待审核。
- `go test ./...`、`go vet ./...`、`go build ./cmd/api ./cmd/worker`：通过。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`、`npm.cmd run capture`：通过；覆盖验证开始提示、状态轮询、成功与失败数量提示及“已验证”文案，桌面和移动端页面无横向溢出。
- `python -m unittest discover -s deploy/tests -p "test_*.py"`：27 项通过。
- `docker compose --env-file .env.example --profile tools config --quiet`、`git diff --check`：通过。
- 生产发布与 ID 1 重新验证证据将在本补丁发布后追加，不提前声明线上问题已消除。

### Notes

- `apps/api/internal/httpapi/merchant_handlers.go`：将批量模型验证改为有时限的后台任务，并使用独立短上下文写回最终状态。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：增加浏览器请求不等待长模型检测的回归用例并同步全部失败断言。
- `apps/web/src/views/admin/AdminChannels.jsx`：增加审核状态轮询和完成数量提示。
- `apps/web/scripts/ui-smoke.mjs`：模拟异步验证响应并验证轮询后的界面结果。
- `docs/商户与渠道审核接口.md`：记录批量验证的异步接口和状态边界。
- `progress.md`：追加生产 504 根因、修复、验证和回滚说明。
- 回滚点：前一提交为 `3cdf835dab83`；执行 `git revert <本轮补丁提交>` 可恢复同步验证行为。本轮不改数据库结构，生产中卡住的 ID 1 只需恢复到待审核后重新触发。

## 2026-07-24 - Task: 发布渠道审核修复到 103

### What was done

- 发布候选 `3cdf835dab83` 后在真实 ID 1 复现旧同步批量验证的 60 秒 `504`，据此完成异步补丁 `4ad2ea845834` 并重新发布，生产 `current` 已原子切换到该版本。
- 定点恢复 ID 1 的旧 `verifying` 遗留状态后重新触发验证；接口在 0 秒内返回 `202`，后台约 90 秒完成顺序检测，最终 8 个已匹配型号中 5 个通过、3 个失败。
- 真实调用“通过”接口成功创建正式渠道 ID 1；渠道状态为启用，只导入 5 个验证成功型号，审核记录进入 `approved`，待审密钥密文已清除。

### Testing

- 新发布目录内外网 `deploy/scripts/verify.sh` 均通过；线上 `/api/version` 返回 `4ad2ea845834`。
- 生产 `POST /api/admin/channel-reviews/1/verify` 返回 `202` 且耗时小于 1 秒，后台最终状态为 `verified`；`POST /api/admin/channel-reviews/1/approve` 返回 `200`。
- 数据核对结果为 `approved | 渠道 ID 1 | active | 5 个型号 | 待审密钥已清除`。
- 六个 ModelRoute 容器运行，五个带健康检查的容器均为 `healthy`；API 与 Worker 最近 10 分钟无 `panic/fatal`。
- `api.lwylink.xyz`、`sdk.lwylink.xyz` 均返回 `200`；Sub2API、PostgreSQL 和 Redis 三个既有容器保持健康。

### Notes

- `progress.md`：追加真实 504 复现、异步验证、审批、版本、容器和隔离验收证据。
- 生产发布目录：`/opt/modelroute-platform/releases/4ad2ea845834`，上一稳定指针为 `/opt/modelroute-platform/releases/e4d271672264`。
- 代码回滚可把 `current` 切回 `e4d271672264` 后执行其部署脚本；本轮没有数据库迁移。正式渠道 ID 1 是本次验收产生的业务数据，代码回滚不会自动删除，可在管理后台下架或删除。

## 2026-07-24 - Task: 改造单厂商密钥与可复用单模型路由

### What was done

- 路由创建时固定一个标准模型及其厂商，同一模型可创建多套不同优先策略；编辑只允许修改名称、倍率范围、优先方式和最低缓存阈值。
- API 密钥创建时固定六类厂商之一，可为同厂商多个模型分别选择一条路由，也可保持空绑定；跨厂商、跨用户和同模型重复绑定均由服务端拦截。
- 请求解析改为“密钥 -> 公开模型 -> 唯一路由 -> 商户与渠道候选”；无效或停用密钥返回 `401`，未绑定或跨厂商模型返回 `403`，已绑定但无候选返回 `503`。
- 删除路由会自动解除全部密钥绑定并返回受影响密钥数；停用密钥继续显示并允许配置或重新启用，删除密钥则删除平台绑定且保留历史调用记录。
- `/v1/models` 只返回当前密钥已绑定且存在可用候选的公开模型，去重并隐藏内部别名；控制台完成路由厂商/模型列、密钥厂商/模型数、模型路由配置和删除影响提示。
- 新增第 11 号迁移清理旧用户密钥、路由和绑定，保留用户、余额、商户、渠道、模型及历史调用；部署脚本只删除核心服务中对应的旧用户 Token，不触碰内部服务令牌。

### Testing

- 先用 `TestListTokenRoutingBindingsIncludesDisabledPlatformKeys`、`TestReplaceTokenRoutesAllowsDisabledPlatformKey` 和 `TestRevokeTokenBindingDeletesPlatformBinding` 稳定复现停用密钥消失、停用后不可配置及删除后残留绑定，修复后目标测试通过。
- `go test ./... -count=1`：通过，全部 Go 单元与契约测试无失败。
- `MODELROUTE_TEST_DATABASE_URL=<临时 PostgreSQL> go test ./internal/httpapi -run 'TestSQLRoutingStoreAgainstPostgres|TestSQLAdminOverviewStoreUsesSettledIntegerLedgerData' -count=1`：通过，真实数据库验证单厂商多模型绑定、删除路由自动解绑、停用/删除密钥边界和运营查询。
- `MODELROUTE_TEST_DATABASE_URL=<临时 PostgreSQL> go test ./internal/migrations -run TestProviderModelRouteMigrationKeepsHistoricalDecisions -count=1`：通过，第 10 号升级到第 11 号后历史调用记录保留且旧外键置空。
- `go vet ./...`、`go build ./cmd/api ./cmd/worker`：通过。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`、`npm.cmd run build`、`npm.cmd run capture`：通过；UI 烟雾覆盖单模型路由、单厂商空密钥、多模型路由配置、跨厂商隔离、删除自动解绑及桌面和移动布局。
- 人工检查 `route-modal-desktop.png`、`route-modal-mobile.png`、`console-keys-desktop.png` 和 `console-keys-mobile.png`：厂商、模型、路由规则、密钥厂商与模型数无文字重叠或横向溢出。
- `python -m unittest discover -s deploy/tests -p "test_*.py"`：27 项通过；路由发布验收已使用新契约，部署脚本旧用户 Token 清理顺序受测试锁定。
- `bash -n deploy/scripts/verify-routing.sh`、`docker compose --env-file .env.example --profile tools config --quiet`、`git diff --check`：通过；差异敏感信息扫描未发现上游密钥或支付凭据正文。

### Notes

- `apps/api/internal/httpapi/account_handlers.go`：密钥接口改为固定厂商和完整模型路由列表，并同步启停及删除绑定。
- `apps/api/internal/httpapi/overview_store_postgres_test.go`：更新真实运营查询夹具以适配模型路由绑定结构。
- `apps/api/internal/httpapi/routing_handlers.go`：区分创建与编辑字段，增加删除路由影响数量和多路由绑定处理。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖单模型路由、空密钥、多模型绑定、接口越权和启停同步。
- `apps/api/internal/httpapi/routing_proxy.go`：按密钥和请求模型解析唯一路由，细分 401、403、503 并过滤模型列表。
- `apps/api/internal/httpapi/routing_proxy_test.go`：覆盖错误码、可路由模型过滤、去重和协议透传回归。
- `apps/api/internal/httpapi/routing_store.go`：实现厂商密钥、模型路由关系、自动解绑、停用可管理和删除绑定。
- `apps/api/internal/httpapi/routing_store_postgres_test.go`：用真实 PostgreSQL 覆盖同厂商多模型、同模型唯一绑定、跨厂商拒绝及历史记录保留。
- `apps/api/internal/httpapi/routing_store_test.go`：增加仓储 SQL、停用密钥可见性、删除清理和受影响数量测试。
- `apps/api/internal/httpapi/server.go`：注册完整模型路由替换接口并移除旧单路由接口。
- `apps/api/internal/migrations/migrate_test.go`：锁定第 11 号迁移结构、厂商约束和历史调用保留。
- `apps/api/internal/migrations/sql/000011_provider_model_routes.sql`：建立单厂商密钥与单模型路由数据结构并清理旧绑定。
- `apps/web/artifacts/console-keys-desktop.png`：更新桌面密钥厂商与已配置模型验收图。
- `apps/web/artifacts/console-keys-mobile.png`：更新移动端密钥页面验收图。
- `apps/web/artifacts/route-modal-desktop.png`：更新桌面单模型路由编辑验收图。
- `apps/web/artifacts/route-modal-mobile.png`：更新移动端单模型路由编辑验收图。
- `apps/web/scripts/capture.mjs`：更新路由和密钥截图夹具为厂商及多模型绑定结构。
- `apps/web/scripts/ui-smoke.mjs`：覆盖新路由、密钥创建和模型路由配置交互。
- `apps/web/src/Modals.jsx`：路由弹窗增加厂商/模型选择，密钥弹窗增加厂商和按模型单选路由。
- `apps/web/src/styles.css`：增加密钥模型路由分组的紧凑样式。
- `apps/web/src/views/ConsoleView.jsx`：路由和密钥列表增加厂商、模型及配置操作，并允许删除已绑定路由。
- `deploy/scripts/deploy.sh`：第 11 号迁移前删除核心服务中对应的旧用户 Token。
- `deploy/scripts/verify-routing.sh`：改为验证单模型路由、单厂商密钥、模型授权、自动解绑和空绑定。
- `deploy/tests/test_deploy.py`：锁定旧用户 Token 定点清理命令和部署顺序。
- `deploy/tests/test_release_tools.py`：锁定新路由发布验收契约并排除旧固定路由字段。
- `docs/模型报价路由与健康.md`：记录单模型路由、厂商密钥、错误码、模型列表和迁移边界。
- `docs/账号与管理员接口.md`：记录密钥模型路由接口及停用、删除语义。
- `docs/部署运行手册.md`：更新路由验收说明和第 11 号迁移的用户 Token 清理顺序。
- `progress.md`：追加本轮实现、验证、文件清单和回滚点。
- 回滚点：本轮前一提交为 `cc6a365`；执行 `git revert <本轮提交哈希>` 可回退代码、页面、测试和文档。若数据库已执行第 11 号迁移，旧用户密钥和路由已按计划清理，代码回滚不会恢复它们；开发环境需切回 `cc6a365` 后按既有重置流程重建。本轮未发布服务器。

## 2026-07-24 - Task: 修复本地预览路由厂商与模型不显示

### What was done

- 确认正式前后端已使用单厂商密钥与单模型路由契约，页面空白来自被忽略的本地预览夹具仍返回旧版单路由字段。
- 同步本地预览夹具，使路由返回厂商、模型 ID 和模型名称，密钥返回厂商、已配置模型数量及完整模型路由列表。
- 补齐本地夹具的完整绑定替换、跨厂商校验、同模型重复校验和删除路由自动解绑行为。

### Testing

- `node data/ui-preview/verify.mjs`：通过，21 个模型可见，页面运行错误为 0，桌面宽度无横向溢出。
- 本地接口核对：路由 ID 71 返回 `OpenAI / gpt-5.6-sol`；密钥 ID 9 返回 OpenAI、1 个已配置模型并绑定路由 ID 71。
- 浏览器核对：路由列表的厂商、模型和绑定密钥列均正确显示；修复前截图中的空白列不再复现。

### Notes

- `data/ui-preview/vite.config.mjs`：同步本地预览接口到单厂商、多模型路由绑定契约；该目录按仓库规则忽略，不纳入正式发布提交。
- `data/ui-preview/verify.mjs`：增加路由厂商、模型和密钥模型绑定的本地页面断言；该目录按仓库规则忽略。
- `progress.md`：追加本地预览问题根因、验证证据和文件清单。
- 回滚方式：停止本地 57723 预览并删除被忽略的 `data/ui-preview/` 后重新生成旧夹具；正式代码与生产 103 均未变更。

## 2026-07-24 - Task: 修复单厂商密钥真实鉴权并完成发布前验证

### What was done

- 在生产候选环境确认核心服务保存的 Token 正文不带 `sk-`，而客户端请求使用带前缀的公开密钥；平台此前对内部正文生成指纹，导致路由和模型绑定虽已配置，真实 `/v1/models` 与模型请求仍匹配不到密钥。
- 平台创建密钥绑定时统一对公开密钥形态生成不可逆指纹；已有前缀保持不变，无前缀时只补一次 `sk-`。
- 增加按需复制接口：先校验 Token 属于当前登录用户，再从核心服务读取真实密钥并以公开形态返回；密钥列表继续只显示掩码，响应禁止缓存。
- 更新账号、商户和路由生产验收脚本，使其遵循单厂商密钥、三字段商户申请和模型路由契约；路由验收增加真实模型列表、实际调用和指纹匹配检查，并在结束时定点清理临时数据。

### Testing

- 先运行 `go test ./internal/httpapi -run TestRevealKeyReturnsOwnedTokenWithPublicPrefix -count=1`，稳定复现接口 `404` 且上游读取次数为 0；实现后该用例与无前缀/已有前缀指纹用例共同通过。
- `go test ./... -count=1`、`go vet ./...`、`go build ./cmd/api ./cmd/worker ./cmd/seed-models`：通过。
- `npm.cmd run test:quota`、`npm.cmd run test:ui`：通过；浏览器实际点击复制后，确认按需读取接口调用一次且剪贴板得到完整公开密钥。
- `python -m unittest discover -s deploy/tests -p "test_*.py" -v`：28 项通过。
- 全部 `deploy/scripts/*.sh` 执行 `bash -n`：通过；`docker compose -f compose.yaml config --quiet`：通过；`git diff --check`：通过。
- 生产发布和真实专项验收将在新提交部署后另行追加，不把候选容器运行等同于正式发布完成。

### Notes

- `apps/api/internal/httpapi/account_handlers.go`：增加归属校验、真实密钥按需读取和公开前缀规范化。
- `apps/api/internal/httpapi/account_test.go`：覆盖真实密钥返回、前缀补齐和跨用户读取拦截。
- `apps/api/internal/httpapi/routing_handlers.go`：路由绑定指纹改为使用公开密钥形态。
- `apps/api/internal/httpapi/routing_handlers_test.go`：覆盖核心密钥无前缀和已有前缀两种契约。
- `apps/api/internal/httpapi/server.go`：注册真实密钥按需复制接口。
- `apps/web/src/views/ConsoleView.jsx`：复制按钮改为按需读取真实密钥，不再复制列表掩码。
- `apps/web/scripts/ui-smoke.mjs`：增加真实密钥接口、剪贴板内容和调用次数回归检查。
- `deploy/scripts/verify-accounts.sh`：按单厂商空路由契约验证临时密钥。
- `deploy/scripts/verify-merchants.sh`：按三字段商户申请、直接审批和资料修改契约验收。
- `deploy/scripts/verify-routing.sh`：增加真实模型列表、模型调用、指纹诊断和精确清理。
- `deploy/tests/test_release_tools.py`：锁定新版账号、商户和路由验收脚本契约。
- `docs/账号与管理员接口.md`：记录真实密钥复制接口、归属校验和禁止缓存语义。
- `docs/模型报价路由与健康.md`：记录公开密钥形态与路由指纹一致性。
- `docs/部署运行手册.md`：记录路由生产验收与临时数据清理边界。
- `progress.md`：追加根因、改动、验证证据和回滚点。
- 回滚点：本轮前一提交为 `4897300`；执行 `git revert <本轮提交哈希>` 可回退代码、页面、验收脚本和文档。本轮没有数据库迁移；已执行的第 11 号迁移和用户重新创建的密钥不受本轮代码回滚影响。

## 2026-07-24 - Task: 移除过期结算验收并保留提现发布检查

### What was done

- 生产候选执行旧 `verify-settlement.sh` 时，已下线的 `/api/admin/settlement-reconciliation` 按产品约束返回 `404`；现有服务端回归测试也明确要求该人工结算与对账接口不再暴露。
- 将发布脚本改名为 `verify-withdrawals.sh`，保留管理员提现列表、商户余额、匿名权限、敏感字段和同源写入拦截检查，删除过期的人工对账请求。
- 同步部署测试和运维文档，不保留旧脚本名或旧接口兼容入口。

### Testing

- 修改测试后先运行提现验收与唯一管理员脚本契约，因 `verify-withdrawals.sh` 尚不存在而失败，确认测试锁定新交付物；完成改名和收敛后目标测试通过。
- 生产候选已独立通过支付验收、运营只读验收以及 200 请求、10 并发的健康入口压力测试；新提现脚本将在新提交发布后重新执行并记录结果。

### Notes

- `deploy/scripts/verify-withdrawals.sh`：由旧结算脚本改名并收敛为提现与余额只读验收。
- `deploy/tests/test_compose.py`：锁定提现验收必须排除已下线的人工对账接口。
- `deploy/tests/test_release_tools.py`：唯一管理员验收脚本清单改为提现脚本。
- `docs/结算与人工提现.md`：更新生产提现验收说明。
- `docs/部署运行手册.md`：发布步骤改为执行提现专项验收。
- `progress.md`：追加生产候选阻塞根因、改动、验证和回滚说明。
- 回滚点：本轮前一提交为 `15fc4c1`；执行 `git revert <本轮提交哈希>` 可恢复旧脚本，但旧人工对账接口仍按产品约束保持下线，因此恢复后该脚本会继续返回 `404`。本轮没有数据库迁移。

## 2026-07-24 - Task: 修复路由生产验收软删除残留

### What was done

- 正式切换前检查发现路由策略夹具已经清零，但两次路由验收创建的核心 Token 只完成软删除，数据库仍保留带 `deleted_at` 的测试行。
- 根因是成功路径在退出钩子执行前把 `KEY_ID` 置空，导致异常路径已有的精确物理删除没有在正常结束时运行。
- 路由验收正常结束时现在显式执行同一清理逻辑，并按 Token ID 与测试名称断言数据库剩余数量为零后才报告通过。

### Testing

- 先增加 `REMAINING_TOKEN_COUNT` 清零契约，目标测试因脚本缺少该断言而失败；补齐正常路径清理后目标测试通过。
- 生产库已确认残留两条均为 `provider-key-contract-*` 测试 Token 且 `deleted_at` 非空，不是可用用户密钥；新脚本发布后将定点清除并重新执行路由专项验收。

### Notes

- `deploy/scripts/verify-routing.sh`：正常结束时显式物理清理测试 Token 并断言无残留。
- `deploy/tests/test_release_tools.py`：锁定路由验收必须验证 Token 清理结果。
- `progress.md`：追加残留性质、根因、修复、验证和回滚说明。
- 回滚点：本轮前一提交为 `6e84908`；执行 `git revert <本轮提交哈希>` 可回退脚本清理断言。本轮没有数据库迁移或业务数据结构变更。

## 2026-07-24 - Task: 发布单厂商密钥与模型路由修复到 103

### What was done

- 将密钥公开形态指纹修复、真实密钥复制、单厂商生产契约、提现验收和测试 Token 清理补丁统一发布为 `5774c931c8ef`。
- 发布过程中使用新 Git 哈希目录重新构建，不复用手工修改过的旧候选目录；第 11 号迁移按幂等路径保持现有数据结构，模型种子确认 21 个型号无新增、无重复调价。
- 全部线上专项通过后，`current` 从 `4ad2ea845834` 原子切换到 `/opt/modelroute-platform/releases/5774c931c8ef`；内网和公网版本接口均返回同一哈希。
- 定点物理删除两条已软删除的旧路由验收 Token；新脚本再次执行后，路由策略和核心 Token 测试夹具均保持为零。

### Testing

- 内网与公网 `verify.sh`：通过，页面、存活、就绪和模型入口鉴权状态正常。
- `verify-accounts.sh`、`verify-merchants.sh`、`verify-routing.sh`、`verify-payments.sh`、`verify-withdrawals.sh`、`verify-operations.sh`：全部通过。
- 真实路由诊断确认 `fingerprint_matches_raw=false`、`fingerprint_matches_prefixed=true`；单厂商临时密钥成功返回已绑定的 `gpt-5.6-luna`，未绑定模型返回预期 `403`，清理后测试路由与 Token 数量均为 `0`。
- 只读压力测试：200 请求、10 并发全部成功，平均 `0.0018s`，P95 `0.004246s`。
- 生产数据库：第 11 号迁移存在，启用模型 21 个，MiniMax 0 个；模型种子重复执行结果为新增 0、调价 0、未变化 21。
- 六个 ModelRoute 容器运行，网关、API、核心服务、PostgreSQL 和 Redis 健康检查均为 `healthy`；API 与 Worker 最近窗口 `panic/fatal` 为 0，核心服务 `429` 为 0。
- `api.lwylink.xyz` 与 `sdk.lwylink.xyz` 均返回 `200`；PostgreSQL、Redis 和核心服务仍只暴露容器内部端口。

### Notes

- `progress.md`：追加最终发布哈希、专项验收、残留清理、容器与隔离证据。
- 生产发布目录：`/opt/modelroute-platform/releases/5774c931c8ef`。
- 回滚点：上一稳定指针为 `/opt/modelroute-platform/releases/4ad2ea845834`；可使用固定目标临时软链接后 `mv -Tf` 原子切回，并执行旧目录的基础验收。本轮数据库迁移已在此前候选执行且不会因代码指针回滚自动恢复旧密钥和路由。

## 2026-07-24 - Task: 整理项目代码与验收生成物

### What was done

- 使用 Go deadcode、staticcheck 和前端引用分析复核真实调用关系，删除两个生产不可达的服务构造/商户申请辅助函数，并将两个仅在文件内部使用的前端符号收回为私有。
- 将生产 UI 验收登记为明确的 npm 命令；删除 50 张可再生成的历史验收截图并忽略截图目录，同时清理本地 Go 二进制、Vite 构建目录、Python 缓存和旧 57723 预览夹具。
- 修复账号验收仅软删除测试 Token 的残留：脚本现在只针对自身生成的 `contract-*` 数字编号记录，清理平台绑定与核心 Token；异常退出与正常结束使用同一保护条件。
- 保留仍被真实业务使用的密钥复制、失败冲正、商户余额和提现内部链路；未因文件较大而拆分路由或商户存储，避免无行为依据的数据库重构。

### Testing

- 先为账号验收补物理清理契约，目标测试因脚本缺少两条定点删除语句而失败；完成脚本修改后目标测试通过。
- `go test ./...`、`go vet ./...`、`go test -race ./...`、Go deadcode 和 staticcheck 全部通过，未再报告不可达函数或静态问题。
- 前端动态额度测试、UI 冒烟测试、生产构建和 knip 引用分析通过；`npm audit --audit-level=high` 返回 0 个漏洞，直接依赖树检查通过。
- 部署工具 28 项 Python 测试全部通过；`docker compose --env-file .env.example --profile tools config --quiet` 与 `git diff --check` 通过。
- `npm ls --all` 仍报告 `@lobehub/icons` 深层 UI 依赖中的两个 React 16-18 peer 声明与 React 19 不一致；当前直接依赖、Vite 构建、UI 冒烟和安全审计均通过，本轮未混入厂商图标库大版本升级。

### Notes

- `.gitignore`：忽略可再生成的前端验收截图目录。
- `apps/api/internal/httpapi/server.go`：删除仅供旧测试调用的服务构造包装。
- `apps/api/internal/httpapi/server_test.go`：测试改为直接使用生产服务构造入口。
- `apps/api/internal/httpapi/merchant_handlers.go`：删除未注册、未引用的管理员申请辅助函数。
- `apps/web/package.json`：登记生产 UI 验收命令。
- `apps/web/src/api.js`：将仅供请求封装内部使用的错误类型改为文件私有。
- `apps/web/src/ChannelModelsModal.jsx`：将仅供模型弹窗内部使用的列表转换函数改为文件私有。
- `apps/web/artifacts/*.png`：删除 50 张历史 UI 验收截图，后续按需生成且不进入 Git。
- `deploy/scripts/verify-accounts.sh`：增加受名称和编号保护的测试 Token 物理清理。
- `deploy/tests/test_release_tools.py`：锁定账号验收必须清理平台绑定和核心 Token。
- `docs/运营管理与升级验收.md`：补充账号验收的数据清理边界。
- `docs/部署运行手册.md`：更新生产 UI 验收命令与截图生成物规则。
- `progress.md`：追加代码整理范围、验证证据、残余依赖风险和回滚点。
- 回滚点：本轮前一提交为 `b5c35ba`；执行 `git revert <本轮提交哈希>` 可恢复代码、脚本和历史截图。本轮没有数据库迁移，物理清理仅匹配验收脚本生成的 `contract-*` 测试 Token。

## 2026-07-24 - Task: 增加调用记录高精度金额与 CC Switch 额度查询

### What was done

- 用户、商户和管理员共用的调用记录金额改用专用格式器，按运行时额度单位换算并固定显示 6 位小数；余额、充值、提现、支付订单和汇总卡片继续显示 2 位。
- 新增 `GET /v1/usage`：使用现有 Bearer 密钥指纹查找所属账户，再通过内部管理服务读取实时可用额度；空路由密钥可以查询，停用或删除密钥、停用账号和内部查询异常返回不同错误，不把查询异常伪装成零余额。
- API 密钥操作菜单增加“导入 CC Switch”。只有已绑定模型的密钥可以进入，用户必须明确选择一个授权模型；确认后才按需读取一次完整密钥并生成 Codex/Responses 深链，附带人民币 Usage Script 和 30 分钟查询间隔。
- 密钥正文仍只在浏览器确认动作中短暂传递，不写入平台数据库、日志、文档或进度记录；本轮不修改数据库结构和核心模型服务源码。

### Testing

- 先运行目标用例，确认后端因缺少 `TokenAccountID`、前端因缺少 `formatUsageQuota` 和 CC Switch 生成器而失败；实现后 `go test ./internal/httpapi -run 'Test(UsageBalance|TokenAccountID)' -count=1` 与 `npm.cmd test` 通过。
- `go test ./... -count=1`、`go vet ./...`、`go build ./cmd/api`、`go build ./cmd/worker`、`go build ./cmd/seed-models`：通过。
- `npm.cmd test`、`npm.cmd run test:ui`：通过；桌面和移动烟雾覆盖三类调用记录六位金额、默认模型显式选择、确认前不读取完整密钥和现有密钥路由交互。
- `python -m unittest discover -s deploy/tests -p "test_*.py" -v`：28 项通过。
- `docker compose --env-file .env.example --profile tools config --quiet`：通过。

### Notes

- `apps/api/internal/httpapi/routing_store.go`：增加只接受有效密钥指纹的账户编号查询，并保留数据库故障语义。
- `apps/api/internal/httpapi/routing_store_test.go`：覆盖有效、停用、删除、未知密钥和数据库异常。
- `apps/api/internal/httpapi/usage_balance.go`：实现 CC Switch 兼容的实时人民币余额响应和错误边界。
- `apps/api/internal/httpapi/usage_balance_test.go`：覆盖有效密钥、空路由、账号停用、账户不匹配和内部服务异常。
- `apps/api/internal/httpapi/server.go`：注册 `GET /v1/usage`。
- `apps/web/src/api.js`：增加调用记录专用六位人民币金额格式器。
- `apps/web/src/ccswitch.js`：集中生成 Usage Script 和 `ccswitch://v1/import` 链接。
- `apps/web/src/views/UsagePanels.jsx`：用户、商户和管理员调用记录统一使用六位金额。
- `apps/web/src/views/ConsoleView.jsx`：密钥操作菜单增加 CC Switch 导入及空路由禁用提示。
- `apps/web/src/Modals.jsx`：增加授权模型选择和确认后按需读取密钥的导入弹窗。
- `apps/web/scripts/quota-unit-test.mjs`：锁定 `257`、`500000` 和零额度的六位显示。
- `apps/web/scripts/ccswitch-import-test.mjs`：解码并核对深链地址、模型、脚本和查询间隔。
- `apps/web/scripts/ui-smoke.mjs`：覆盖三种调用记录视角和 CC Switch 导入交互。
- `apps/web/package.json`：登记前端逻辑测试和 CC Switch 专项命令。
- `docs/CC Switch接入.md`：记录导入、额度查询、错误码和模型检测约束。
- `docs/统一调用记录.md`：记录调用金额六位、其他金额两位的展示口径。
- `docs/账号与管理员接口.md`：补充额度接口和密钥导入流程。
- `progress.md`：追加本轮实现、验证证据、文件清单和回滚点。
- 回滚点：本轮前一提交为 `b9a1fc2`；执行 `git revert <本轮提交哈希>` 可回退额度接口、前端导入和六位显示。本轮没有数据库迁移，回滚不会改变现有账号、密钥、路由、余额或调用记录。

## 2026-07-24 - Task: 发布调用记录高精度金额与 CC Switch 额度查询到 103

### What was done

- 将代码整理提交 `b9a1fc2` 与额度查询提交 `a26123dafcdf` 一并归档发布到 103；本地与服务器归档 SHA-256 一致，从独立目录重新构建六个服务。
- 新版本基础验收通过后，将 `current` 从 `5774c931c8ef` 原子切换到 `/opt/modelroute-platform/releases/a26123dafcdf`；模型种子保持 21 个型号不变，没有新增或重复调价。
- 使用一次性空路由密钥验证 `/v1/usage` 可读取实时人民币余额，返回整数额度、动态额度单位与 `CNY`，结果与核心账户整数额度一致；验收密钥及平台绑定已清理。
- 公网前端资源确认包含调用记录六位金额格式器、CC Switch 导入协议和额度查询脚本；未修改数据库结构或核心模型服务源码。

### Testing

- 新发布目录 `deploy/scripts/verify.sh`：通过；内网版本接口返回 `a26123dafcdf` 和 `quota_per_unit=500000`。
- 公网 `/v1/usage`：无密钥返回 `401 invalid_api_key` 且带 `Cache-Control: no-store`；有效空路由密钥返回 `200`、`unit=CNY`、`quota_per_unit=500000`，余额与账户整数额度一致。
- `verify-routing.sh`：通过，临时单厂商密钥只返回已绑定的 `gpt-5.5`，密钥指纹、未绑定模型拒绝和删除路由解绑均符合契约，测试数据已清理。
- `verify-operations.sh`：通过，全平台用量、运营总览、设置、兑换码、任务和渠道只读接口正常。
- 公网 JavaScript 资源检查：包含 `minimumFractionDigits:6`、`maximumFractionDigits:6`、`ccswitch://v1/import`、`/v1/usage` 和 `usageEnabled`。
- 只读压力测试：200 请求、10 并发全部成功，平均 `0.0722s`，P95 `0.088090s`。
- 六个 ModelRoute 容器均为 `running`，重启次数均为 0；网关、API、核心服务、PostgreSQL 和 Redis 健康，最近 10 分钟 API、Worker 和核心服务 `panic/fatal` 为 0。

### Notes

- `progress.md`：追加本次生产发布、专项验收、稳定窗口和回滚证据。
- 生产发布目录：`/opt/modelroute-platform/releases/a26123dafcdf`。
- 回滚点：`/opt/modelroute-platform/releases/5774c931c8ef`；可创建指向该固定目录的临时软链接并使用 `mv -Tf` 原子切回，再执行旧目录的 `deploy/scripts/verify.sh`。本轮没有数据库迁移，回滚不需要恢复数据库。

## 2026-07-24 - Task: 增加模型市场具体模型筛选

### What was done

- 在模型市场厂商分类下增加具体模型筛选行；模型选项只来自当前市场中至少有一条可用渠道的有效模型。
- 具体模型与厂商联动：选择厂商后只显示该厂商型号，切换厂商自动恢复“全部模型”；模型筛选继续与搜索、排序和分页叠加生效。
- 具体模型按钮在桌面和移动端均自动换行，不改变现有厂商单行布局，也不增加后端接口或数据库字段。

### Testing

- 先增加模型筛选 UI 回归，确认旧页面因缺少“全部模型”按钮而失败；完成实现后 `npm.cmd run test:ui` 通过。
- UI 烟雾覆盖全部模型、具体模型、跨厂商联动、厂商切换自动重置、报价行过滤、390px 自动换行和页面级横向溢出。
- `npm.cmd test`：动态额度单位与 CC Switch 导入专项通过。
- Vite 生产构建：1612 个模块转换完成，构建通过。

### Notes

- `apps/web/src/views/MarketView.jsx`：增加可用模型选项、具体模型状态和厂商联动过滤。
- `apps/web/src/styles.css`：增加具体模型自动换行按钮样式和响应式边距。
- `apps/web/scripts/ui-smoke.mjs`：增加桌面筛选联动与移动端布局回归。
- `docs/模型报价路由与健康.md`：记录模型市场两级筛选口径。
- `progress.md`：追加本轮实现、测试证据、文件清单和回滚点。
- 回滚点：本轮前一提交为 `2287582e2ca0`；执行 `git revert <本轮提交哈希>` 可回退模型筛选。该改动只有前端与文档，不涉及数据库和后端接口。

## 2026-07-24 - Task: 发布模型市场具体模型筛选到 103

### What was done

- 将 `6772cc72c6ef` 归档发布到 103，原子切换 `current` 至 `/opt/modelroute-platform/releases/6772cc72c6ef`；前一稳定发布目录 `a26123dafcdf` 保留。
- 新版本从独立发布目录构建六个服务，模型种子重复执行结果为新增 0、调价 0、未变化 21；本轮没有数据库迁移或数据重置。
- 生产模型市场确认显示“全部模型”和 4 个可用 OpenAI 模型；选择 `gpt-5.5` 后市场表只保留对应报价，390px 页面模型按钮自动换行且无横向溢出。

### Testing

- 本地归档与服务器归档 SHA-256 一致；新目录 `deploy/scripts/verify.sh` 通过，内部版本接口返回 `6772cc72c6ef`。
- `verify-routing.sh` 与 `verify-operations.sh`：通过，临时密钥和路由验收数据已清理。
- 真实无登录浏览器：具体模型按钮共 5 个（含“全部模型”），选择 `gpt-5.5` 后报价行数为 1，移动端 `flex-wrap=wrap`、无页面级横向溢出、页面错误数为 0。
- 只读压力测试：200 请求、10 并发全部成功，平均 `0.0716s`，P95 `0.088794s`。
- 六个 ModelRoute 容器均为 `running`，重启次数均为 0；近 10 分钟 API、Worker 和核心服务 `panic/fatal` 为 0。

### Notes

- `progress.md`：追加本次发布、浏览器筛选验收、容器与稳定窗口证据。
- 生产发布目录：`/opt/modelroute-platform/releases/6772cc72c6ef`。
- 回滚点：`/opt/modelroute-platform/releases/a26123dafcdf`；可创建指向该固定目录的临时软链接并使用 `mv -Tf` 原子切回，再执行该目录的 `deploy/scripts/verify.sh`。本轮没有数据库迁移，回滚不需要恢复数据库。

## 2026-07-24 - Task: 项目代码检查与一致性修复

### What was done

- 完成后端、前端、部署编排、依赖、死代码和敏感正文检查；未发现新的阻断项，保留 React 19 与图标库深层依赖声明范围不一致这一非阻断警告，不在本轮升级依赖。
- 修复渠道审核通过的原子性缺口：正式渠道模型绑定改为与渠道归属、健康初始状态、审核状态和待审密钥清除在同一数据库事务内完成，任一步骤失败都会整体回滚。
- 修复并发创建用户密钥时可能识别到同一核心 Token 的竞态：当前单 API 实例串行执行创建前后列表识别及路由绑定；同时删除事务外模型提升接口及其测试桩。

### Testing

- 两个回归用例先确认失败：审批事务缺少模型绑定 SQL；并发密钥创建最大同时进入核心创建数为 2。修复后聚焦测试通过，竞态检测通过。
- `go test ./... -count=1`、`go test -race ./...`、`go vet ./...`、`staticcheck ./...` 和 API、Worker、模型种子构建：通过。
- `npm.cmd test`、`npm.cmd run test:ui` 和 Vite 生产构建：通过，1612 个模块转换完成。
- 部署 Python 测试：28 项通过；`docker compose --env-file .env.example config --quiet`：通过。
- `npm audit --audit-level=high`：0 个漏洞；`npx knip`、Go TODO/FIXME 和受版本控制敏感正文扫描均无结果。

### Notes

- `apps/api/internal/httpapi/server.go`：增加单 API 实例的密钥创建互斥边界。
- `apps/api/internal/httpapi/account_handlers.go`：串行完成核心密钥创建、编号识别和路由绑定。
- `apps/api/internal/httpapi/merchant_store.go`：将正式模型绑定写入移入审批事务并删除独立提升入口。
- `apps/api/internal/httpapi/merchant_handlers.go`：删除审批事务提交后的第二次模型写入。
- `apps/api/internal/httpapi/merchant_types.go`：移除废弃的模型提升仓储接口。
- `apps/api/internal/httpapi/merchant_handlers_test.go`：清理废弃接口测试桩。
- `apps/api/internal/httpapi/merchant_store_test.go`：锁定正式模型绑定必须在审批事务提交前完成。
- `apps/api/internal/httpapi/routing_handlers_test.go`：增加并发密钥创建串行化回归。
- `docs/商户与渠道审核接口.md`：补充渠道审批的完整事务边界。
- `docs/账号与管理员接口.md`：记录密钥编号识别与单 API 实例约束。
- `progress.md`：追加本轮检查、修复、验证证据和回滚点。
- 回滚点：本轮前一提交为 `f3920b31e90d`；执行 `git revert <本轮提交哈希>` 可回退本轮修复。本轮没有数据库迁移，不影响现有账号、密钥、路由、渠道或调用记录数据。

## 2026-07-24 - Task: 修复小额调用后的余额展示精度

### What was done

- 保持核心额度扣减和整数账务不变，将余额及普通金额的两位小数展示改为按分截断，避免小额消费被四舍五入回原金额。
- 使用 BigInt 进行整数额度到展示分的换算，避免 `¥257.28` 等金额因 JavaScript 浮点误差错误显示为 `¥257.27`；调用记录的六位金额格式保持不变。

### Testing

- `npm.cmd run test:quota`：通过，覆盖 `¥257.28`、`¥199.99` 截断和六位调用金额。
- `npm.cmd test`：通过，额度与 CC Switch 导入测试通过。
- `npm.cmd run test:ui`：通过，桌面/移动 UI 烟雾测试通过。
- `npm.cmd run build`：通过，Vite 生产构建完成。

### Notes

- `apps/web/src/api.js`：余额展示使用整数分截断与 BigInt 换算。
- `apps/web/scripts/quota-unit-test.mjs`：增加浮点边界和小额扣费展示回归断言。
- `docs/统一调用记录.md`：记录余额展示精度口径。
- 本轮无数据库、核心服务或部署配置变更；回滚方式为 `git revert <本轮提交哈希>`。

## 2026-07-24 - Task: 发布项目检查修复到 103

### What was done

- 将提交 `7767f69f428a` 生成不可变发布归档并上传到 103，服务器目录为 `/opt/modelroute-platform/releases/7767f69f428a`；保留共享 `.env`、数据和备份目录，未执行数据库重置。
- 使用新目录重新构建平台 API、Worker、模型种子和网关，模型种子结果为新增 `0`、调价 `0`、未变化 `21`；完成后原子切换 `current` 到新发布目录。
- 将提交推送到远端分支 `codex/full-platform`。

### Testing

- 本地与服务器发布归档 SHA-256 均为 `59546a5f8d81be633a5d695d2a2cedb2288de03d672af73620f37a651e4879b4`。
- 新目录内网、公网 `verify.sh`：通过；`/api/version` 内外网均返回 `7767f69f428a`。
- 账号、商户、单厂商密钥与模型路由、提现、运营五组生产契约：全部通过，契约测试临时密钥和路由已清理。
- 只读压力测试 `REQUESTS=200 CONCURRENCY=10`：`200/200` 成功，平均 `0.0016s`，P95 `0.003869s`。
- `current` 指向 `/opt/modelroute-platform/releases/7767f69f428a`；网关、New API、平台 API、Worker、PostgreSQL 和 Redis 均运行，带健康检查的服务为 healthy；`aggregate.lwylink.xyz`、`api.lwylink.xyz`、`sdk.lwylink.xyz` 均返回 `200`。
- 最近 10 分钟平台 API、Worker 和 New API 日志无 `panic/fatal`。

### Notes

- `progress.md`：追加本次生产发布、专项契约、压力测试、稳定窗口和回滚点。
- 生产发布目录：`/opt/modelroute-platform/releases/7767f69f428a`。
- 回滚点：`/opt/modelroute-platform/releases/6772cc72c6ef`；可创建指向该目录的临时软链接并使用 `mv -Tf` 原子切回，再执行旧目录的 `deploy/scripts/deploy.sh` 和 `verify.sh`。本轮无数据库迁移，代码回滚不会改变账号、密钥、路由、渠道或调用记录。
