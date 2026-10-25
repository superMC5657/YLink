<script setup lang="ts">
/**
 * 管理后台 · 套餐管理:列表 / 新建 / 编辑 / 删除。
 * 数据:GET/POST/PUT/DELETE /admin/plans + GET /admin/server-groups(docs/api/README.md §16)
 * CRUD 脚手架归 useAdminCrud(分组为旁路数据,fetch 闭包内同步拉取)。
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import type { AdminPlanItem, AdminPlanReq, AdminServerGroupItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage } from 'naive-ui'
import { useAdminCrud } from '@/composables/useAdminCrud'

const { t } = useI18n()
const message = useMessage()

const groups = ref<AdminServerGroupItem[]>([])

const { loading, list, modal, editingId, saving, form, load, openCreate, openEdit, save, remove } =
  useAdminCrud<AdminPlanItem, AdminPlanReq>({
    fetch: async () => {
      const [planRes, groupRes] = await Promise.all([apiAdmin.plans(), apiAdmin.serverGroups()])
      groups.value = groupRes.list
      return planRes
    },
    emptyForm: () => ({
      name: '',
      content: '',
      month_price: null,
      quarter_price: null,
      half_year_price: null,
      year_price: null,
      onetime_price: null,
      traffic_gb: 0,
      speed_limit: null,
      device_limit: null,
      group_ids: [],
      is_show: true,
      sort: 0,
    }),
    toForm: (p) => ({
      name: p.name,
      content: p.content,
      month_price: p.month_price,
      quarter_price: p.quarter_price,
      half_year_price: p.half_year_price,
      year_price: p.year_price,
      onetime_price: p.onetime_price,
      traffic_gb: p.traffic_gb,
      speed_limit: p.speed_limit,
      device_limit: p.device_limit,
      group_ids: p.group_ids ?? [],
      is_show: p.is_show,
      sort: p.sort,
    }),
    create: (body) => apiAdmin.createPlan(body),
    update: (id, body) => apiAdmin.updatePlan(id, body),
    validate: (f) => {
      if (!f.name.trim()) {
        message.warning(t('adminPlans.enterName'))
        return false
      }
      return true
    },
    remove: (id) => apiAdmin.deletePlan(id),
    removeTexts: {
      titleKey: 'adminPlans.deleteTitle',
      content: (p) => t('adminPlans.deleteConfirm', { name: p.name }),
      positiveKey: 'adminPlans.delete',
    },
    messages: {
      created: 'adminPlans.created',
      updated: 'adminPlans.updated',
      deleted: 'adminPlans.deleted',
    },
  })

function price(v: number | null): string {
  return v === null ? '-' : `¥${v}`
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminPlans.pageTitle')" :subtitle="t('adminPlans.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminPlans.newPlan') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[860px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('adminPlans.name') }}</th>
              <th>{{ t('adminPlans.month') }}</th>
              <th>{{ t('adminPlans.quarter') }}</th>
              <th>{{ t('adminPlans.halfYear') }}</th>
              <th>{{ t('adminPlans.year') }}</th>
              <th>{{ t('adminPlans.onetime') }}</th>
              <th>{{ t('adminPlans.traffic') }}</th>
              <th>{{ t('adminPlans.devices') }}</th>
              <th>{{ t('adminPlans.status') }}</th>
              <th>{{ t('adminPlans.sort') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in list" :key="p.id">
              <td>{{ p.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ p.name }}</td>
              <td class="num-font">{{ price(p.month_price) }}</td>
              <td class="num-font">{{ price(p.quarter_price) }}</td>
              <td class="num-font">{{ price(p.half_year_price) }}</td>
              <td class="num-font">{{ price(p.year_price) }}</td>
              <td class="num-font">{{ price(p.onetime_price) }}</td>
              <td>{{ p.traffic_gb }} GB</td>
              <td>{{ p.device_limit ?? t('adminPlans.unlimited') }}</td>
              <td>
                <StatusBadge :type="p.is_show ? 'success' : 'neutral'">
                  {{ p.is_show ? t('adminPlans.onSale') : t('adminPlans.hidden') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ p.sort }}</td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="openEdit(p)">
                    {{ t('adminPlans.edit') }}
                  </button>
                  <button class="btn-danger h-7 px-3 text-14" @click="remove(p)">
                    {{ t('adminPlans.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="12"><EmptyState :text="t('adminPlans.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 新建/编辑弹窗 -->
    <n-modal
      v-model:show="modal"
      preset="card"
      :title="editingId === null ? t('adminPlans.createTitle') : t('adminPlans.editTitle')"
      style="width: 560px"
    >
      <n-form label-placement="top">
        <div class="grid grid-cols-2 gap-x-4">
          <n-form-item :label="t('adminPlans.name')">
            <n-input v-model:value="form.name" :placeholder="t('adminPlans.namePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.trafficLabel')">
            <n-input-number v-model:value="form.traffic_gb" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.monthLabel')">
            <n-input-number v-model:value="form.month_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.quarterLabel')">
            <n-input-number v-model:value="form.quarter_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.halfYearLabel')">
            <n-input-number v-model:value="form.half_year_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.yearLabel')">
            <n-input-number v-model:value="form.year_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.onetimeLabel')">
            <n-input-number v-model:value="form.onetime_price" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.deviceLimitLabel')">
            <n-input-number v-model:value="form.device_limit" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.speedLimitLabel')">
            <n-input-number v-model:value="form.speed_limit" :min="0" class="w-full" />
          </n-form-item>
          <n-form-item :label="t('adminPlans.sortLabel')">
            <n-input-number v-model:value="form.sort" class="w-full" />
          </n-form-item>
        </div>
        <n-form-item :label="t('adminPlans.groupsLabel')">
          <n-select
            v-model:value="form.group_ids"
            multiple
            :options="groups.map((g) => ({ label: g.name, value: g.id }))"
            :placeholder="t('adminPlans.groupsPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('adminPlans.onSaleLabel')">
          <n-switch v-model:value="form.is_show" />
        </n-form-item>
        <n-form-item :label="t('adminPlans.contentLabel')">
          <n-input
            v-model:value="form.content"
            type="textarea"
            :rows="3"
            :placeholder="t('adminPlans.contentPlaceholder')"
          />
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
  </div>
</template>
