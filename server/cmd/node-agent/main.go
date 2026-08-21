// 流量模式 A 演示 agent:模拟节点侧定时上报每用户累计流量(本地全链路联调用)。
//
// 用法(先起后端与迁移 0004):
//
//	go run ./cmd/node-agent \
//	  -endpoint http://127.0.0.1:8081/api/v1 \
//	  -key <管理端节点列表的 node_key> \
//	  -interval 60s \
//	  -users 5f2b7c9e-xxxx,...
//
// 行为:启动时 GET /node/users 拉取有效订阅用户(未指定 -users 时自动采用),
// 之后每 interval 上报一次,每用户累计值按随机增量单调递增(真实 agent 应读代理后端计数器)。
// 响应中的 accepted/skipped 会打到日志,便于核对幂等与跳过原因。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
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

type reportReq struct {
	Data []reportItem `json:"data"`
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
			log.Fatalf("拉取用户失败: %v", err)
		}
		uuids = fetched
	}
	if len(uuids) == 0 {
		log.Fatal("无上报用户(检查节点分组下是否有有效订阅用户)")
	}
	log.Printf("node-agent 启动:endpoint=%s users=%d interval=%s", *endpoint, len(uuids), *interval)

	// 每用户累计计数(自 agent 启动起单调递增;重启即归零,可验证面板计数器回退逻辑)
	counters := make(map[string]reportItem, len(uuids))
	for _, id := range uuids {
		counters[id] = reportItem{UUID: id}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	tick := time.NewTicker(*interval)
	reportOnce(client, *endpoint, *key, counters, rng) // 启动先报一次(全 0,建立快照基线)
	for range tick.C {
		reportOnce(client, *endpoint, *key, counters, rng)
	}
}

// reportOnce 随机增量推进累计值并上报。
func reportOnce(client *http.Client, endpoint, key string, counters map[string]reportItem, rng *rand.Rand) {
	items := make([]reportItem, 0, len(counters))
	for id, c := range counters {
		c.U += int64(rng.Intn(50)+1) * 1024 * 1024  // 每轮 1–50 MiB 上行
		c.D += int64(rng.Intn(500)+1) * 1024 * 1024 // 每轮 1–500 MiB 下行
		counters[id] = c
		items = append(items, c)
	}
	body, _ := json.Marshal(reportReq{Data: items})
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
		return
	}
	log.Printf("上报 %d 条: accepted=%d skipped=%d", len(items), rr.Data.Accepted, len(rr.Data.Skipped))
	for _, sk := range rr.Data.Skipped {
		log.Printf("  跳过 %s: %s", sk.UUID, sk.Reason)
	}
}

// fetchUsers GET /node/users 拉取有效订阅用户 uuid。
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
