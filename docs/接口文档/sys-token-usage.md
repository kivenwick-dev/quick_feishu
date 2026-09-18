# 获取令牌使用情况

## 接口信息

| 项目 | 说明 |
| --- | --- |
| 请求方法 | `GET` |
| 请求地址 | `https://api.quickrouter.ai/api/usage/token/` |
| 请求类型 | 系统 API |

## 鉴权方式

在 Header 中添加 `Authorization` 参数，其值为 `Bearer` 之后拼接 Token。

示例：`Authorization: Bearer ********************`

## 请求参数

### Header 参数

| 参数 | 类型 | 必填 | 说明 | 示例 |
| --- | --- | --- | --- | --- |
| `Authorization` | string | 是 | 你的系统令牌 | `Bearer {{YOUR_API_KEY}}` |

## 请求示例

```bash
curl --location --request GET 'https://api.quickrouter.ai/api/usage/token/' \
--header 'Accept: application/json' \
--header 'Authorization: Bearer YOUR_API_KEY' \
--header 'Content-Type: application/json'
```

## 返回响应

- 状态码：`200 OK`
- 响应类型：`application/json`
- 响应体：`object`（可选）

## 相关文档

- [发出请求](/docs/making-requests.html)
- [代理接口调用地址](/docs/proxy-endpoint.html)
- [HTTP 状态码说明](/docs/help-http-status.html)
- [分组详细表格](/docs/group-details.html)