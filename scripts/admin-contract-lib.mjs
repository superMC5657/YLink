#!/usr/bin/env node
/**
 * 管理端契约 single-owner 工具库(ADR-002)。
 *
 * Owner:server/internal/model/dto_admin.go(形状) + docs/api/README.md §16(语义)。
 * 本库只做三件事,不做判断外的新事实:
 *   1. 解析 Go DTO structs(json 名/Go 类型/binding/注释)。
 *   2. 按确定性规则发射 TS 片段 / mock 骨架 / 机器快照(纯函数,输入不变则字节不变)。
 *   3. 审计 `src/types/api.d.ts` 管理端接口与 DTO 的一致性(字段存在 + 核心类型 + 必填)。
 *
 * 有意为之的宽松(见 ADR-002):
 *   - swagger-JSON 路线未采用:管理端注解覆盖不全且 swagger.json 已滞后,
 *     DTO 直解析无工具链依赖、可复现。
 *   - TS 侧 `| null` / `?` 不做硬断言:Go nil slice 即 null、PageOK 兜底、
 *     请求侧省略语义都使空值策略偏前端健壮性,硬卡只会逼出豁免表。
 *     卡的是加/删/改名/改核心类型四类真漂移。
 */
import { createHash } from 'node:crypto'

export const DTO_PATH = 'server/internal/model/dto_admin.go'
export const GEN_DTS_PATH = 'src/types/admin.generated.d.ts'
export const GEN_SKELETON_PATH = 'mock/admin.skeletons.generated.ts'
export const GEN_SNAPSHOT_PATH = 'docs/api/admin-contract.json'

const GO_SCALARS = new Map([
  ['int', 'number'],
  ['int8', 'number'],
  ['int16', 'number'],
  ['int32', 'number'],
  ['int64', 'number'],
  ['uint', 'number'],
  ['uint8', 'number'],
  ['uint16', 'number'],
  ['uint32', 'number'],
  ['uint64', 'number'],
  ['float32', 'number'],
  ['float64', 'number'],
  ['string', 'string'],
  ['bool', 'boolean'],
  ['byte', 'number'],
  ['rune', 'number'],
])

/** TS 别名 → 核心类型(审计用;api.d.ts 新增被审计接口用到的别名时同步加表)。 */
export const TS_ALIAS_CORE = {
  AdminRole: 'number',
  TicketLevel: 'number',
  TicketStatus: 'number',
  OrderStatus: 'number',
  ServerStatus: 'number',
  AdminApplyStatus: 'number',
  CommissionStatus: 'number',
  AdminBatchAction: 'string',
  AdminBatchServerAction: 'string',
  TrafficResetMode: 'string',
}

/**
 * TS 接口 → Go struct 映射。绝大多数同名;改名 4 组是历史命名差
 * (View/Resp 后缀统一为 Item),形状一致,审计按 Go 名取 DTO。
 */
const SAME_NAME = [
  'AdminOverviewResp',
  'NodeSystemStats',
  'AdminUserItem',
  'AdminUpdateUserReq',
  'AdjustBalanceReq',
  'AdminPlanReq',
  'AdminServerGroupReq',
  'AdminServerReq',
  'AdminOrderItem',
  'RefundReq',
  'AdminCouponReq',
  'AdminNoticeItem',
  'AdminKnowledgeItem',
  'AdminNoticeReq',
  'AdminKnowledgeReq',
  'AdminTicketItem',
  'AdminReplyReq',
  'AdminAgentApplyItem',
  'AdminApproveReq',
  'AdminCommissionItem',
  'TrafficImportItem',
  'TrafficImportReq',
  'AdminSettingsReq',
  'AdminAuditLogItem',
  'AdminBatchServerReq',
  'AdminBatchServerResp',
  'AdminSortServerReq',
  'AdminSortItem',
  'AdminTrafficResetReq',
  'AdminTrafficResetResp',
  'AdminTrafficResetLogItem',
  'AdminStatOrderPoint',
  'AdminStatOrdersResp',
  'AdminStatUserPoint',
  'AdminStatPlanSlice',
  'AdminStatUsersResp',
  'AdminStatUserTraffic',
  'AdminStatNodeTraffic',
  'AdminStatTrafficResp',
  'AdminBatchUserReq',
  'AdminBatchFailedItem',
  'AdminBatchUserResp',
  'AdminSendMailReq',
  'AdminSendMailResp',
  'AdminSortReq',
  'AdminKnowledgeCategoryItem',
  'AdminKnowledgeCategoryReq',
  'AdminKnowledgeCategoryUpdateReq',
  'AdminMailTemplateItem',
  'AdminMailTemplateReq',
  'AdminMailTemplateTestReq',
  'AdminVersionResp',
  'AdminSubscriptionTemplateItem',
  'AdminSubscriptionTemplateReq',
  'AdminSubscriptionTemplatePreviewResp',
  'TelegramBindCodeResp',
  'AdminTelegramWebhookSetupResp',
]
const RENAMED = {
  AdminPlanItem: 'AdminPlanView',
  AdminServerItem: 'AdminServerView',
  AdminCouponItem: 'AdminCouponView',
  AdminSettingsItem: 'AdminSettingsResp',
}
export const MAPPING = new Map([...SAME_NAME.map((n) => [n, n]), ...Object.entries(RENAMED)])

/** TS 比 DTO 多的字段须逐项登记理由;未登记=漂移,失败。 */
export const EXTENSIONS = new Map([
  [
    'AdminKnowledgeItem',
    new Map([
      [
        'category_id',
        '前端 F15 分类筛选回填位:列表场景需要归属分类 ID,DTO Item 暂未展开,仅可选扩展',
      ],
    ]),
  ],
])

/**
 * 无 dto_admin.go 主人的管理端相关接口(审计不断言形状,只断言存在防表腐烂)。
 * - AdminServerGroupItem:线型=表结构 model.ServerGroup(models.go)。
 * - AdminTicketDetail:线型=model.TicketDetailResp(dto_ticket.go,用户/管理共用)。
 * - AdminAuditLogsResp:分页包 + actions 聚合的组合体,无独立 DTO。
 * - NodeSettings:settings 表 node 键 JSON 的客户端解析形,非线型。
 * - TicketReplyReq:AdminReplyReq 的历史重名,同形 {message}。
 */
export const ADHOC = new Map([
  ['AdminServerGroupItem', '线型=表结构 model.ServerGroup'],
  ['AdminTicketDetail', '线型=model.TicketDetailResp(用户/管理共用,后继折叠)'],
  ['AdminAuditLogsResp', '分页包 + actions 组合,无独立 DTO'],
  ['NodeSettings', 'settings 表 node 键的客户端解析形,非线型'],
  ['TicketReplyReq', 'AdminReplyReq 历史重名,同形'],
])

/** 非 Admin 前缀但归属管理端契约的接口名(孤儿扫描用)。 */
const RELATED = new Set([
  'NodeSystemStats',
  'TrafficImportItem',
  'TrafficImportReq',
  'RefundReq',
  'AdjustBalanceReq',
  'TicketReplyReq',
  'NodeSettings',
  'TelegramBindCodeResp',
])

// ---------- Go DTO 解析 ----------

const STRUCT_RE = /^type (\w+) struct \{$/
const FIELD_RE = /^\s*([A-Za-z_]\w*)\s+(\S+)\s+`([^`]*)`(?:\s*\/\/(.*))?$/

function parseTags(raw) {
  const out = {}
  for (const m of raw.matchAll(/(\w+):"([^"]*)"/g)) out[m[1]] = m[2]
  return out
}

/**
 * 解析 dto_admin.go → [{name, doc, fields:[{field, go, json, omitEmpty, binding, required, comment}]}]。
 * 只认顶层 `type X struct` 与单行字段;文件无内嵌/多行字段,出现即抛错(防静默漏解析)。
 */
export function parseGoDTOs(source) {
  const structs = []
  const lines = source.split('\n')
  let pendingDoc = []
  let cur = null
  let pendingFieldDoc = []
  lines.forEach((line, idx) => {
    const no = idx + 1
    const sm = line.match(STRUCT_RE)
    if (sm) {
      cur = { name: sm[1], doc: pendingDoc.join('\n'), fields: [] }
      structs.push(cur)
      pendingDoc = []
      return
    }
    if (cur) {
      if (/^\}/.test(line)) {
        cur = null
        pendingFieldDoc = []
        return
      }
      if (/^\s*$/.test(line)) return
      const cmIn = line.match(/^\s*\/\/(.*)$/)
      if (cmIn) {
        pendingFieldDoc.push(cmIn[1].trim())
        return
      }
      const fm = line.match(FIELD_RE)
      if (!fm) throw new Error(`DTO 解析失败 ${DTO_PATH}:${no}: ${line.trim()}`)
      const tags = parseTags(fm[3])
      if (!('json' in tags)) throw new Error(`DTO 字段缺 json tag ${DTO_PATH}:${no}`)
      const [jsonName, ...flags] = tags.json.split(',')
      if (!jsonName || jsonName === '-') return
      const binding = tags.binding ?? ''
      const inline = (fm[4] ?? '').trim()
      const comment = [...pendingFieldDoc, ...(inline ? [inline] : [])].join(' ')
      pendingFieldDoc = []
      cur.fields.push({
        field: fm[1],
        go: fm[2],
        json: jsonName,
        omitEmpty: flags.includes('omitempty'),
        binding,
        required: /(?:^|,)required(?:,|$)/.test(binding),
        comment,
      })
      return
    }
    const cm = line.match(/^\s*\/\/(.*)$/)
    if (cm) pendingDoc.push(cm[1].trim())
    else if (/^\s*$/.test(line)) pendingDoc = []
    else if (/^package |^import |^\)/.test(line)) pendingDoc = []
    else if (line.trim() && !line.startsWith('import')) pendingDoc = []
  })
  if (structs.length === 0) throw new Error('DTO 解析结果为空,检查文件格式')
  return structs
}

/** Go 类型 → {core(TS 核心类型,含 [] 后缀), nullable}。未知类型抛错。 */
export function goToTS(go) {
  let nullable = false
  let t = go
  while (t.startsWith('*')) {
    nullable = true
    t = t.slice(1)
  }
  if (t.startsWith('[]')) {
    const inner = goToTS(t.slice(2))
    return { core: `${inner.core}[]`, nullable: true }
  }
  if (t === 'time.Time') return { core: 'string', nullable }
  if (t === 'time.Duration') return { core: 'number', nullable }
  if (GO_SCALARS.has(t)) return { core: GO_SCALARS.get(t), nullable }
  if (t.startsWith('map[')) {
    const vm = t.match(/^map\[[^\]]+\](.+)$/)
    if (!vm) throw new Error(`无法映射 Go 类型: ${go}`)
    return { core: `Record<string, ${goToTS(vm[1]).core}>`, nullable: true }
  }
  if (/^[A-Za-z_]\w*$/.test(t)) return { core: t, nullable }
  throw new Error(`无法映射 Go 类型: ${go}`)
}

export function sha12(text) {
  return createHash('sha256').update(text).digest('hex').slice(0, 12)
}

export function shaFull(text) {
  return createHash('sha256').update(text).digest('hex')
}

// ---------- 发射物 ----------

function dtsHeader(hash) {
  return `/**
 * 管理端契约生成片段 —— DO NOT EDIT,改 Go DTO 后跑 pnpm contract:gen。
 * Owner:server/internal/model/dto_admin.go(形状) + docs/api/README.md §16(语义)。
 * 消费:新管理端类型直接引此文件;src/types/api.d.ts 存量接口由
 * scripts/check-admin-contract.mjs 逐字段审计(决策见 ADR-0002)。
 * source sha256:${hash}
 */
`
}

function skeletonHeader(hash) {
  return `/**
 * 管理端 mock 骨架 —— DO NOT EDIT,改 Go DTO 后跑 pnpm contract:gen。
 * 全零/空占位,不含任何业务语义;mock/admin.ts 以 {...骨架, 手填值} 消费,
 * DTO 新增字段自动带零值,语义值仍手填并以 HAND-VALUE 标记(决策见 ADR-002)。
 * source sha256:${hash}
 */
`
}

/** 零值:number 0 / string '' / boolean false / 数组 [] / 可空与对象引用 null。 */
function zeroValue(core, nullable) {
  if (nullable) return 'null'
  if (core === 'number') return '0'
  if (core === 'string') return "''"
  if (core === 'boolean') return 'false'
  if (core.endsWith('[]')) return '[]'
  if (core.startsWith('Record<')) return '{}'
  return 'null'
}

export function emitDTS(structs, hash) {
  const out = [dtsHeader(hash)]
  for (const s of structs) {
    if (s.doc) for (const dl of s.doc.split('\n')) out.push(`// ${dl}`)
    out.push(`export interface ${s.name} {`)
    for (const f of s.fields) {
      const { core, nullable } = goToTS(f.go)
      const opt = f.omitEmpty ? '?' : ''
      const tail = f.comment ? ` // ${f.comment}` : ''
      out.push(`  ${f.json}${opt}: ${core}${nullable ? ' | null' : ''}${tail}`)
    }
    out.push('}')
    out.push('')
  }
  return `${out.join('\n').trimEnd()}\n`
}

export function emitSkeletons(structs, hash) {
  const out = [skeletonHeader(hash), 'export const AdminSkeletons = {']
  for (const s of structs) {
    out.push(`  ${s.name}: {`)
    for (const f of s.fields) {
      const { core, nullable } = goToTS(f.go)
      out.push(`    ${f.json}: ${zeroValue(core, nullable)},`)
    }
    out.push('  },')
  }
  out.push('}')
  out.push('')
  return `${out.join('\n').trimEnd()}\n`
}

export function emitSnapshot(structs, dtoSource) {
  const snap = {
    generated_by: 'node scripts/gen-admin-contract.mjs',
    owner: ['server/internal/model/dto_admin.go', 'docs/api/README.md §16'],
    source: { file: DTO_PATH, sha256: shaFull(dtoSource) },
    structs: {},
  }
  for (const s of structs) {
    snap.structs[s.name] = {
      fields: s.fields.map((f) => {
        const { core, nullable } = goToTS(f.go)
        return {
          field: f.field,
          json: f.json,
          go: f.go,
          ts: `${core}${nullable ? ' | null' : ''}`,
          optional: f.omitEmpty,
          nullable,
          binding: f.binding,
          required: f.required,
        }
      }),
    }
  }
  return `${JSON.stringify(snap, null, 2)}\n`
}

// ---------- api.d.ts 解析与审计 ----------

/** 花括号配平扫描抽取 `export interface Name { ... }`(体含泛型 <> 与注释均可)。 */
export function parseTSInterfaces(source) {
  const out = new Map()
  const key = 'export interface '
  let i = 0
  while (true) {
    const k = source.indexOf(key, i)
    if (k < 0) break
    const nm = source.slice(k + key.length).match(/^\w+/)
    if (!nm) throw new Error(`TS 解析失败:offset ${k}`)
    let j = k + key.length + nm[0].length
    const open = source.indexOf('{', j)
    if (open < 0) throw new Error(`TS 接口 ${nm[0]} 缺主体`)
    let depth = 0
    let p = open
    for (; p < source.length; p++) {
      if (source[p] === '{') depth++
      else if (source[p] === '}') {
        depth--
        if (depth === 0) break
      }
    }
    if (depth !== 0) throw new Error(`TS 接口 ${nm[0]} 花括号未配平`)
    const body = source
      .slice(open + 1, p)
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .split('\n')
    const fields = []
    for (const raw of body) {
      const line = raw
        .replace(/(^|\s)\/\/.*$/, '')
        .replace(/[,;]\s*$/, '')
        .trim()
      if (!line) continue
      const fm = line.match(/^(\w+)(\?)?:\s*(.+)$/)
      if (!fm) throw new Error(`TS 字段解析失败 ${nm[0]}: ${line}`)
      fields.push({ field: fm[1], optional: fm[2] === '?', type: fm[3].trim() })
    }
    out.set(nm[0], fields)
    i = p + 1
  }
  return out
}

/** TS 类型 → 核心类型(剥 null/undefined,字面量并归纳,别名查表,DTO 本地引用透传)。 */
export function tsCore(t, goNames) {
  const parts = t
    .replace(/\s+/g, '')
    .split('|')
    .filter((s) => s && s !== 'null' && s !== 'undefined')
  const cores = parts.map((b) => branchCore(b, goNames))
  const uniq = [...new Set(cores)]
  if (uniq.length !== 1) throw new Error(`TS 联合类型无法归纳: ${t}`)
  return uniq[0]
}

function branchCore(b, goNames) {
  if (/^'[^']*'$/.test(b) || /^"[^"]*"$/.test(b)) return 'string'
  if (/^\d+(\.\d+)?$/.test(b)) return 'number'
  if (b === 'true' || b === 'false') return 'boolean'
  if (b === 'number' || b === 'string' || b === 'boolean') return b
  const arr = b.match(/^Array<(.+)>$/)
  if (arr) return `${branchCore(arr[1], goNames)}[]`
  if (b.endsWith('[]')) return `${branchCore(b.slice(0, -2), goNames)}[]`
  if (TS_ALIAS_CORE[b]) return TS_ALIAS_CORE[b]
  if (goNames.has(b)) return b
  throw new Error(`TS 类型未收录(加别名表或 DTO 引用): ${b}`)
}

/**
 * 审计 api.d.ts。返回问题列表(空=通过)。
 * 规则:①Go 字段须在 TS 同名存在 ②核心类型相等 ③Go binding required 在 TS 不得可选
 * ④TS 多余字段须 EXTENSIONS 登记 ⑤孤儿接口(新增未登记)失败 ⑥登记表腐烂(指向不存在)失败。
 */
export function audit(tsSource, structs) {
  const problems = []
  const goByName = new Map(structs.map((s) => [s.name, s]))
  const goNames = new Set(structs.map((s) => s.name))
  const tsIfaces = parseTSInterfaces(tsSource)

  for (const [tsName, goName] of MAPPING) {
    const go = goByName.get(goName)
    if (!go) {
      problems.push(`映射表腐烂:Go 缺 ${goName}(TS ${tsName} 指向它)`)
      continue
    }
    const ts = tsIfaces.get(tsName)
    if (!ts) {
      problems.push(`映射表腐烂:api.d.ts 缺 ${tsName}`)
      continue
    }
    const tsByField = new Map(ts.map((f) => [f.field, f]))
    for (const gf of go.fields) {
      const tf = tsByField.get(gf.json)
      if (!tf) {
        problems.push(`${tsName}.${gf.json}:DTO 有而 TS 缺(先改 DTO+README 再跑 contract:gen)`)
        continue
      }
      const expected = goToTS(gf.go).core
      let actual
      try {
        actual = tsCore(tf.type, goNames)
      } catch (e) {
        problems.push(`${tsName}.${tf.field}:类型无法归纳(${tf.type}):${e.message}`)
        continue
      }
      if (actual !== expected)
        problems.push(`${tsName}.${tf.field}:核心类型漂移(TS ${actual} vs DTO ${expected})`)
      if (gf.required && tf.optional)
        problems.push(`${tsName}.${tf.field}:DTO binding required 而 TS 可选`)
    }
    const goJson = new Set(go.fields.map((f) => f.json))
    const allowedExtra = EXTENSIONS.get(tsName) ?? new Map()
    for (const tf of ts) {
      if (!goJson.has(tf.field) && !allowedExtra.has(tf.field))
        problems.push(`${tsName}.${tf.field}:TS 多余字段未在 EXTENSIONS 登记`)
    }
    for (const ex of allowedExtra.keys()) {
      if (!tsByField.has(ex)) problems.push(`EXTENSIONS 腐烂:${tsName}.${ex} 已不在 TS 中`)
    }
  }

  for (const [name, why] of ADHOC) {
    if (!tsIfaces.has(name)) problems.push(`ADHOC 腐烂:api.d.ts 缺 ${name}(${why})`)
  }
  for (const [name] of tsIfaces) {
    if (MAPPING.has(name) || ADHOC.has(name)) continue
    if (name.startsWith('Admin') || RELATED.has(name))
      problems.push(`孤儿接口:${name} 未进 MAPPING/ADHOC(新增管理端类型须先落 DTO)`)
  }
  for (const alias of Object.keys(TS_ALIAS_CORE)) {
    if (!new RegExp(`export type ${alias} =`).test(tsSource))
      problems.push(`别名表腐烂:api.d.ts 缺 export type ${alias}`)
  }
  return problems
}
