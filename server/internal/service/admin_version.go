package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ylink-backend/internal/model"
	redispkg "ylink-backend/internal/pkg/redis"
)

// 版本域 module：当前版本 + 远端清单（F20 子集）。

// versionManifest 更新源响应格式：{"version":"x.y.z","notes":"变更日志"}。
type versionManifest struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

const versionManifestCacheTTL = 10 * time.Minute

// VersionInfo GET /admin/version：返回当前版本（app.version，部署注入）；
// 配置了 update.manifest_url 时远端拉取最新版本与变更日志（3s 超时 + 10min 缓存），
// 拉取失败 latest 保持空（前端展示「无法获取」），不影响接口成功。
func (m *adminVersionModule) VersionInfo(ctx context.Context) (*model.AdminVersionResp, error) {
	current := m.cfg.App.Version
	if current == "" {
		current = "dev"
	}
	resp := &model.AdminVersionResp{Version: current}
	if m.cfg.Update.ManifestURL == "" {
		return resp, nil
	}
	cacheKey := redispkg.Key("cache", "version", "manifest")
	var manifest versionManifest
	if raw, err := m.rdb.Get(ctx, cacheKey).Result(); err == nil && json.Unmarshal([]byte(raw), &manifest) == nil {
		fillVersion(resp, current, manifest)
		return resp, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.Update.ManifestURL, nil)
	if err != nil {
		return resp, nil
	}
	client := &http.Client{Timeout: 3 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		return resp, nil
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return resp, nil
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&manifest); err != nil || manifest.Version == "" {
		return resp, nil
	}
	if raw, err := json.Marshal(manifest); err == nil {
		m.rdb.Set(ctx, cacheKey, string(raw), versionManifestCacheTTL)
	}
	fillVersion(resp, current, manifest)
	return resp, nil
}

func fillVersion(resp *model.AdminVersionResp, current string, manifest versionManifest) {
	latest := manifest.Version
	resp.Latest = &latest
	up := compareVersions(manifest.Version, current) > 0
	resp.HasUpdate = &up
	if manifest.Notes != "" {
		notes := manifest.Notes
		resp.Notes = &notes
	}
}

// compareVersions 语义化版本比较（数值段逐位比较，段数不足按 0 处理）；非数字段按字符串比较兜底。
func compareVersions(a, b string) int {
	as := strings.Split(strings.TrimPrefix(a, "v"), ".")
	bs := strings.Split(strings.TrimPrefix(b, "v"), ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		av, bv := 0, 0
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}
	return 0
}
