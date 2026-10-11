#!/usr/bin/env node
/**
 * 管理端契约一致性校验(ADR-002,串入 pnpm lint:本地与 CI PR 门禁同时覆盖)。
 *
 * 两层,任一失败即 exit 1:
 *   1. 新鲜度:DTO 重跑的发射物须与入库三文件字节一致
 *      (改 DTO 未跑 pnpm contract:gen / 手改生成文件都会红)。
 *   2. 审计:api.d.ts 管理端接口 vs DTO(字段存在 + 核心类型 + 必填 + 扩展登记 + 孤儿拦截)。
 */
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  DTO_PATH,
  GEN_DTS_PATH,
  GEN_SKELETON_PATH,
  GEN_SNAPSHOT_PATH,
  audit,
  emitDTS,
  emitSnapshot,
  emitSkeletons,
  parseGoDTOs,
  sha12,
} from './admin-contract-lib.mjs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const read = (p) => readFileSync(resolve(root, p), 'utf8')

const dto = read(DTO_PATH)
const structs = parseGoDTOs(dto)
const hash = sha12(dto)

const problems = []
for (const [path, expected] of [
  [GEN_DTS_PATH, emitDTS(structs, hash)],
  [GEN_SKELETON_PATH, emitSkeletons(structs, hash)],
  [GEN_SNAPSHOT_PATH, emitSnapshot(structs, dto)],
]) {
  let actual
  try {
    actual = read(path)
  } catch {
    problems.push(`缺生成文件:${path}(跑 pnpm contract:gen)`)
    continue
  }
  if (actual !== expected)
    problems.push(`新鲜度漂移:${path}(DTO 或生成文件被改,跑 pnpm contract:gen 同步)`)
}

for (const p of audit(read('src/types/api.d.ts'), structs)) problems.push(p)

if (problems.length > 0) {
  console.error('check-admin-contract: 契约漂移:')
  for (const p of problems) console.error(`  - ${p}`)
  process.exit(1)
}
console.log(
  `check-admin-contract: OK(${structs.length} DTO structs,三发射物新鲜,api.d.ts 管理端接口一致)`,
)
