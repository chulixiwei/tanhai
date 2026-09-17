# Tanhai (探海) — Red Team Asset Liveness Probe

> Discovering every target in the vast ocean of networks.
> Fast & Multi-purpose HTTP Toolkit for Red Team Asset Discovery.

A high-concurrency HTTP liveness probe built with Go's standard library. Designed for red team engagements, with zero dependencies, single-file deployment, and full cross-platform support.

## Features

- 🌊 **Zero Dependencies**: Pure Go standard library, single binary compiles and runs
- 🌊 **Smart Concurrency**: Auto-adapts to CPU cores and target count (10-500)
- 🌊 **Format Compatibility**: Supports various non-standard URL formats
- 🌊 **Color Output**: Terminal Chinese color + JSON/CSV dual reports
- 🌊 **Auto Reports**: Auto-generates `探海报告-<timestamp>.json` and `.csv`
- 🌊 **Cross-Platform**: Native support for Windows / Linux / macOS
- 🌊 **Error Classification**: DNS / TLS / connection / timeout error categorization
- 🌊 **Proxy Support**: HTTP proxy support (red team IP rotation)

## Project Naming

| Field | Name |
|-------|------|
| Chinese Name | **探海** |
| English Name | **Tanhai** |
| Go Module | `tanhai` |
| Binary | `tanhai` / `tanhai.exe` |
| Reports | `探海报告-<timestamp>.json` / `.csv` |

## Supported Target Formats

| Input | Result |
|-------|--------|
| `http://tager.com/web` | Standard HTTP |
| `https://tager.tager.cn` | Standard HTTPS |
| `tager.ico/` | Auto-prefix `http://`, path kept as-is |
| `https://dadg.com:8080` | Port number |
| `http://dafasg.com:8888/` | Port + path + trailing slash |
| `tager.com` | Auto-prefix `http://` |
| `tager.com:8080/admin` | Port + path |
| `//tager.com/path` | protocol-relative, auto-prefix `http:` |
| `http://[::1]:8080/` | IPv6 |
| `192.168.1.1:8080` | IP + port |
| `"http://tager.com"` | Auto-remove quotes |
| `:8080` | Rejected (no host) |

## Build

### Local Build

```bash
cd tanhai
go build -o tanhai main.go
```

### Cross-Platform Build (one-click generation of 8 platform binaries)

```bash
# Unix / macOS / Linux
./build.sh

# Windows
build.bat
```

Products in `dist/` directory:

| Platform | File |
|----------|------|
| Windows amd64 | `tanhai.exe` |
| Windows x86 | `tanhai-x86.exe` |
| Windows arm64 | `tanhai-arm64.exe` |
| Linux amd64 | `tanhai-linux` |
| Linux x86 | `tanhai-linux-x86` |
| Linux arm64 | `tanhai-linux-arm64` |
| macOS amd64 | `tanhai-darwin-amd64` |
| macOS arm64 | `tanhai-darwin-arm64` |

## Usage Examples

### Basic

```bash
# Single URL probe
./tanhai -u https://target.com

# Read from file
./tanhai -f urls.txt

# Pipe input
cat urls.txt | ./tanhai
echo "tager.com" | ./tanhai
```

### Advanced

```bash
# Probe via HTTP proxy (red team IP rotation)
./tanhai -f urls.txt -proxy http://127.0.0.1:8080
./tanhai -f urls.txt -proxy http://user:pass@proxy.example.com:3128

# Bulk probe with FOFA / Quake exports
fofa_export.txt | ./tanhai
```

## Command Line Options

```
-f string     Target file (one URL per line)
-u string     Single URL
-proxy string Proxy (http://host:port)
```

Only 3 options, everything else is automatic:
- Threads: Auto-adapted (CPU cores × 50 + target count ÷ 2)
- Timeout: 5 seconds default
- HTTP Method: GET default
- TLS: Skip certificate verification (required for red team)
- Output: Auto JSON + CSV dual files

## Runtime Examples

### Help Output

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

### Probe Runtime

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

## Output Formats

### Auto Output Files

- `探海报告-<timestamp>.json` — Complete structured data (includes Chinese status_desc, error_desc)
- `探海报告-<timestamp>.csv` — Excel compatible (UTF-8 BOM, no garbled Chinese)

### JSON Output Example

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

### CSV Output Example

```csv
URL,在线,状态码,状态描述,标题,服务,IP,最终URL,内容长度,耗时(ms),错误类型,错误信息
https://www.baidu.com,是,200,请求成功,百度一下，你就知道,BWS/1.1,180.101.49.44,,81,234,,
```

## Performance

- LAN healthy targets: 300-800 req/s
- Public domestic targets: 30-100 req/s
- Memory footprint: < 30 MB (million-scale targets no pressure)
- Cross-platform performance variance: < 20%

## Error Type Classification

| Type | Description |
|------|-------------|
| `parse_error` | URL Parse Failed |
| `dns_error` | DNS Resolution Failed |
| `connect_error` | Connection Failed (Refused, Timeout, Unreachable) |
| `tls_error` | TLS Certificate Error |
| `timeout` | Request Timeout |
| `http_error` | HTTP Error |
| `unknown` | Unknown Error |

## Status Code Mapping

| Code | Description |
|------|-------------|
| 200 | OK |
| 301 | Permanent Redirect |
| 302 | Temporary Redirect |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 429 | Too Many Requests |
| 500 | Internal Server Error |
| 502 | Bad Gateway |
| 503 | Service Unavailable |
| 504 | Gateway Timeout |

## Cross-Platform Compatibility

### Windows
- ✅ Color disabled by default (compatible with legacy cmd.exe)
- ✅ Windows Terminal / PowerShell 7+ auto-enables colors
- ✅ Chinese filename support
- ✅ Signal handling compatible (os.Interrupt + SIGINT + SIGTERM)

### Linux
- ✅ Color enabled by default
- ✅ epoll high performance
- ✅ UTF-8 filenames

### macOS
- ✅ Color enabled by default
- ✅ kqueue high performance
- ✅ Supports both Intel (amd64) and Apple Silicon (arm64)

## Red Team Best Practices

### 1. Bulk Probe from FOFA / Quake Exports
```bash
# FOFA export → direct probe
cat fofa_export.txt | ./tanhai -w 500
```

### 2. IP Rotation via Proxy Pool
```bash
for proxy in $(cat proxies.txt); do
    ./tanhai -f urls.txt -proxy "$proxy"
done
```

### 3. Secondary Filtering (jq with JSON Reports)
```bash
# Filter online targets
jq '[.[] | select(.online==true)]' 探海报告-*.json

# Filter 200 OK targets
jq '[.[] | select(.status_code==200)]' 探海报告-*.json

# Count each IP's frequency
jq -r '.[].ip' 探海报告-*.json | sort | uniq -c | sort -rn | head -20
```

### 4. CSV in Excel
- File contains UTF-8 BOM, Excel auto-detects Chinese encoding
- Field order: URL, 在线, 状态码, 状态描述, 标题, 服务, IP, 最终URL, 内容长度, 耗时(ms), 错误类型, 错误信息

## Use Cases

- ✅ Asset liveness validation
- ✅ Subdomain / URL list probing
- ✅ Large-scale IP / domain fast scanning
- ✅ HTTP service fingerprinting (title + server)
- ✅ Internal network asset liveness discovery
- ❌ Not suitable for: authentication, web vulnerability scanning, deep crawling (use Burp/ZAP or other professional tools)

## License

For authorized security testing and red team engagements only. Unauthorized use is strictly prohibited.

---

**探海 v1.0** — 在网络之海中，探得每一个目标。