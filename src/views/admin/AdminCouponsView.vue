<script setup lang="ts">
/**
 * 管理后台 · 优惠券管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/coupons + GET /admin/plans(docs/api/README.md §16.1)
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminCouponItem, AdminCouponReq, AdminNoticeReq, AdminPlanItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatTime } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()

const loading = ref(false)
const list = ref<AdminCouponItem[]>([])
const plans = ref<AdminPlanItem[]>([])

const PERIODS = [
  { value: 'month', labelKey: 'adminCoupons.month' },
  { value: 'quarter', labelKey: 'adminCoupons.quarter' },
  { value: 'half_year', labelKey: 'adminCoupons.halfYear' },
  { value: 'year', labelKey: 'adminCoupons.year' },
  { value: 'onetime', labelKey: 'adminCoupons.onetime' },
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
  const discount =
    c.type === 1
      ? t('adminCoupons.draftDiscountFixed', { amount: c.value.toFixed(2) })
      : t('adminCoupons.draftDiscountPercent', { value: c.value })
  const gate =
    c.min_spend > 0 ? t('adminCoupons.draftGate', { amount: c.min_spend.toFixed(2) }) : ''
  const period = periodText(c.valid_periods)
  const plan = planText(c.plan_ids)
  return {
    title: t('adminCoupons.draftTitle', { code: c.code }),
    content: t('adminCoupons.draftBody', { code: c.code, discount, gate, period, plan }),
  }
}

function openNotice(c: AdminCouponItem) {
  noticeFrom.value = c
  Object.assign(noticeForm, { is_show: true, sort: 0, ...buildNoticeDraft(c) })
  noticeModal.value = true
}

async function publishNotice() {
  if (!noticeForm.title.trim()) {
    message.warning(t('adminCoupons.enterNoticeTitle'))
    return
  }
  if (!noticeForm.content.trim()) {
    message.warning(t('adminCoupons.enterNoticeContent'))
    return
  }
  noticeSaving.value = true
  try {
    await apiAdmin.createNotice({ ...noticeForm })
    message.success(t('adminCoupons.noticePublished', { code: noticeFrom.value?.code ?? '' }))
    noticeModal.value = false
  } finally {
    noticeSaving.value = false
  }
}

const typeLabel = computed(() =>
  form.type === 1 ? t('adminCoupons.fixed') : t('adminCoupons.percent'),
)

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
    message.warning(t('adminCoupons.enterCode'))
    return
  }
  if (typeof form.value !== 'number' || form.value <= 0) {
    message.warning(t('adminCoupons.enterValue', { type: typeLabel.value }))
    return
  }
  saving.value = true
  try {
    if (editingId.value === null) {
      await apiAdmin.createCoupon({ ...form })
      message.success(t('adminCoupons.created'))
    } else {
      await apiAdmin.updateCoupon(editingId.value, { ...form })
      message.success(t('adminCoupons.updated'))
    }
    modal.value = false
    void load()
  } finally {
    saving.value = false
  }
}

function remove(c: AdminCouponItem) {
  dialog.warning({
    title: t('adminCoupons.deleteTitle'),
    content: t('adminCoupons.deleteConfirm', { code: c.code }),
    positiveText: t('adminCoupons.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.deleteCoupon(c.id)
      message.success(t('adminCoupons.deleted'))
      void load()
    },
  })
}

function discountText(c: AdminCouponItem): string {
  return c.type === 1 ? `¥${c.value.toFixed(2)}` : `${c.value}%`
}

function periodText(periods: string[]): string {
  if (!periods || periods.length === 0) return t('adminCoupons.allPeriods')
  return periods
    .map((p) => PERIODS.find((x) => x.value === p)?.labelKey ?? p)
    .map((k) => t(k))
    .join(', ')
}

function planText(ids: number[]): string {
  if (!ids || ids.length === 0) return t('adminCoupons.allPlans')
  const names = ids.map((id) => plans.value.find((p) => p.id === id)?.name)
  return names.filter(Boolean).join(', ') || String(ids.length)
}

function periodOptions() {
  return PERIODS.map((p) => ({ label: t(p.labelKey), value: p.value }))
}

function planOptions() {
  return plans.value.map((p) => ({ label: p.name, value: p.id }))
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminCoupons.pageTitle')" :subtitle="t('adminCoupons.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminCoupons.newCoupon') }}
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
              <th>{{ t('adminCoupons.code') }}</th>
              <th>{{ t('adminCoupons.type') }}</th>
              <th>{{ t('adminCoupons.value') }}</th>
              <th>{{ t('adminCoupons.minSpend') }}</th>
              <th>{{ t('adminCoupons.perUserLimit') }}</th>
              <th>{{ t('adminCoupons.total') }}</th>
              <th>{{ t('adminCoupons.used') }}</th>
              <th>{{ t('adminCoupons.periods') }}</th>
              <th>{{ t('adminCoupons.plans') }}</th>
              <th>{{ t('adminCoupons.validUntil') }}</th>
              <th>{{ t('adminCoupons.status') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in list" :key="c.id">
              <td class="num-font">{{ c.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ c.code }}</td>
              <td class="text-14">
                {{ c.type === 1 ? t('adminCoupons.fixed') : t('adminCoupons.percent') }}
              </td>
              <td class="num-font">{{ discountText(c) }}</td>
              <td class="num-font">
                {{ c.min_spend > 0 ? `¥${c.min_spend.toFixed(2)}` : t('adminCoupons.none') }}
              </td>
              <td class="num-font">
                {{ c.limit_per_user > 0 ? c.limit_per_user : t('adminCoupons.unlimited') }}
              </td>
              <td class="num-font">
                {{ c.total_limit > 0 ? c.total_limit : t('adminCoupons.unlimited') }}
              </td>
              <td class="num-font">{{ c.used_count }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ periodText(c.valid_periods) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ planText(c.plan_ids) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ c.started_at ? formatTime(c.started_at, false) : t('adminCoupons.unlimited') }} ~
                {{ c.ended_at ? formatTime(c.ended_at, false) : t('adminCoupons.unlimited') }}
              </td>
              <td>
                <StatusBadge :type="c.is_enable ? 'success' : 'neutral'">
                  {{ c.is_enable ? t('adminCoupons.enabled') : t('adminCoupons.disabled') }}
                </StatusBadge>
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(c)">
                    {{ t('adminCoupons.edit') }}
                  </button>
                  <button class="btn-soft-warning h-7 px-3 text-14" @click="openNotice(c)">
                    {{ t('adminCoupons.publishNotice') }}
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(c)">
                    {{ t('adminCoupons.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="13"><EmptyState :text="t('adminCoupons.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? t('adminCoupons.createTitle') : t('adminCoupons.editTitle')"
      style="width: 600px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminCoupons.code')">
            <n-input v-model:value="form.code" :placeholder="t('adminCoupons.codePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.type')">
            <n-radio-group v-model:value="form.type">
              <n-radio-button :value="2">{{ t('adminCoupons.percent') }}</n-radio-button>
              <n-radio-button :value="1">{{ t('adminCoupons.fixed') }}</n-radio-button>
            </n-radio-group>
          </n-form-item>
          <n-form-item
            :label="form.type === 1 ? t('adminCoupons.valueFixed') : t('adminCoupons.valuePercent')"
          >
            <n-input-number v-model:value="form.value" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.minSpendLabel')">
            <n-input-number v-model:value="form.min_spend" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.perUserLimitLabel')">
            <n-input-number v-model:value="form.limit_per_user" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.totalLimitLabel')">
            <n-input-number v-model:value="form.total_limit" :min="0" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminCoupons.periodsLabel')">
          <n-select v-model:value="form.valid_periods" multiple :options="periodOptions()" />
        </n-form-item>
        <n-form-item :label="t('adminCoupons.plansLabel')">
          <n-select v-model:value="form.plan_ids" multiple :options="planOptions()" />
        </n-form-item>
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminCoupons.startedAt')">
            <n-date-picker
              v-model:formatted-value="form.started_at"
              value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
              type="datetime"
              clearable
              class="w-full"
            />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.endedAt')">
            <n-date-picker
              v-model:formatted-value="form.ended_at"
              value-format="yyyy-MM-dd'T'HH:mm:ssXXX"
              type="datetime"
              clearable
              class="w-full"
            />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminCoupons.enable')">
          <n-switch v-model:value="form.is_enable" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="modal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="save">
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 一键公告弹窗：为优惠券生成公告草稿，可编辑后发布 -->
    <n-modal
      v-model:show="noticeModal"
      preset="card"
      :title="t('adminCoupons.noticeTitle', { code: noticeFrom?.code ?? '' })"
      style="width: 640px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminNotices.title')">
            <n-input
              v-model:value="noticeForm.title"
              :placeholder="t('adminCoupons.noticeTitlePlaceholder')"
            />
          </n-form-item>
          <n-form-item :label="t('adminCoupons.sort')">
            <n-input-number v-model:value="noticeForm.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminCoupons.contentLabel')">
          <n-input
            v-model:value="noticeForm.content"
            type="textarea"
            :rows="7"
            :placeholder="t('adminCoupons.contentPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('adminCoupons.show')">
          <n-switch v-model:value="noticeForm.is_show" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="noticeModal = false">
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="noticeSaving"
            @click="publishNotice"
          >
            {{ t('adminCoupons.publish') }}
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
