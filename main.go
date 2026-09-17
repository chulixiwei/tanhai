// Package tanhai 探海 - 资产存活探测工具（跨平台版）
//
// 中文名: 探海
// 英文名: Tanhai
//
// 支持平台: Windows / Linux / macOS
// 支持架构: amd64 / arm64 / 386
//
// 用法:
//
//	./tanhai -f urls.txt
//	./tanhai -u https://target.com
//	cat urls.txt | ./tanhai
//
// 可选参数:
//
//	-f / -u / -proxy（其他全自动）
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// 终端颜色
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// 探测配置
type ProbeConfig struct {
	Timeout          time.Duration
	Workers          int
	UserAgent        string
	InsecureTLS      bool
	Proxy            string
	EnableColor      bool
	BodyLimit        int64 // 响应体读取上限
	IdleConnsPerHost int   // 单 host 空闲连接数
	MaxIdleConns     int   // 全局最大空闲连接数
}

// 探测结果
type Result struct {
	URL         string `json:"url"`
	Online      bool   `json:"online"`
	StatusCode  int    `json:"status_code"`
	StatusDesc  string `json:"status_desc"`
	Title       string `json:"title,omitempty"`
	Server      string `json:"server,omitempty"`
	IP          string `json:"ip,omitempty"`
	FinalURL    string `json:"final_url,omitempty"`
	ContentLen  int64  `json:"content_length"`
	RedirectNum int    `json:"redirect_num"`
	Error       string `json:"error,omitempty"`
	ErrorType   string `json:"error_type,omitempty"`
	ErrorDesc   string `json:"error_desc,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
}

// 中文映射
var errTypeCN = map[string]string{
	"parse_error":   "URL解析失败",
	"dns_error":     "DNS解析失败",
	"connect_error": "连接失败",
	"tls_error":     "TLS证书错误",
	"timeout":       "请求超时",
	"http_error":    "HTTP错误",
	"unknown":       "未知错误",
}

var statusCodeCN = map[int]string{
	200: "请求成功", 201: "已创建", 204: "无内容",
	301: "永久重定向", 302: "临时重定向", 304: "未修改",
	400: "请求错误", 401: "未授权", 403: "禁止访问",
	404: "未找到", 405: "方法不允许", 429: "请求过多",
	500: "服务器错误", 502: "网关错误", 503: "服务不可用", 504: "网关超时",
}

// SmartParseURL 智能解析 URL
func SmartParseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "\"'`<>")
	if raw == "" {
		return nil, fmt.Errorf("URL为空")
	}
	if strings.HasPrefix(raw, "//") {
		raw = "http:" + raw
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		if strings.HasPrefix(raw, ":") {
			return nil, fmt.Errorf("缺少主机")
		}
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("URL解析失败: %v", err)
	}
	if parsed.Host == "" {
		if strings.Contains(raw, ".") && !strings.Contains(raw, " ") {
			return url.Parse("http://" + raw)
		}
		return nil, fmt.Errorf("URL缺少主机")
	}
	return parsed, nil
}

// classifyError 错误归类
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "DNS"):
		return "dns_error"
	case strings.Contains(msg, "timeout"):
		return "timeout"
	case strings.Contains(msg, "tls"), strings.Contains(msg, "x509"), strings.Contains(msg, "certificate"):
		return "tls_error"
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "connect: "),
		strings.Contains(msg, "network is unreachable"), strings.Contains(msg, "no route to host"):
		return "connect_error"
	case strings.Contains(msg, "parse"), strings.Contains(msg, "invalid"):
		return "parse_error"
	default:
		return "unknown"
	}
}

// extractTitle 提取 HTML 标题
func extractTitle(html string) string {
	if len(html) == 0 || len(html) > 1024*1024 {
		return ""
	}
	lower := strings.ToLower(html)
	startIdx := strings.Index(lower, "<title")
	if startIdx < 0 {
		return ""
	}
	gtIdx := strings.Index(lower[startIdx:], ">")
	if gtIdx < 0 {
		return ""
	}
	contentStart := startIdx + gtIdx + 1
	endIdx := strings.Index(lower[contentStart:], "</title>")
	if endIdx < 0 {
		return ""
	}
	title := strings.TrimSpace(html[contentStart : contentStart+endIdx])
	if len(title) > 120 {
		title = title[:120] + "..."
	}
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, title)
}

// autoWorkers 自动适配线程数（优化版：基于 IO 密集型特性）
func autoWorkers(targetCount int) int {
	cpu := runtime.NumCPU()
	if targetCount <= 0 {
		return 10
	}
	// HTTP 探测是 IO 密集型，瓶颈是带宽/目标响应而非 CPU
	// 基准：CPU * 50 + 目标数 / 2（更激进）
	base := cpu*50 + targetCount/2
	if base < 10 {
		base = 10
	}
	if base > 500 {
		base = 500
	}
	return base
}

// Probe 探测单个目标
func Probe(ctx context.Context, target string, cfg ProbeConfig) Result {
	start := time.Now()
	res := Result{URL: target}

	u, err := SmartParseURL(target)
	if err != nil {
		res.Error = err.Error()
		res.ErrorType = classifyError(err)
		res.ErrorDesc = errTypeCN[res.ErrorType]
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	ips, _ := net.DefaultResolver.LookupHost(ctx, u.Hostname())
	if len(ips) > 0 {
		res.IP = ips[0]
	}

	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: cfg.InsecureTLS},
		ResponseHeaderTimeout: cfg.Timeout,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   cfg.Timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          cfg.MaxIdleConns,     // 全局空闲连接池
		MaxIdleConnsPerHost:   cfg.IdleConnsPerHost, // 单 host 空闲连接（关键）
		DisableCompression:    true,
		DisableKeepAlives:     false, // 启用 keep-alive
	}

	if cfg.Proxy != "" {
		if proxyURL, err := url.Parse(cfg.Proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	transport.DialContext = (&net.Dialer{
		Timeout:   cfg.Timeout,
		KeepAlive: 30 * time.Second,
	}).DialContext

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		res.Error = err.Error()
		res.ErrorType = classifyError(err)
		res.ErrorDesc = errTypeCN[res.ErrorType]
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	req.Header.Set("User-Agent", cfg.UserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		res.ErrorType = classifyError(err)
		res.ErrorDesc = errTypeCN[res.ErrorType]
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, cfg.BodyLimit))
	res.StatusCode = resp.StatusCode
	res.StatusDesc = statusCodeCN[resp.StatusCode]
	res.Title = extractTitle(string(body))
	res.Server = resp.Header.Get("Server")
	if resp.Request != nil && resp.Request.URL != nil {
		res.FinalURL = resp.Request.URL.String()
	}
	res.ContentLen = resp.ContentLength
	if res.ContentLen < 0 {
		res.ContentLen = int64(len(body))
	}
	res.Online = true
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}

// loadTargets 加载目标
func loadTargets(file, single string) []string {
	var targets []string

	if single != "" {
		targets = append(targets, single)
	}

	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[-] 打开文件失败: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			targets = append(targets, line)
		}
	}

	if file == "" && single == "" {
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				targets = append(targets, line)
			}
		}
	}

	return targets
}

// runProbe 并发探测
func runProbe(ctx context.Context, targets []string, cfg ProbeConfig) []Result {
	targetCh := make(chan string, len(targets))
	resultCh := make(chan Result, len(targets))

	var doneCount atomic.Int64
	total := int64(len(targets))

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range targetCh {
				r := Probe(ctx, t, cfg)
				resultCh <- r
				doneCount.Add(1)
			}
		}()
	}

	// 进度条（跨平台用 \r 回车即可）
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				completed := doneCount.Load()
				if completed >= total {
					return
				}
				pct := float64(completed) / float64(total) * 100
				elapsed := time.Since(start).Seconds()
				speed := float64(completed) / elapsed
				fmt.Fprintf(os.Stderr, "\r[*] 进度 %d/%d (%.1f%%) 速度 %.0f/秒   ",
					completed, total, pct, speed)
			}
		}
	}()

	for _, t := range targets {
		targetCh <- t
	}
	close(targetCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]Result, 0, len(targets))
	for r := range resultCh {
		results = append(results, r)
	}
	fmt.Fprintln(os.Stderr)
	return results
}

// printResult 实时输出
func printResult(r Result, color bool) {
	if color {
		prefix := colorRed + "[-] 离线" + colorReset
		if r.Online {
			prefix = colorGreen + "[+] 在线" + colorReset
		}

		statusColor := colorRed
		if r.Online {
			switch {
			case r.StatusCode >= 200 && r.StatusCode < 300:
				statusColor = colorGreen
			case r.StatusCode >= 300 && r.StatusCode < 400:
				statusColor = colorCyan
			case r.StatusCode >= 400 && r.StatusCode < 500:
				statusColor = colorYellow
			}
		}

		line := fmt.Sprintf("%s [%s%d %s%s] %s", prefix, statusColor, r.StatusCode, r.StatusDesc, colorReset, r.URL)
		if r.IP != "" {
			line += fmt.Sprintf(" [IP:%s]", r.IP)
		}
		if r.Title != "" {
			line += fmt.Sprintf(" [标题:%s]", r.Title)
		}
		if r.Server != "" {
			line += fmt.Sprintf(" [服务:%s]", r.Server)
		}
		if !r.Online && r.ErrorDesc != "" {
			line += fmt.Sprintf(" [%s]", r.ErrorDesc)
		}
		fmt.Println(line)
	} else {
		// 纯文本（Windows 老 cmd.exe 友好）
		status := fmt.Sprintf("%d", r.StatusCode)
		if r.StatusDesc != "" {
			status += " " + r.StatusDesc
		}
		if r.Online {
			fmt.Printf("[+] 在线 [%s] %s", status, r.URL)
		} else {
			fmt.Printf("[-] 离线 [%s] %s", status, r.URL)
		}
		if r.IP != "" {
			fmt.Printf(" [IP:%s]", r.IP)
		}
		if r.Title != "" {
			fmt.Printf(" [标题:%s]", r.Title)
		}
		if r.Server != "" {
			fmt.Printf(" [服务:%s]", r.Server)
		}
		if !r.Online && r.ErrorDesc != "" {
			fmt.Printf(" [%s]", r.ErrorDesc)
		}
		fmt.Println()
	}
}

// saveJSON 保存 JSON
func saveJSON(results []Result, filename string) error {
	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, out, 0644)
}

// saveCSV 保存 CSV（兼容 Excel）
func saveCSV(results []Result, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	// UTF-8 BOM（让 Excel 正确识别中文）
	file.WriteString("\xEF\xBB\xBF")

	if err := w.Write([]string{
		"URL", "在线", "状态码", "状态描述", "标题", "服务",
		"IP", "最终URL", "内容长度", "耗时(ms)", "错误类型", "错误信息",
	}); err != nil {
		return err
	}

	for _, r := range results {
		online := "否"
		if r.Online {
			online = "是"
		}
		row := []string{
			r.URL,
			online,
			strconv.Itoa(r.StatusCode),
			r.StatusDesc,
			r.Title,
			r.Server,
			r.IP,
			r.FinalURL,
			strconv.FormatInt(r.ContentLen, 10),
			strconv.FormatInt(r.DurationMs, 10),
			r.ErrorDesc,
			r.Error,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// isTerminal 判断是否终端
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// shouldEnableColor 跨平台颜色判断
//
// Windows:
//   - 默认禁用（兼容老 cmd.exe）
//   - Windows Terminal (WT_SESSION) / PowerShell 7+ / ConEmu 启用
//
// Unix:
//   - 默认启用（终端直连时）
//   - 管道/文件重定向时禁用
func shouldEnableColor() bool {
	// 1. 非终端 → 禁用
	if !isTerminal(os.Stdout) {
		return false
	}
	// 2. 强制禁用
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// 3. 强制启用
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	// 4. 按平台默认
	switch runtime.GOOS {
	case "windows":
		// Windows Terminal
		if os.Getenv("WT_SESSION") != "" {
			return true
		}
		// ConEmu
		if os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		// PowerShell 7+ 默认支持 ANSI
		if os.Getenv("PSVersionTable") != "" || os.Getenv("PSModulePath") != "" {
			// PS 5+ 也支持，但保险起见仅在 PS 7+ 启用
			psVer := os.Getenv("PS_VERSION")
			if strings.HasPrefix(psVer, "7.") {
				return true
			}
		}
		// 默认禁用（兼容老 cmd.exe）
		return false
	default:
		// macOS / Linux 默认启用
		return true
	}
}

// setupSignalHandler 跨平台信号处理
func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	// Windows 支持 os.Interrupt 和 syscall.SIGINT，SIGTERM 在 Windows 上有限
	// Unix 三者都支持
	switch runtime.GOOS {
	case "windows":
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	default:
		signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	}
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\n[!] 中断信号，正在退出...")
		cancel()
	}()
}

func main() {
	var (
		file  = flag.String("f", "", "目标文件(每行一个URL)")
		url   = flag.String("u", "", "单个URL")
		proxy = flag.String("proxy", "", "代理(http://host:port)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"探海 v1.0 - 资产存活探测工具\n\n"+
				"用法: %s -f <文件> | -u <URL> [-proxy 代理]\n"+
				"      cat urls.txt | %s\n\n"+
				"支持格式: http://t.com/web | https://t.com:8080 | t.com | t.ico/ | 192.168.1.1:80\n\n"+
				"平台: %s/%s\n\n选项:\n",
			os.Args[0], os.Args[0], runtime.GOOS, runtime.GOARCH)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr,
			"\n示例:\n"+
				"  %s -f urls.txt\n"+
				"  %s -u https://target.com\n"+
				"  echo tager.com | %s\n"+
				"  %s -f urls.txt -proxy http://127.0.0.1:8080\n\n"+
				"输出: 自动保存 探海报告-<时间戳>.json 和 .csv 到当前目录\n",
			os.Args[0], os.Args[0], os.Args[0], os.Args[0])
	}
	flag.Parse()

	targets := loadTargets(*file, *url)
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "[-] 未提供目标。使用 -f 或 -u 参数，或通过管道输入")
		flag.Usage()
		os.Exit(1)
	}

	workers := autoWorkers(len(targets))
	cfg := ProbeConfig{
		Timeout:          5 * time.Second,
		Workers:          workers,
		UserAgent:        defaultUA,
		InsecureTLS:      true,
		Proxy:            *proxy,
		EnableColor:      shouldEnableColor(),
		BodyLimit:        64 * 1024, // 64KB（足够提取 title）
		IdleConnsPerHost: 100,       // 单 host 100 空闲连接（vs 默认 10）
		MaxIdleConns:     500,       // 全局 500 空闲连接
	}

	fmt.Fprintf(os.Stderr, "[*] 探海 v1.0 | 平台 %s/%s | 目标 %d | 自适应并发 %d\n",
		runtime.GOOS, runtime.GOARCH, len(targets), workers)
	if cfg.Proxy != "" {
		fmt.Fprintf(os.Stderr, "[*] 使用代理: %s\n", cfg.Proxy)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	startTime := time.Now()
	results := runProbe(ctx, targets, cfg)
	elapsed := time.Since(startTime)

	// 实时输出
	for _, r := range results {
		printResult(r, cfg.EnableColor)
	}

	// 自动保存 JSON + CSV（中文文件名，Windows 兼容）
	timestamp := time.Now().Format("20060102-150405")
	jsonFile := fmt.Sprintf("探海报告-%s.json", timestamp)
	csvFile := fmt.Sprintf("探海报告-%s.csv", timestamp)

	if err := saveJSON(results, jsonFile); err != nil {
		fmt.Fprintf(os.Stderr, "[-] 保存JSON失败: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "[*] 已保存: %s\n", jsonFile)
	}
	if err := saveCSV(results, csvFile); err != nil {
		fmt.Fprintf(os.Stderr, "[-] 保存CSV失败: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "[*] 已保存: %s\n", csvFile)
	}

	// 统计
	onlineCount := 0
	statusStats := make(map[int]int)
	typeCount := make(map[string]string)
	for _, r := range results {
		if r.Online {
			onlineCount++
			statusStats[r.StatusCode]++
		} else {
			typeCount[r.ErrorType] = r.ErrorDesc
		}
	}

	fmt.Fprintf(os.Stderr, "\n========== 探测统计 ==========\n")
	fmt.Fprintf(os.Stderr, "[*] 总数 %d | 在线 %d | 离线 %d\n",
		len(results), onlineCount, len(results)-onlineCount)
	fmt.Fprintf(os.Stderr, "[*] 耗时 %v | 速度 %.1f/秒\n",
		elapsed, float64(len(results))/elapsed.Seconds())

	if len(statusStats) > 0 {
		fmt.Fprintf(os.Stderr, "[*] 状态码:\n")
		keys := make([]int, 0, len(statusStats))
		for k := range statusStats {
			keys = append(keys, k)
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[i] > keys[j] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		for _, code := range keys {
			fmt.Fprintf(os.Stderr, "    %d %s: %d\n", code, statusCodeCN[code], statusStats[code])
		}
	}
	if len(typeCount) > 0 {
		fmt.Fprintf(os.Stderr, "[*] 错误:\n")
		for et, desc := range typeCount {
			c := 0
			for _, r := range results {
				if !r.Online && r.ErrorType == et {
					c++
				}
			}
			if c > 0 {
				fmt.Fprintf(os.Stderr, "    %s: %d\n", desc, c)
			}
		}
	}
}
