# 探海 (Tanhai) — 护网红队资产存活探测工具

[简体中文](README.md) | [English](README_zh.md)

> 在网络之海中，探得每一个目标。
> Fast & Multi-purpose HTTP Toolkit for Red Team Asset Discovery.

基于 Go 标准库的高并发 HTTP 存活探测工具，专为护网红队场景设计，零依赖、单文件、跨平台。

## 特性

- 🌊 **零依赖**：纯 Go 标准库，单文件可编译运行
- 🌊 **智能并发**：自动适配 CPU 核数与目标数（10-500）
- 🌊 **格式兼容**：支持各种非标准 URL 格式
- 🌊 **彩色输出**：终端中文彩色 + JSON/CSV 双报告
- 🌊 **自动报告**：自动生成 `探海报告-<时间戳>.json` 和 `.csv`
- 🌊 **跨平台**：Windows / Linux / macOS 三大平台原生支持
- 🌊 **错误归类**：DNS / TLS / 连接 / 超时 错误分类统计
- 🌊 **代理支持**：HTTP 代理支持（红队 IP 轮换）

## 项目命名

| 维度 | 名称 |
|------|------|
| 中文名 | **探海** |
| 英文名 | **Tanhai** |
| Go 模块 | `tanhai` |
| 二进制 | `tanhai` / `tanhai.exe` |
| 报告 | `探海报告-<时间戳>.json` / `.csv` |

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
| `//tager.com/path` | protocol-relative，自动补 `http:` |
| `http://[::1]:8080/` | IPv6 |
| `192.168.1.1:8080` | IP + 端口 |
| `"http://tager.com"` | 自动去引号 |
| `:8080` | 拒绝（无主机）|

## 编译

### 本机编译

```bash
cd tanhai
go build -o tanhai main.go
```

### 跨平台编译（一键生成 8 个平台二进制）

```bash
# Unix / macOS / Linux
./build.sh

# Windows
build.bat
```

产物在 `dist/` 目录：

| 平台 | 文件 |
|------|------|
| Windows amd64 | `tanhai.exe` |
| Windows x86 | `tanhai-x86.exe` |
| Windows arm64 | `tanhai-arm64.exe` |
| Linux amd64 | `tanhai-linux` |
| Linux x86 | `tanhai-linux-x86` |
| Linux arm64 | `tanhai-linux-arm64` |
| macOS amd64 | `tanhai-darwin-amd64` |
| macOS arm64 | `tanhai-darwin-arm64` |

## 使用示例

### 基础

```bash
# 单个 URL 探测
./tanhai -u https://target.com

# 从文件读取
./tanhai -f urls.txt

# 通过管道输入
cat urls.txt | ./tanhai
echo "tager.com" | ./tanhai
```

### 高级

```bash
# 通过 HTTP 代理探测（红队 IP 轮换）
./tanhai -f urls.txt -proxy http://127.0.0.1:8080
./tanhai -f urls.txt -proxy http://user:pass@proxy.example.com:3128

# 配合 FOFA / Quake 批量探测
fofa_export.txt | ./tanhai
```

## 命令行参数

```
-f string     目标文件(每行一个URL)
-u string     单个URL
-proxy string 代理(http://host:port)
```

只有 3 个参数，其他全自动：
- 线程数：自动适配（CPU 核数 × 50 + 目标数 ÷ 2）
- 超时：默认 5 秒
- HTTP 方法：默认 GET
- TLS：跳过证书验证（红队场景必需）
- 输出：自动 JSON + CSV 双文件

## 运行示例

### Help 输出

```
探海 v1.0 - 资产存活探测工具

用法: ./tanhai -f <文件> | -u <URL> [-proxy 代理]
      cat urls.txt | ./tanhai

支持格式: http://t.com/web | https://t.com:8080 | t.com | t.ico/ | 192.168.1.1:80

平台: darwin/arm64

选项:
  -f string
    	目标文件(每行一个URL)
  -proxy string
    	代理(http://host:port)
  -u string
    	单个URL

示例:
  ./tanhai -f urls.txt
  ./tanhai -u https://target.com
  echo tager.com | ./tanhai
  ./tanhai -f urls.txt -proxy http://127.0.0.1:8080

输出: 自动保存 探海报告-<时间戳>.json 和 .csv 到当前目录
```

### 探测运行

```
[*] 探海 v1.0 | 平台 darwin/arm64 | 目标 51 | 自适应并发 500
[+] 在线 [200 请求成功] https://www.baidu.com [IP:180.101.49.44] [标题:百度一下，你就知道] [服务:BWS/1.1]
[+] 在线 [200 请求成功] https://www.qq.com [IP:101.91.42.232] [标题:腾讯网] [服务:tRPC-Gateway]
[+] 在线 [200 请求成功] https://www.zhihu.com [IP:43.242.197.211] [标题:知乎 - 有问题，就会有答案] [服务:BLB/25.12.0.2]
[*] 已保存: 探海报告-20260917-161006.json
[*] 已保存: 探海报告-20260917-161006.csv

========== 探测统计 ==========
[*] 总数 3 | 在线 3 | 离线 0
[*] 耗时 3.76s | 速度 0.8/秒
[*] 状态码:
    200 请求成功: 3
```

## 输出格式

### 自动输出文件

- `探海报告-<时间戳>.json` — 完整结构化数据（含中文 status_desc、error_desc）
- `探海报告-<时间戳>.csv` — Excel 兼容（含 UTF-8 BOM，中文不乱码）

### JSON 输出示例

```json
[
  {
    "url": "https://www.baidu.com",
    "online": true,
    "status_code": 200,
    "status_desc": "请求成功",
    "title": "百度一下，你就知道",
    "server": "BWS/1.1",
    "ip": "180.101.49.44",
    "content_length": 81,
    "duration_ms": 234
  },
  {
    "url": "https://offline.com",
    "online": false,
    "status_code": 0,
    "error": "Get \"https://offline.com\": dial tcp: no such host",
    "error_type": "dns_error",
    "error_desc": "DNS解析失败",
    "duration_ms": 88
  }
]
```

### CSV 输出示例

```csv
URL,在线,状态码,状态描述,标题,服务,IP,最终URL,内容长度,耗时(ms),错误类型,错误信息
https://www.baidu.com,是,200,请求成功,百度一下，你就知道,BWS/1.1,180.101.49.44,,81,234,,
```

## 性能

- 局域网健康目标：300-800 req/s
- 公网国内目标：30-100 req/s
- 内存占用：< 30 MB（百万级目标无压力）
- 跨平台性能差异：< 20%

## 错误类型分类

| 类型 | 中文描述 |
|------|---------|
| `parse_error` | URL 解析失败 |
| `dns_error` | DNS 解析失败 |
| `connect_error` | 连接失败（被拒、超时、不可达）|
| `tls_error` | TLS 证书错误 |
| `timeout` | 请求超时 |
| `http_error` | HTTP 错误 |
| `unknown` | 未知错误 |

## 状态码映射

| 状态码 | 中文描述 |
|--------|---------|
| 200 | 请求成功 |
| 301 | 永久重定向 |
| 302 | 临时重定向 |
| 400 | 请求错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 未找到 |
| 429 | 请求过多 |
| 500 | 服务器错误 |
| 502 | 网关错误 |
| 503 | 服务不可用 |
| 504 | 网关超时 |

## 跨平台兼容

### Windows
- ✅ 默认禁用彩色（兼容老 cmd.exe）
- ✅ Windows Terminal / PowerShell 7+ 自动启用颜色
- ✅ 中文文件名支持
- ✅ 信号处理兼容（os.Interrupt + SIGINT + SIGTERM）

### Linux
- ✅ 默认启用彩色
- ✅ epoll 高性能
- ✅ UTF-8 文件名

### macOS
- ✅ 默认启用彩色
- ✅ kqueue 高性能
- ✅ 同时支持 Intel (amd64) 与 Apple Silicon (arm64)

## 护网红队最佳实践

### 1. 从 FOFA / Quake 导出后批量探测
```bash
# FOFA 导出 → 直接探测
cat fofa_export.txt | ./tanhai -w 500
```

### 2. 配合代理池做 IP 轮换
```bash
for proxy in $(cat proxies.txt); do
    ./tanhai -f urls.txt -proxy "$proxy"
done
```

### 3. 二次筛选（jq 配合 JSON 报告）
```bash
# 筛选在线目标
jq '[.[] | select(.online==true)]' 探海报告-*.json

# 筛选 200 OK 目标
jq '[.[] | select(.status_code==200)]' 探海报告-*.json

# 统计每个 IP 出现次数
jq -r '.[].ip' 探海报告-*.json | sort | uniq -c | sort -rn | head -20
```

### 4. CSV 在 Excel 中打开
- 文件含 UTF-8 BOM，Excel 自动识别中文编码
- 字段顺序：URL、在线、状态码、状态描述、标题、服务、IP、最终URL、内容长度、耗时(ms)、错误类型、错误信息

## 适用场景

- ✅ 资产存活验证
- ✅ 子域/URL 列表探测
- ✅ 大批量 IP/域名快速扫描
- ✅ HTTP 服务指纹识别（title + server）
- ✅ 内网资产存活摸排
- ❌ 不适合：登录认证、Web 漏洞扫描、深度爬虫（请用 Burp/ZAP 等专业工具）

## License

仅供授权的安全测试与攻防演练使用。严禁用于未授权的非法活动。

---

**探海 v1.0** — 在网络之海中，探得每一个目标。
