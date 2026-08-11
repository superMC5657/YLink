<script setup lang="ts">
/**
 * 管理后台 · 优惠券管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/coupons + GET /admin/plans(docs/api/README.md §16.1)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { apiAdmin } from '@/api/admin'
import type { AdminCouponItem, AdminCouponReq, AdminNoticeReq, AdminPlanItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminCouponItem[]>([])
const plans = ref<AdminPlanItem[]>([])

const PERIODS = [
  { value: 'month', label: '月付' },
  { value: 'quarter', label: '季付' },
  { value: 'half_year', label: '半年付' },
  { value: 'year', label: '年付' },
  { value: 'onetime', label: '一次性' },
]

// 新建/编辑弹窗
const modal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const form = reactive<AdminCouponReq>({
  code: '',
  type: 2,
  value: 0,
  min_spend: 0,
  limit_per_user: 0,
  total_limit: 0,
  valid_periods: [],
  plan_ids: [],
  started_at: null,
  ended_at: null,
  is_enable: true,
})

// 一键公告弹窗：为某张优惠券生成公告草稿并发布
const noticeModal = ref(false)
const noticeSaving = ref(false)
const noticeFrom = ref<AdminCouponItem | null>(null)
const noticeForm = reactive<AdminNoticeReq>({ title: '', content: '', is_show: true, sort: 0 })

/** 依据优惠券生成公告草稿（优惠码用反引号包裹，用户端 NoticePanel 渲染为高亮可复制 chip） */
function buildNoticeDraft(c: AdminCouponItem) {
  const discount = c.type === 1 ? `立减 ${c.value.toFixed(2)} 元` : `享 ${c.value}% 折扣`
  const gate = c.min_spend > 0 ? `（满 ${c.min_spend.toFixed(2)} 元可用）` : ''
  const period = periodText(c.valid_periods)
  const plan = planText(c.plan_ids)
  return {
    title: `限时福利:优惠码 ${c.code}`,
    content:
      `## 限时福利\n\n` +
      `使用优惠码 **${c.code}** 下单${discount}${gate},适用${period}·${plan}。\n\n` +
      `优惠码:\`${c.code}\`(点击复制,下单时输入或直接点选「可用优惠券」)`,
  }
}

function openNotice(c: AdminCouponItem) {
  noticeFrom.value = c
  Object.assign(noticeForm, { is_show: true, sort: 0, ...buildNoticeDraft(c) })
  noticeModal.value = true
}

async function publishNotice() {
  if (!noticeForm.title.trim()) {
    message.warning('请输入公告标题')
    return
  }
  if (!noticeForm.content.trim()) {
    message.warning('请输入公告内容')
    return
  }
  noticeSaving.value = true
  try {
    await apiAdmin.createNotice({ ...noticeForm })
    message.success(`公告已发布,用户端仪表板立即可见（含优惠码 ${noticeFrom.value?.code ?? ''}）`)
    noticeModal.value = false
  } finally {
    noticeSaving.value = false
  }
}

const typeLabel = computed(() => (form.type === 1 ? '固定金额' : '百分比'))

async function load() {
  loading.value = true
  try {
    const [couponRes, planRes] = await Promise.all([apiAdmin.coupons(), apiAdmin.plans()])
    list.value = couponRes.list
    plans.value = planRes.list
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  Object.assign(form, {
    code: '',
    type: 2,
    value: 0,
    min_spend: 0,
    limit_per_user: 0,
    total_limit: 0,
    valid_periods: [],
    plan_ids: [],
    started_at: null,
    ended_at: null,
    is_enable: true,
  })
  modal.value = true
}

function openEdit(c: AdminCouponItem) {
  editingId.value = c.id
  Object.assign(form, {
    code: c.code,
    type: c.type,
    value: c.value,
    min_spend: c.min_spend,
    limit_per_user: c.limit_per_user,
    total_limit: c.total_limit,
    valid_periods: [...c.valid_periods],
    plan_ids: [...c.plan_ids],
    started_at: c.started_at,
    ended_at: c.ended_at,
    is_enable: c.is_enable,
  })
  modal.value = true
}

async function save() {
  if (!form.code.trim()) {
    message.warning('请输入优惠码')
    return
  }
  if (typeof form.value !== 'number' || form.value <= 0) {
    message.warning(`请输入${typeLabel.value}金额`)
    return
  }
  saving.value = true
  try {
    if (editingId.value === null) {
      await apiAdmin.createCoupon({ ...form })
      message.success('优惠券已创建')
    } else {
      await apiAdmin.updateCoupon(editingId.value, { ...form })
      message.success('优惠券已更新')
    }
    modal.value = false
    void load()
  } finally {
    saving.value = false
  }
}

function remove(c: AdminCouponItem) {
  dialog.warning({
    title: '删除优惠券',
    content: `确定删除优惠券「${c.code}」吗?该操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.deleteCoupon(c.id)
      message.success('已删除')
      void load()
    },
  })
}

function discountText(c: AdminCouponItem): string {
  return c.type === 1 ? `¥${c.value.toFixed(2)}` : `${c.value}%`
}

function periodText(periods: string[]): string {
  if (!periods || periods.length === 0) return '全部周期'
  return periods.map((p) => PERIODS.find((x) => x.value === p)?.label ?? p).join(', ')
}

function planText(ids: number[]): string {
  if (!ids || ids.length === 0) return '全部套餐'
  const names = ids.map((id) => plans.value.find((p) => p.id === id)?.name)
  return names.filter(Boolean).join(', ') || String(ids.length)
}

function periodOptions() {
  return PERIODS.map((p) => ({ label: p.label, value: p.value }))
}

function planOptions() {
  return plans.value.map((p) => ({ label: p.name, value: p.id }))
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader title="优惠券管理" subtitle="折扣码与满减券的新建、编辑与上下架">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> 刷新
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> 新建优惠券
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[960px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>优惠码</th>
              <th>类型</th>
              <th>面值</th>
              <th>最低消费</th>
              <th>每人限用</th>
              <th>总量</th>
              <th>已用</th>
              <th>适用周期</th>
              <th>适用套餐</th>
              <th>有效期</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in list" :key="c.id">
              <td class="num-font">{{ c.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ c.code }}</td>
              <td class="text-14">{{ c.type === 1 ? '固定金额' : '百分比' }}</td>
              <td class="num-font">{{ discountText(c) }}</td>
              <td class="num-font">{{ c.min_spend > 0 ? `¥${c.min_spend.toFixed(2)}` : '无' }}</td>
              <td class="num-font">{{ c.limit_per_user > 0 ? c.limit_per_user : '不限' }}</td>
              <td class="num-font">{{ c.total_limit > 0 ? c.total_limit : '不限' }}</td>
              <td class="num-font">{{ c.used_count }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ periodText(c.valid_periods) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ planText(c.plan_ids) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ c.started_at ? formatTime(c.started_at, false) : '不限' }} ~
                {{ c.ended_at ? formatTime(c.ended_at, false) : '不限' }}
              </td>
              <td>
                <StatusBadge :type="c.is_enable ? 'success' : 'neutral'">
                  {{ c.is_enable ? '启用' : '停用' }}
                </StatusBadge>
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(c)">
                    编辑
                  </button>
                  <button class="btn-soft-warning h-7 px-3 text-14" @click="openNotice(c)">
                    发公告
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(c)">删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="13"><EmptyState text="暂无优惠券" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? '新建优惠券' : '编辑优惠券'"
      style="width: 600px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="优惠码">
            <n-input v-model:value="form.code" placeholder="如 WELCOME10" />
          </n-form-item>
          <n-form-item label="类型">
            <n-radio-group v-model:value="form.type">
              <n-radio-button :value="2">百分比</n-radio-button>
              <n-radio-button :value="1">固定金额</n-radio-button>
            </n-radio-group>
          </n-form-item>
          <n-form-item :label="typeLabel === '固定金额' ? '面值(元)' : '折扣(%)'">
            <n-input-number v-model:value="form.value" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="最低消费(元,0=不限)">
            <n-input-number v-model:value="form.min_spend" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="每人限用次数(0=不限)">
            <n-input-number v-model:value="form.limit_per_user" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item label="总发放量(0=不限)">
            <n-input-number v-model:value="form.total_limit" :min="0" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item label="适用周期(不选=全部)">
          <n-select v-model:value="form.valid_periods" multiple :options="periodOptions()" />
        </n-form-item>
        <n-form-item label="适用套餐(不选=全部)">
          <n-select v-model:value="form.plan_ids" multiple :options="planOptions()" />
        </n-form-item>
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="生效时间(留空=立即)">
            <n-date-picker
              v-model:formatted-value="form.started_at"
              value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
              type="datetime"
              clearable
              class="w-full"
            />
          </n-form-item>
          <n-form-item label="失效时间(留空=长期)">
            <n-date-picker
              v-model:formatted-value="form.ended_at"
              value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
              type="datetime"
              clearable
              class="w-full"
            />
          </n-form-item>
        </div>
        <n-form-item label="启用">
          <n-switch v-model:value="form.is_enable" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="modal = false">取消</button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="save">
            保存
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 一键公告弹窗：为优惠券生成公告草稿，可编辑后发布 -->
    <n-modal
      v-model:show="noticeModal"
      preset="card"
      :title="`发布公告 · 优惠码 ${noticeFrom?.code ?? ''}`"
      style="width: 640px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item label="标题">
            <n-input v-model:value="noticeForm.title" placeholder="公告标题" />
          </n-form-item>
          <n-form-item label="排序">
            <n-input-number v-model:value="noticeForm.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item label="内容(Markdown,优惠码用反引号包裹会在用户端高亮可复制)">
          <n-input
            v-model:value="noticeForm.content"
            type="textarea"
            :rows="7"
            placeholder="公告正文,支持 Markdown"
          />
        </n-form-item>
        <n-form-item label="展示">
          <n-switch v-model:value="noticeForm.is_show" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="noticeModal = false">
            取消
          </button>
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="noticeSaving"
            @click="publishNotice"
          >
            发布公告
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
