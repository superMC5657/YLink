package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUniProxyConfig_VLESSReality(t *testing.T) {
	srv := &Server{
		ID:   1,
		Name: "HK-VLESS-Reality",
		Type: "vless",
		Host: "hk01.example.com",
		Port: 443,
		Config: `{
			"network": "tcp",
			"security": "reality",
			"sni": "hk01.example.com",
			"flow": "xtls-rprx-vision",
			"reality_settings": {
				"show": false,
				"dest": "www.apple.com:443",
				"server_names": ["www.apple.com"],
				"private_key": "private-key-test",
				"short_ids": ["0123456789abcdef"]
			},
			"push_interval": 30,
			"pull_interval": 45
		}`,
	}

	cfg, err := ParseUniProxyConfig(srv)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 443, cfg.ServerPort)
	assert.Equal(t, "tcp", cfg.Network)
	assert.Equal(t, "vless", cfg.Protocol)
	assert.Equal(t, "xtls-rprx-vision", cfg.Flow)
	assert.Equal(t, 1, cfg.TLS)
	assert.Equal(t, 30, cfg.BaseConfig.PushInterval)
	assert.Equal(t, 45, cfg.BaseConfig.PullInterval)

	require.NotNil(t, cfg.TLSSettings)
	assert.Equal(t, "hk01.example.com", cfg.TLSSettings.ServerName)
	assert.Equal(t, "private-key-test", cfg.TLSSettings.PrivateKey)
	assert.Equal(t, "0123456789abcdef", cfg.TLSSettings.ShortID)

	require.NotNil(t, cfg.TLSSettings.RealitySettings)
	assert.False(t, cfg.TLSSettings.RealitySettings.Show)
	assert.Equal(t, "www.apple.com:443", cfg.TLSSettings.RealitySettings.Dest)
	assert.Equal(t, []string{"www.apple.com"}, cfg.TLSSettings.RealitySettings.ServerNames)
	assert.Equal(t, "private-key-test", cfg.TLSSettings.RealitySettings.PrivateKey)
	assert.Equal(t, []string{"0123456789abcdef"}, cfg.TLSSettings.RealitySettings.ShortIDs)

	// 双信封测试
	resp := NewUniProxyConfigResp(*cfg)
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	// 验证顶层字段与 data 嵌套字段共存
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, float64(443), raw["server_port"])
	assert.Equal(t, "vless", raw["protocol"])
	assert.Contains(t, raw, "data")

	dataMap, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(443), dataMap["server_port"])
	assert.Equal(t, "vless", dataMap["protocol"])
}

func TestParseUniProxyConfig_Hysteria2(t *testing.T) {
	srv := &Server{
		ID:   2,
		Name: "US-Hysteria2",
		Type: "hysteria2",
		Host: "us01.example.com",
		Port: 8443,
		Config: `{
			"sni": "us01.example.com",
			"up_mbps": 100,
			"down_mbps": 500,
			"obfs": "mypassword"
		}`,
	}

	cfg, err := ParseUniProxyConfig(srv)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 8443, cfg.ServerPort)
	assert.Equal(t, "udp", cfg.Network)
	assert.Equal(t, "hysteria2", cfg.Protocol)
	assert.Equal(t, 1, cfg.TLS)
	assert.Equal(t, 100, cfg.UpMbps)
	assert.Equal(t, 500, cfg.DownMbps)
	assert.Equal(t, "mypassword", cfg.Obfs)
	assert.Equal(t, "us01.example.com", cfg.ServerName)
	assert.Equal(t, 60, cfg.BaseConfig.PushInterval)
}

func TestParseUniProxyConfig_Shadowsocks(t *testing.T) {
	srv := &Server{
		ID:   3,
		Name: "JP-SS",
		Type: "shadowsocks",
		Host: "jp01.example.com",
		Port: 8388,
		Config: `{
			"cipher": "2022-blake3-aes-128-gcm"
		}`,
	}

	cfg, err := ParseUniProxyConfig(srv)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 8388, cfg.ServerPort)
	assert.Equal(t, "tcp", cfg.Network)
	assert.Equal(t, "shadowsocks", cfg.Protocol)
	assert.Equal(t, "2022-blake3-aes-128-gcm", cfg.Cipher)
	assert.Equal(t, 0, cfg.TLS)
}

func TestParseUniProxyConfig_Trojan(t *testing.T) {
	srv := &Server{
		ID:   4,
		Name: "SG-Trojan",
		Type: "trojan",
		Host: "sg01.example.com",
		Port: 443,
		Config: `{
			"sni": "sg.example.com",
			"network": "ws",
			"path": "/trojan-ws"
		}`,
	}

	cfg, err := ParseUniProxyConfig(srv)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 443, cfg.ServerPort)
	assert.Equal(t, "ws", cfg.Network)
	assert.Equal(t, "trojan", cfg.Protocol)
	assert.Equal(t, 1, cfg.TLS)
	assert.Equal(t, "sg.example.com", cfg.ServerName)
	assert.Equal(t, "/trojan-ws", cfg.NetworkSettings["path"])
}

func TestUniProxyUserResp_DualEnvelope(t *testing.T) {
	users := []UniProxyUserItem{
		{
			ID:          1001,
			UUID:        "uuid-1001",
			SpeedLimit:  100,
			DeviceLimit: 2,
		},
		{
			ID:          1002,
			UUID:        "uuid-1002",
			SpeedLimit:  0,
			DeviceLimit: 0,
		},
	}

	resp := NewUniProxyUserResp(users)
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	// 顶层 users 供 V2bX 直接读取
	topUsers, ok := raw["users"].([]any)
	require.True(t, ok)
	assert.Len(t, topUsers, 2)

	// data.users 供标准信封读取
	dataObj, ok := raw["data"].(map[string]any)
	require.True(t, ok)
	nestedUsers, ok := dataObj["users"].([]any)
	require.True(t, ok)
	assert.Len(t, nestedUsers, 2)
}

func TestUniProxyPushItem_Unmarshal(t *testing.T) {
	jsonStr := `[
		{"user_id": 1001, "u": 1024, "d": 2048},
		{"user_id": 1002, "u": 4096, "d": 8192}
	]`

	var items []UniProxyPushItem
	err := json.Unmarshal([]byte(jsonStr), &items)
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, int64(1001), items[0].UserID)
	assert.Equal(t, int64(1024), items[0].U)
	assert.Equal(t, int64(2048), items[0].D)
}

func TestParseUniProxyPushTraffic(t *testing.T) {
	// 1. Array 格式
	arrayJSON := `[
		{"user_id": 101, "u": 100, "d": 200},
		{"user_id": 102, "u": 300, "d": 400}
	]`
	items, err := ParseUniProxyPushTraffic([]byte(arrayJSON))
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, int64(101), items[0].UserID)
	assert.Equal(t, int64(100), items[0].U)
	assert.Equal(t, int64(200), items[0].D)

	// 2. Map 格式
	mapJSON := `{"101": [100, 200], "102": [300, 400]}`
	items, err = ParseUniProxyPushTraffic([]byte(mapJSON))
	require.NoError(t, err)
	assert.Len(t, items, 2)
	uMap := make(map[int64]UniProxyPushItem)
	for _, it := range items {
		uMap[it.UserID] = it
	}
	assert.Equal(t, int64(100), uMap[101].U)
	assert.Equal(t, int64(200), uMap[101].D)
	assert.Equal(t, int64(300), uMap[102].U)
	assert.Equal(t, int64(400), uMap[102].D)

	// 3. 空报文
	items, err = ParseUniProxyPushTraffic([]byte("   \n\t"))
	require.NoError(t, err)
	assert.Empty(t, items)

	// 4. 异常格式
	_, err = ParseUniProxyPushTraffic([]byte("hello world"))
	assert.Error(t, err)

	// 5. 非法 JSON Array
	_, err = ParseUniProxyPushTraffic([]byte("[invalid json"))
	assert.Error(t, err)

	// 6. 非法 JSON Map
	_, err = ParseUniProxyPushTraffic([]byte("{invalid map"))
	assert.Error(t, err)
}
