# ModelRoute CC Switch 接入

## 一键导入

用户在“控制台 -> API 密钥”的操作菜单中选择“导入 CC Switch”。只有已经绑定模型路由的密钥可以导入；弹窗只列出该密钥已授权的模型，并要求用户主动选择一个默认模型。

确认后，页面通过现有密钥归属校验接口按需读取一次完整公开密钥，并在浏览器内生成 `ccswitch://v1/import` 链接。链接固定使用：

- `resource=provider`
- `app=codex`
- 当前站点公开地址作为 `homepage` 和 `endpoint`
- 用户选择的已授权模型
- `usageEnabled=true`
- `usageAutoInterval=30`
- Base64 编码的 Usage Script

六种密钥厂商都使用 Codex/Responses 客户端入口。厂商只限制该密钥可绑定的模型，不改变公开请求协议。完整密钥只交给浏览器的 CC Switch 协议处理器，不写入平台日志、文档或业务数据库。

此前手工导入或旧版本一键导入的配置不包含新版 Usage Script，需要重新执行一键导入。

## 额度查询

`GET /v1/usage` 使用与模型请求相同的 Bearer 密钥鉴权。该接口查询密钥所属用户的实时可用余额，与单把密钥的额度上限无关；密钥没有绑定模型路由时也可以查询。

正常响应示例：

```json
{
  "isValid": true,
  "is_active": true,
  "remaining": 0.000514,
  "balance": 0.000514,
  "unit": "CNY",
  "remaining_quota": 257,
  "quota_per_unit": 500000
}
```

`remaining_quota` 保持整数额度，人民币金额只在响应时按运行中的 `quota_per_unit` 换算，不改变账务存储。响应带 `Cache-Control: no-store`，且不包含用户邮箱、昵称、密钥正文或模型路由。

- 密钥不存在、已停用或已删除：`401 invalid_api_key`。
- 所属账号已停用：`403 account_inactive`。
- 账户服务或平台绑定查询异常：`503 balance_unavailable`，避免客户端把暂时查询失败误显示为零余额。

## 模型检测

CC Switch 与 Codex++ 检测时必须使用导入时选择的授权模型。平台保持严格模型授权：未绑定模型返回 `403 model_not_allowed`，不会自动替换为其他模型。

## 验证

前端测试解码深链并检查公开地址、默认模型、Usage Script 和 30 分钟间隔；后端测试覆盖有效密钥、停用或删除密钥、停用账号、空路由密钥和内部账户查询异常。
