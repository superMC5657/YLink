#!/usr/bin/env node
/**
 * 管理端契约生成:DTO → TS 片段 + mock 骨架 + 机器快照(幂等,复跑字节一致)。
 * 用法:pnpm contract:gen | node scripts/gen-admin-contract.mjs
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  DTO_PATH,
  GEN_DTS_PATH,
  GEN_SKELETON_PATH,
  GEN_SNAPSHOT_PATH,
  emitDTS,
  emitSnapshot,
  emitSkeletons,
  parseGoDTOs,
  sha12,
} from './admin-contract-lib.mjs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const dto = readFileSync(resolve(root, DTO_PATH), 'utf8')
const structs = parseGoDTOs(dto)
const hash = sha12(dto)

writeFileSync(resolve(root, GEN_DTS_PATH), emitDTS(structs, hash))
writeFileSync(resolve(root, GEN_SKELETON_PATH), emitSkeletons(structs, hash))
writeFileSync(resolve(root, GEN_SNAPSHOT_PATH), emitSnapshot(structs, dto))
console.log(
  `contract:gen OK(${structs.length} structs, sha ${hash} → ${GEN_DTS_PATH} / ${GEN_SKELETON_PATH} / ${GEN_SNAPSHOT_PATH})`,
)
