<script setup lang="ts">
/**
 * 管理后台 · 用户管理:搜索/分页/封禁/角色调整/余额调整。
 * 数据:GET/PUT /admin/users、POST /admin/users/{id}/balance(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { AdminRole, AdminUserItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'
import { formatBytes } from '@/utils/format'

const { t } = useI18n()
const message = useMessage()
const dialog = useDialog()
const auth = useAuthStore()

const loading = ref(false)
const list = ref<AdminUserItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const keyword = ref('')

// 调余额弹窗
const balanceModal = ref(false)
const balanceTarget = ref<AdminUserItem | null>(null)
const balanceAmount = ref<number | null>(null)
const balanceRemark = ref('')
const balanceSaving = ref(false)

// 角色调整弹窗
const roleModal = ref(false)
const roleTarget = ref<AdminUserItem | null>(null)
const roleValue = ref<AdminRole>(0)
const roleSaving = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await apiAdmin.users({
      page: page.value,
      page_size: pageSize,
      keyword: keyword.value,
    })
    list.value = res.list
    total.value = res.total
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  void load()
}

function onPageChange(p: number) {
  page.value = p
  void load()
}

function toggleBan(u: AdminUserItem) {
  if (u.id === auth.user?.id) {
    message.warning(t('adminUsers.cannotOperateSelf'))
    return
  }
  dialog.warning({
    title: u.is_banned ? t('adminUsers.unbanTitle') : t('adminUsers.banTitle'),
    content: u.is_banned
      ? t('adminUsers.unbanConfirm', { email: u.email })
      : t('adminUsers.banConfirm', { email: u.email }),
    positiveText: t('adminUsers.ok'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await apiAdmin.updateUser(u.id, { banned: !u.is_banned })
      message.success(t('adminUsers.updated'))
      void load()
    },
  })
}

function changeRole(u: AdminUserItem) {
  if (u.id === auth.user?.id) {
    message.warning(t('adminUsers.cannotOperateSelf'))
    return
  }
  roleTarget.value = u
  roleValue.value = u.role
  roleModal.value = true
}

async function saveRole() {
  if (!roleTarget.value) return
  roleSaving.value = true
  try {
    await apiAdmin.updateUser(roleTarget.value.id, { role: roleValue.value })
    message.success(t('adminUsers.roleUpdated'))
    roleModal.value = false
    void load()
  } finally {
    roleSaving.value = false
  }
}

function openBalance(u: AdminUserItem) {
  balanceTarget.value = u
  balanceAmount.value = null
  balanceRemark.value = ''
  balanceModal.value = true
}

function roleLabel(role: AdminRole): string {
  return role === 1
    ? t('adminUsers.roleAdmin')
    : role === 2
      ? t('adminUsers.roleAgent')
      : t('adminUsers.roleNormal')
}

async function saveBalance() {
  if (!balanceTarget.value || balanceAmount.value === null) return
  balanceSaving.value = true
  try {
    await apiAdmin.adjustBalance(balanceTarget.value.id, {
      amount: balanceAmount.value,
      remark: balanceRemark.value,
    })
    message.success(t('adminUsers.balanceAdjusted'))
    balanceModal.value = false
    void load()
  } finally {
    balanceSaving.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div>
    <PageHeader :title="t('adminUsers.title')" :subtitle="t('adminUsers.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-input
            v-model:value="keyword"
            :placeholder="t('adminUsers.searchPlaceholder')"
            clearable
            class="shrink-0"
            style="width: 22rem"
            @keyup.enter="onSearch"
          />
          <button class="btn-primary h-9 shrink-0 px-4 text-14" @click="onSearch">
            {{ t('adminUsers.search') }}
          </button>
          <button class="btn-soft-neutral h-9 shrink-0 px-3 text-14" @click="onSearch">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[760px]">
          <thead>
            <tr>
              <th>ID</th>
              <th>{{ t('adminUsers.email') }}</th>
              <th>{{ t('adminUsers.role') }}</th>
              <th>{{ t('adminUsers.balance') }}</th>
              <th>{{ t('adminUsers.commission') }}</th>
              <th>{{ t('adminUsers.traffic') }}</th>
              <th>{{ t('adminUsers.status') }}</th>
              <th>{{ t('adminUsers.registeredAt') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in list" :key="u.id">
              <td>{{ u.id }}</td>
              <td class="whitespace-nowrap font-500 text-[var(--c-text)]">{{ u.email }}</td>
              <td>
                <StatusBadge
                  :type="u.role === 1 ? 'primary' : u.role === 2 ? 'marketing' : 'neutral'"
                >
                  {{ roleLabel(u.role) }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ u.balance.toFixed(2) }}</td>
              <td class="num-font">{{ u.commission_balance.toFixed(2) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ formatBytes(u.u) }} ↑ / {{ formatBytes(u.d) }} ↓
              </td>
              <td>
                <StatusBadge :type="u.is_banned ? 'danger' : 'success'">
                  {{ u.is_banned ? t('adminUsers.banned') : t('adminUsers.normal') }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ u.created_at.slice(0, 10) }}
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-3 text-14" @click="changeRole(u)">
                    {{ t('adminUsers.adjustRole') }}
                  </button>
                  <button class="btn-soft-warning h-7 px-3 text-14" @click="openBalance(u)">
                    {{ t('adminUsers.adjustBalance') }}
                  </button>
                  <button
                    class="h-7 px-3 text-14"
                    :class="u.is_banned ? 'btn-soft-success' : 'btn-danger'"
                    @click="toggleBan(u)"
                  >
                    {{ u.is_banned ? t('adminUsers.unban') : t('adminUsers.ban') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="9"><EmptyState :text="t('adminUsers.empty')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
      <div class="flex justify-end p-4">
        <n-pagination
          v-model:page="page"
          :item-count="total"
          :page-size="pageSize"
          @update:page="onPageChange"
        />
      </div>
    </div>

    <!-- 调余额弹窗 -->
    <n-modal
      v-model:show="balanceModal"
      preset="card"
      :title="t('adminUsers.adjustBalanceTitle')"
      style="width: 420px"
    >
      <n-form label-placement="left" label-width="90">
        <n-form-item :label="t('adminUsers.user')">
          <span class="text-14">{{ balanceTarget?.email }}</span>
        </n-form-item>
        <n-form-item :label="t('adminUsers.amount')">
          <n-input-number
            v-model:value="balanceAmount"
            :placeholder="t('adminUsers.amountPlaceholder')"
            class="w-full"
          />
        </n-form-item>
        <n-form-item :label="t('adminUsers.remark')">
          <n-input v-model:value="balanceRemark" :placeholder="t('adminUsers.remarkPlaceholder')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="balanceModal = false">
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="balanceSaving"
            @click="saveBalance"
          >
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 角色调整弹窗 -->
    <n-modal
      v-model:show="roleModal"
      preset="card"
      :title="t('adminUsers.adjustRoleTitle')"
      style="width: 360px"
    >
      <n-form label-placement="left" label-width="60">
        <n-form-item :label="t('adminUsers.user')">
          <span class="text-14">{{ roleTarget?.email }}</span>
        </n-form-item>
        <n-form-item :label="t('adminUsers.role')">
          <n-select
            v-model:value="roleValue"
            :options="[
              { label: t('adminUsers.roleNormal'), value: 0 },
              { label: t('adminUsers.roleAdmin'), value: 1 },
              { label: t('adminUsers.roleAgent'), value: 2 },
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-soft-neutral h-9 px-4 text-14" @click="roleModal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="roleSaving" @click="saveRole">
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
