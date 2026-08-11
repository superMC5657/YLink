<script setup lang="ts">
/**
 * 管理后台 · 流量导入(模式 B 手工导入):逐行录入用户流量,一次批量提交。
 * 数据:POST /admin/traffic/import(docs/api/README.md §16.1)
 */
import { reactive, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { TrafficImportItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage } from 'naive-ui'
import { formatBytes } from '@/utils/format'

const message = useMessage()

const rows = reactive<TrafficImportItem[]>([
  { user_id: 10086, date: new Date().toISOString().slice(0, 10), u: 0, d: 0 },
])

const submitting = ref(false)

function addRow() {
  rows.push({ user_id: 0, date: new Date().toISOString().slice(0, 10), u: 0, d: 0 })
}

function removeRow(index: number) {
  rows.splice(index, 1)
}

async function submit() {
  if (rows.length === 0) {
    message.warning('请至少添加一行流量数据')
    return
  }
  for (const r of rows) {
    if (!r.user_id) {
      message.warning('请填写用户 ID')
      return
    }
    if (!r.date) {
      message.warning('请选择日期')
      return
    }
  }
  submitting.value = true
  try {
    await apiAdmin.importTraffic({
      items: rows.map((r) => ({
        user_id: r.user_id,
        date: r.date,
        u: Math.max(0, Math.floor(Number(r.u) || 0)),
        d: Math.max(0, Math.floor(Number(r.d) || 0)),
      })),
    })
    message.success(`已导入 ${rows.length} 条流量记录(写入审计)`)
    rows.splice(0, rows.length, {
      user_id: 0,
      date: new Date().toISOString().slice(0, 10),
      u: 0,
      d: 0,
    })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="流量导入" subtitle="模式 B 手工导入:按用户与日期录入上行/下行流量(单位字节)">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="addRow">
            <AppIcon name="plus" :size="15" /> 添加一行
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="submitting" @click="submit">
            批量导入
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-table :bordered="false" :single-line="false" class="min-w-[760px]">
        <thead>
          <tr>
            <th>#</th>
            <th>用户 ID</th>
            <th>日期</th>
            <th>上行(字节)</th>
            <th>下行(字节)</th>
            <th>合计预览</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(r, i) in rows" :key="i">
            <td class="num-font text-[var(--c-text-sub)]">{{ i + 1 }}</td>
            <td>
              <n-input-number
                v-model:value="r.user_id"
                :min="1"
                placeholder="用户 ID"
                class="w-32"
              />
            </td>
            <td>
              <n-date-picker
                v-model:formatted-value="r.date"
                value-format="yyyy-MM-dd"
                type="date"
                class="w-36"
              />
            </td>
            <td>
              <n-input-number v-model:value="r.u" :min="0" placeholder="上行字节" class="w-44" />
            </td>
            <td>
              <n-input-number v-model:value="r.d" :min="0" placeholder="下行字节" class="w-44" />
            </td>
            <td class="num-font text-14 text-[var(--c-text-sub)]">
              {{ formatBytes(Number(r.u) + Number(r.d)) }}
            </td>
            <td>
              <button class="btn-soft-danger h-7 px-3 text-14" @click="removeRow(i)">移除</button>
            </td>
          </tr>
        </tbody>
      </n-table>

      <div
        class="mt-4 rounded-lg bg-[var(--c-warning-bg)] px-4 py-3 text-14 text-[var(--c-warning)]"
      >
        <AppIcon name="info" :size="15" class="mr-1 inline" />
        流量单位必须为字节。示例:1 GB = 1073741824 字节;1 MB = 1048576 字节。
        同一用户同一天重复导入将覆盖当日已记录的上行/下行数据。
      </div>
    </div>
  </div>
</template>
