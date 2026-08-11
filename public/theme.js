/* eslint-disable */
// 防闪烁(FOUC):渲染前读取持久化主题并写入 data-theme。
// 独立文件以兼容 CSP script-src 'self'(内联脚本会被 WebView 拦截)。
// 注意:storage 写入的是 JSON 字符串(如 "dark"),JSON.parse 后即为模式本身。
;(function () {
  try {
    var raw = localStorage.getItem('app:theme')
    var theme = raw ? JSON.parse(raw) : null
    if (!theme || theme === 'system') {
      var dark = window.matchMedia('(prefers-color-scheme: dark)').matches
      theme = theme === 'system' ? (dark ? 'dark' : 'light') : dark ? 'dark' : 'light'
    }
    document.documentElement.setAttribute('data-theme', theme === 'dark' ? 'dark' : 'light')
  } catch {
    document.documentElement.setAttribute('data-theme', 'light')
  }
})()
