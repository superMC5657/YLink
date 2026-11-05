# 04: [Ops & UI] V2bX 对接部署文档、安装弹窗与联调验证套件

Status: resolved
Priority: P1
Blocked by: 03

---

## 1. 任务背景
在接口开发完成后，形成节点部署运维指南，并在前端管理后台节点管理弹窗中集成 V2bX 部署配置生成与多语言支持，同时提供本地自动化验证套件。

---

## 2. 详细工作内容

1. **编写文档 `docs/backend/v2bx-deployment-guide.md`**：
   - **V2bX 安装**：提供官方一键安装脚本与 Systemd 服务管理说明；
   - **配置范例 (`/etc/V2bX/config.json`)**：正确使用 `hysteria2` 作为 Hysteria2 协议核心名称；
   - **管理后台配置指引**：如何在 YLink 管理后台新建节点、获取 Node ID 与 Node Key。

2. **管理后台部署弹窗联动与国际化**：
   - `src/components/admin/NodeInstallDialog.vue`：添加 V2bX 安装与配置标签，自动生成 V2bX 配置 JSON 与一键部署脚本，使用 `vue-i18n` 提取中英文文本，遵循语义 Tailwind 配色。
   - `src/locales/zh-CN.ts` & `src/locales/en-US.ts`：同步补齐国际化词条。

3. **自动化联调验证套件**：
   - 编写 `server/scripts/test_v2bx_e2e.sh` 模拟 V2bX 请求 `/config`、`/user` 与 `/push`，验证配置下发、有效用户下发与增量流量入账。

---

## 3. 验收标准
- [x] 部署文档详尽准确，运维人员可按照步骤 5 分钟内完成节点上线；
- [x] 前端管理弹窗 V2bX 标签支持各协议核心自动配置且单测全绿；
- [x] 自动化测试套件全链路验证闭环。
