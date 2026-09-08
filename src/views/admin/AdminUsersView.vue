<script setup lang="ts">
/**
 * 管理后台 · 用户管理:搜索/分页/封禁/角色调整/余额调整。
 * 数据:GET/PUT /admin/users、POST /admin/users/{id}/balance(docs/api/README.md §16)
 * 状态与动作在 composables/useAdminUsers;四个弹窗拆在 components/admin/。
 */
import { useI18n } from 'vue-i18n'
import { useAdminUsers } from '@/composables/useAdminUsers'
import { formatBytes } from '@/utils/format'
import PageHeader from '@/components/ui/PageHeader.vue'
import UserBalanceDialog from '@/components/admin/UserBalanceDialog.vue'
import UserRoleDialog from '@/components/admin/UserRoleDialog.vue'
import UserBatchBalanceDialog from '@/components/admin/UserBatchBalanceDialog.vue'
import UserMailDialog from '@/components/admin/UserMailDialog.vue'

const { t } = useI18n()
const {
  loading,
  list,
  total,
  page,
  pageSize,
  keyword,
  balanceModal,
  balanceTarget,
  balanceAmount,
  balanceRemark,
  balanceSaving,
  roleModal,
  roleTarget,
  roleValue,
  roleSaving,
  selectedIds,
  batchSaving,
  batchBalanceModal,
  batchBalanceAmount,
  batchBalanceRemark,
  mailModal,
  mailSubject,
  mailBody,
  mailSaving,
  exporting,
  roleLabel,
  allChecked,
  onSearch,
  onPageChange,
  toggleBan,
  changeRole,
  saveRole,
  openBalance,
  saveBalance,
  isSelected,
  toggleRow,
  toggleAll,
  clearSelection,
  batchBan,
  openBatchBalance,
  saveBatchBalance,
  openSendMail,
  saveSendMail,
  exportCSV,
  resetSub,
} = useAdminUsers()
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
          <button
            class="btn-soft-neutral h-9 shrink-0 px-3 text-14"
            :disabled="exporting"
            @click="exportCSV"
          >
            <AppIcon name="download" :size="15" /> {{ t('adminUsers.exportCsv') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <!-- F05 批量操作工具栏(选中 > 0 时显示) -->
    <div v-if="selectedIds.length > 0" class="card-base mb-4 flex flex-wrap items-center gap-2 p-3">
      <span class="text-14 text-[var(--c-text-sub)]">
        {{ t('adminUsers.selectedCount', { count: selectedIds.length }) }}
      </span>
      <button
        class="btn-soft-warning h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchBan(true)"
      >
        {{ t('adminUsers.batchBan') }}
      </button>
      <button
        class="btn-soft-success h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchBan(false)"
      >
        {{ t('adminUsers.batchUnban') }}
      </button>
      <button
        class="btn-soft-primary h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="openBatchBalance"
      >
        {{ t('adminUsers.batchAdjustBalance') }}
      </button>
      <button
        class="btn-soft-primary h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="openSendMail"
      >
        {{ t('adminUsers.sendMail') }}
      </button>
      <button class="btn-soft-neutral h-8 px-3 text-14" @click="clearSelection">
        {{ t('adminUsers.clearSelection') }}
      </button>
    </div>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[820px]">
          <thead>
            <tr>
              <th class="w-10">
                <n-checkbox :checked="allChecked" @update:checked="toggleAll" />
              </th>
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
              <td>
                <n-checkbox
                  :checked="isSelected(u.id)"
                  @update:checked="(v: boolean) => toggleRow(u.id, v)"
                />
              </td>
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
                  <button class="btn-soft-neutral h-7 px-3 text-14" @click="resetSub(u)">
                    {{ t('adminUsers.resetSub') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="10"><EmptyState :text="t('adminUsers.empty')" /></td>
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
    <UserBalanceDialog
      v-model:show="balanceModal"
      v-model:amount="balanceAmount"
      v-model:remark="balanceRemark"
      :target="balanceTarget"
      :saving="balanceSaving"
      @save="saveBalance"
    />

    <!-- 角色调整弹窗 -->
    <UserRoleDialog
      v-model:show="roleModal"
      v-model:role="roleValue"
      :target="roleTarget"
      :saving="roleSaving"
      @save="saveRole"
    />

    <!-- F05 批量调余额弹窗 -->
    <UserBatchBalanceDialog
      v-model:show="batchBalanceModal"
      v-model:amount="batchBalanceAmount"
      v-model:remark="batchBalanceRemark"
      :selected-count="selectedIds.length"
      :saving="batchSaving"
      @save="saveBatchBalance"
    />

    <!-- F05 发送邮件弹窗 -->
    <UserMailDialog
      v-model:show="mailModal"
      :subject="mailSubject"
      :body="mailBody"
      :selected-count="selectedIds.length"
      :saving="mailSaving"
      @update:subject="mailSubject = $event"
      @update:body="mailBody = $event"
      @save="saveSendMail"
    />
  </div>
</template>
