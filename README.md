# 探海(tanhai) — 护网红队资产存活探测工具

基于 Go 标准库的高并发 HTTP 存活探测工具，专为护网红队场景设计，支持各种非标准 URL。

## 特性

- ✅ **零依赖**：纯 Go 标准库，开箱即用
- ✅ **高并发**：worker pool，默认 50 并发，可配 200+
- ✅ **格式兼容**：支持各种奇葩 URL 格式
- ✅ **彩色输出**：终端彩色 + JSON 输出
- ✅ **信息提取**：自动提取 title、server、IP、content-length
- ✅ **错误归类**：DNS / TLS / 连接 / 超时 错误分类统计
- ✅ **跨平台**：macOS / Linux / Windows

## 支持的目标格式

| 输入 | 处理结果 |
|------|---------|
| `http://tager.com/web` | 标准 HTTP |
| `https://tager.tager.cn` | 标准 HTTPS |
| `tager.ico/` | 自动补 `http://`，原样保留 |
| `https://dadg.com:8080` | 端口号 |
| `http://dafasg.com:8888/` | 端口 + 路径 + 斜杠 |
| `tager.com` | 自动补 `http://` |
| `tager.com:8080/admin` | 端口 + 路径 |
| `//tager.com/path` | protocol-relative，补 `http:` |
| `http://[::1]:8080/` | IPv6 |
| `192.168.1.1:8080` | IP + 端口 |
| `"http://tager.com"` | 自动去引号 |
| `:8080` | 拒绝（无主机）|

## 编译

```bash
cd asset-probe
go build -o asset-probe main.go
```

## 使用示例

```bash
# 单个 URL 探测
./asset-probe -u https://target.com

# 从文件读取
./asset-probe -f urls.txt -w 100 -t 3s

# 通过管道输入
cat urls.txt | ./asset-probe -w 50

# 输出 JSON 结果
./asset-probe -f urls.txt -o result.json

# 详细输出（包含错误）
./asset-probe -f urls.txt -v

# 强制 HTTPS + 跟随重定向
./asset-probe -f urls.txt -scheme https -r -max-redir 5

# 通过代理探测（HTTP 代理）
./asset-probe -f urls.txt -proxy http://127.0.0.1:8080

# 禁用彩色输出（重定向到文件）
./asset-probe -f urls.txt -no-color > result.txt

# 自定义 User-Agent
./asset-probe -u https://target.com -ua "Custom UA"
```

## 命令行参数

```
-f string        目标列表文件 (一行一个 URL，支持 # 注释)
-u string        单个 URL
-w int           并发数 (建议 50-200, 默认 50)
-t duration      单个请求超时 (默认 5s)
-m string        HTTP 方法 (GET/HEAD, 默认 GET)
-ua string       自定义 User-Agent
-o string        JSON 输出文件
-k bool          跳过 TLS 证书验证 (默认 true)
-r bool          跟随重定向
-max-redir int   最大重定向次数 (默认 3)
-scheme string   强制协议 (http/https/留空)
-proxy string    HTTP 代理 (例: http://127.0.0.1:8080)
-v bool          详细输出 (含错误信息)
-no-color bool   禁用彩色输出
```

## 输出示例

### 彩色终端输出

```
[*] 共 5 个目标 | 并发: 50 | 超时: 5s | 方法: GET
[*] 进度: 5/5 (100.0%) | 速度: 23 req/s

========== 统计 ==========
[*] 总数: 5 | 在线: 4 | 离线: 1
[*] 耗时: 213ms | 速度: 23.5 req/s
[*] 状态码分布:
    200: 3
    301: 1
[*] 错误分布:
    connect_error: 1
```

### JSON 输出格式

```json
[
  {
    "url": "http://tager.com/web",
    "online": true,
    "status_code": 200,
    "title": "Tager Platform",
    "server": "nginx/1.21.0",
    "ip": "93.184.216.34",
    "final_url": "http://tager.com/web",
    "content_length": 12345,
    "redirect_num": 0,
    "duration_ms": 187
  },
  {
    "url": "https://dadg.com:8080",
    "online": false,
    "status_code": 0,
    "error": "dial tcp 1.2.3.4:8080: connect: connection refused",
    "error_type": "connect_error",
    "duration_ms": 5023
  }
]
```

## 护网红队最佳实践

1. **批量探测**：从 FOFA / Quake 导出的目标直接 `cat urls.txt | ./asset-probe -w 100`
2. **筛选活跃目标**：JSON 输出后用 `jq` 筛选 `online==true`
3. **状态码二次筛选**：`jq 'map(select(.status_code == 200))' result.json`
4. **多线程 + 代理轮换**：配合 `-proxy` 切换不同出口

## 错误类型分类

| 类型 | 含义 |
|------|------|
| `parse_error` | URL 解析失败 |
| `dns_error` | DNS 解析失败 |
| `connect_error` | TCP 连接失败（被拒、超时、不可达） |
| `tls_error` | TLS 握手失败（证书、协议） |
| `timeout` | 总超时（含读写） |
| `http_error` | 其他 HTTP 层错误 |
| `unknown` | 未知错误 |

## 性能参数

- 默认并发：50，可提升到 200-500（取决于网络/目标）
- 单目标耗时：通常 100-500ms（成功），5s（超时）
- 内存占用：~10MB（百万级目标无压力）

## License

仅供授权的安全测试与攻防演练使用。
