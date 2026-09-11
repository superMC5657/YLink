// 流量模式 A 生产/演示 Node Agent: 负责向 YLink 节点接口上报用户流量与系统硬件探针(CPU/内存/网络/连接数/Uptime)。
//
// 用法:
//
//	go run ./cmd/node-agent \
//	  -endpoint http://127.0.0.1:8081/api/v1 \
//	  -key <管理端节点列表的 node_key> \
//	  -interval 60s
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type nodeUsersResp struct {
	Code int `json:"code"`
	Data struct {
		Rate  float64 `json:"rate"`
		Users []struct {
			UUID           string `json:"uuid"`
			U              int64  `json:"u"`
			D              int64  `json:"d"`
			TransferEnable int64  `json:"transfer_enable"`
		} `json:"users"`
	} `json:"data"`
}

type nodeSystemStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	NetSpeedIn    int64   `json:"net_speed_in"`  // 瞬时下行带宽(B/s)
	NetSpeedOut   int64   `json:"net_speed_out"` // 瞬时上行带宽(B/s)
	Connections   int     `json:"connections"`   // TCP 连接数
	Uptime        int64   `json:"uptime"`        // 开机时长(秒)
}

type reportReq struct {
	Data   []reportItem     `json:"data"`
	System *nodeSystemStats `json:"system,omitempty"`
}

type reportItem struct {
	UUID string `json:"uuid"`
	U    int64  `json:"u"`
	D    int64  `json:"d"`
}

type reportResp struct {
	Code int `json:"code"`
	Data struct {
		Accepted int `json:"accepted"`
		Skipped  []struct {
			UUID   string `json:"uuid"`
			Reason string `json:"reason"`
		} `json:"skipped"`
	} `json:"data"`
}

var (
	startTime     = time.Now()
	lastCPUIdle   uint64
	lastCPUTotal  uint64
	lastNetRx     uint64
	lastNetTx     uint64
	lastNetSample = time.Now()
)

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:8081/api/v1", "面板 API base(至 /api/v1)")
	key := flag.String("key", "", "节点密钥 X-Node-Key(管理端节点列表查看)")
	interval := flag.Duration("interval", 60*time.Second, "上报周期")
	usersFlag := flag.String("users", "", "逗号分隔的用户 uuid;缺省自动拉取 /node/users")
	flag.Parse()
	if *key == "" {
		log.Fatal("必须提供 -key(管理端节点列表/重置获得)")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var uuids []string
	if *usersFlag != "" {
		for _, s := range strings.Split(*usersFlag, ",") {
			if s = strings.TrimSpace(s); s != "" {
				uuids = append(uuids, s)
			}
		}
	} else {
		fetched, err := fetchUsers(client, *endpoint, *key)
		if err != nil {
			log.Printf("拉取用户提示: %v(节点将以纯探针心跳模式运行)", err)
		} else {
			uuids = fetched
		}
	}

	log.Printf("ylink-node agent 启动: endpoint=%s users=%d interval=%s", *endpoint, len(uuids), *interval)

	counters := make(map[string]reportItem, len(uuids))
	for _, id := range uuids {
		counters[id] = reportItem{UUID: id}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tick := time.NewTicker(*interval)

	// 启动先报一次
	reportOnce(client, *endpoint, *key, counters, rng)
	for range tick.C {
		// 周期性若发现无用户，尝试再次同步一次用户
		if len(counters) == 0 && *usersFlag == "" {
			if fetched, err := fetchUsers(client, *endpoint, *key); err == nil && len(fetched) > 0 {
				log.Printf("同步到 %d 名有效用户，开始计费上报", len(fetched))
				for _, id := range fetched {
					counters[id] = reportItem{UUID: id}
				}
			}
		}
		reportOnce(client, *endpoint, *key, counters, rng)
	}
}

func reportOnce(client *http.Client, endpoint, key string, counters map[string]reportItem, rng *rand.Rand) {
	items := make([]reportItem, 0, len(counters))
	for id, c := range counters {
		items = append(items, reportItem{UUID: id, U: c.U, D: c.D})
	}

	sysStats := collectSystemStats()
	reqBody := reportReq{
		Data:   items,
		System: sysStats,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPost, endpoint+"/node/report", bytes.NewReader(body))
	if err != nil {
		log.Printf("构造请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Node-Key", key)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("上报失败: %v", err)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var rr reportResp
	if err := json.Unmarshal(raw, &rr); err != nil {
		log.Printf("上报响应解析失败(%d): %s", resp.StatusCode, raw)
		return
	}
	if resp.StatusCode != http.StatusOK || rr.Code != 0 {
		log.Printf("上报被拒(HTTP %d code=%d): %s", resp.StatusCode, rr.Code, raw)
	} else {
		log.Printf("上报成功(CPU: %.1f%%, 内存: %.1f%%, 活跃连接: %d): accepted=%d skipped=%d",
			sysStats.CPUPercent, sysStats.MemoryPercent, sysStats.Connections, rr.Data.Accepted, len(rr.Data.Skipped))
		for _, sk := range rr.Data.Skipped {
			log.Printf("  跳过 %s: %s", sk.UUID, sk.Reason)
		}
	}

	// 模拟推进下一轮流量累计值(真实接入 Xray 时替换为从 Xray gRPC 读累计值)
	for id, c := range counters {
		c.U += int64(rng.Intn(50)+1) * 1024 * 1024
		c.D += int64(rng.Intn(500)+1) * 1024 * 1024
		counters[id] = c
	}
}

func fetchUsers(client *http.Client, endpoint, key string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint+"/node/users", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Node-Key", key)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}
	var ur nodeUsersResp
	if err := json.Unmarshal(raw, &ur); err != nil {
		return nil, err
	}
	if ur.Code != 0 {
		return nil, fmt.Errorf("code=%d", ur.Code)
	}
	uuids := make([]string, 0, len(ur.Data.Users))
	for _, u := range ur.Data.Users {
		uuids = append(uuids, u.UUID)
	}
	return uuids, nil
}

// collectSystemStats 采集机器指标(Linux 读 /proc，非 Linux/Windows 兜底模拟)。
func collectSystemStats() *nodeSystemStats {
	stats := &nodeSystemStats{
		Uptime: int64(time.Since(startTime).Seconds()),
	}

	if runtime.GOOS == "linux" {
		readLinuxStats(stats)
	} else {
		// Windows / macOS 本地调试 fallback
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		stats.CPUPercent = float64(rand.Intn(150)+50) / 10.0 // 5.0% ~ 20.0%
		stats.MemoryPercent = 35.5
		stats.DiskPercent = 25.0
		stats.NetSpeedIn = int64(rand.Intn(500000) + 10240)
		stats.NetSpeedOut = int64(rand.Intn(1000000) + 51200)
		stats.Connections = runtime.NumGoroutine() * 2
	}
	return stats
}

func readLinuxStats(s *nodeSystemStats) {
	// 1. CPU
	if f, err := os.Open("/proc/stat"); err == nil {
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 5 && fields[0] == "cpu" {
				var total, idle uint64
				for i := 1; i < len(fields); i++ {
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					total += val
					if i == 4 { // idle is 4th field
						idle = val
					}
				}
				if lastCPUTotal > 0 && total > lastCPUTotal {
					dTotal := total - lastCPUTotal
					dIdle := idle - lastCPUIdle
					if dTotal > 0 && dTotal >= dIdle {
						s.CPUPercent = float64(dTotal-dIdle) / float64(dTotal) * 100.0
					}
				}
				lastCPUTotal = total
				lastCPUIdle = idle
			}
		}
		_ = f.Close()
	}

	// 2. 内存
	if f, err := os.Open("/proc/meminfo"); err == nil {
		scanner := bufio.NewScanner(f)
		var memTotal, memAvailable uint64
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				memTotal = parseMemKB(line)
			} else if strings.HasPrefix(line, "MemAvailable:") {
				memAvailable = parseMemKB(line)
			}
		}
		_ = f.Close()
		if memTotal > 0 {
			used := memTotal - memAvailable
			s.MemoryPercent = float64(used) / float64(memTotal) * 100.0
		}
	}

	// 3. 网络带宽
	now := time.Now()
	if f, err := os.Open("/proc/net/dev"); err == nil {
		scanner := bufio.NewScanner(f)
		var totalRx, totalTx uint64
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "Inter-") || strings.HasPrefix(line, "face") {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				iface := strings.TrimSpace(parts[0])
				if iface == "lo" {
					continue
				}
				fields := strings.Fields(parts[1])
				if len(fields) >= 9 {
					rx, _ := strconv.ParseUint(fields[0], 10, 64)
					tx, _ := strconv.ParseUint(fields[8], 10, 64)
					totalRx += rx
					totalTx += tx
				}
			}
		}
		_ = f.Close()

		dt := now.Sub(lastNetSample).Seconds()
		if dt > 0 && lastNetRx > 0 && totalRx >= lastNetRx && totalTx >= lastNetTx {
			s.NetSpeedIn = int64(float64(totalRx-lastNetRx) / dt)
			s.NetSpeedOut = int64(float64(totalTx-lastNetTx) / dt)
		}
		lastNetRx = totalRx
		lastNetTx = totalTx
		lastNetSample = now
	}

	// 4. 连接数 (/proc/net/tcp 行数)
	connCount := 0
	for _, p := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if f, err := os.Open(p); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				connCount++
			}
			_ = f.Close()
		}
	}
	if connCount > 2 {
		s.Connections = connCount - 2
	}
}

func parseMemKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		return v
	}
	return 0
}
