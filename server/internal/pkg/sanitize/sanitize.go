// Package sanitize 提供 XSS 白名单清洗（安全清单第 5 条：Markdown 内容入库前 bluemonday 清洗）。
package sanitize

import "github.com/microcosm-cc/bluemonday"

// mdPolicy 宽松策略：允许常见格式化标签（b/i/strong/em/a/img/ul/ol/li/blockquote/code/pre/table 等），
// 去除 script/iframe/事件处理器/危险协议；适用于 Markdown 渲染富文本。
var mdPolicy = bluemonday.UGCPolicy()

// textPolicy 严格策略：仅保留纯文本（去全部标签）；适用于标题、工单消息等。
var textPolicy = bluemonday.StrictPolicy()

// Markdown 清洗富文本（公告/知识库/套餐 content）。
func Markdown(s string) string { return mdPolicy.Sanitize(s) }

// Text 清洗普通文本（标题、工单消息等）。
func Text(s string) string { return textPolicy.Sanitize(s) }
