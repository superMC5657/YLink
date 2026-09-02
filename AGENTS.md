# AGENTS.md

## Agent skills

### Issue tracker

Issues and specs live as markdown files under `.scratch/<feature>/` in this repo. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical roles mapped to label strings: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. 本仓暂未创建这两者，按需惰性生成（缺失时的处理约定见 `docs/agents/domain.md`）。

## Notes

- 每次完成更新后需要你将完成的工作同步到需求和过程文档里，确保用户知道哪些工作做了，哪些未完成：完成的需求在 spec 标题栏打上 ✅，不再需要的功能用删除线标识（删除线仅表示「不做」，不要挪作他用）。
- 文档只维护「当前态」，过时的修改记录从原文档删除（历史交给 git log 与 docs/reviews/）：实施细节写进 progress 与契约文档，不堆进 spec 正文；progress.md 只记录能力清单、未完成项与前置条件，禁止追加式流水账。
- 同一事实只在一处维护（接口契约在 docs/api、表结构在 data-model、面向访客的功能清单在根 README、需求状态台账在各 spec），其余位置用链接指向，不要复制内容。
