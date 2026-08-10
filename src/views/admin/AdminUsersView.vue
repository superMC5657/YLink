<script setup lang="ts">
/**
 * 管理后台 · 用户管理:搜索/分页/封禁/角色调整/余额调整。
 * 数据:GET/PUT /admin/users、POST /admin/users/{id}/balance(docs/api/README.md §16)
 */
import { onMounted, ref } from 'vue'
import { apiAdmin, ADMIN_ROLE_LABEL } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { AdminRole, AdminUserItem } from '@/types/api'
import PageHeader from '@/components/ui/PageHeader.vue'
import { useMessage, useDialog } from 'naive-ui'

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

function formatTraffic(v: number): string {
  if (v <= 0) return '-'
  if (v >= 1024 ** 3) return `${(v / 1024 ** 3).toFixed(1)} GB`
  if (v >= 1024 ** 2) return `${(v / 1024 ** 2).toFixed(0)} MB`
  return `${v} B`
}

function toggleBan(u: AdminUserItem) {
  if (u.id === auth.user?.id) {
    message.warning('不能操作当前登录的管理员账号')
    return
  }
  dialog.warning({
    title: u.is_banned ? '解封用户' : '封禁用户',
    content: u.is_banned
      ? `确定解封 ${u.email} 吗?`
      : `确定封禁 ${u.email} 吗?封禁后该账号无法登录。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      await apiAdmin.updateUser(u.id, { banned: !u.is_banned })
      message.success('已更新')
      void load()
    },
  })
}

function changeRole(u: AdminUserItem) {
  if (u.id === auth.user?.id) {
    message.warning('不能操作当前登录的管理员账号')
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
    message.success('角色已更新')
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

async function saveBalance() {
  if (!balanceTarget.value || balanceAmount.value === null) return
  balanceSaving.value = true
  try {
    await apiAdmin.adjustBalance(balanceTarget.value.id, {
      amount: balanceAmount.value,
      remark: balanceRemark.value,
    })
    message.success('余额已调整')
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
    <PageHeader title="用户管理" subtitle="搜索 / 封禁 / 角色 / 余额调整(操作均写入审计)">
      <template #actions>
        <div class="flex items-center gap-2">
          <n-input
            v-model:value="keyword"
            placeholder="搜索邮箱"
            clearable
            class="w-48"
            @keyup.enter="onSearch"
          />
          <button class="btn-primary h-9 px-4 text-14" @click="onSearch">搜索</button>
          <button class="btn-ghost h-9 px-3 text-14" @click="onSearch">
            <AppIcon name="refresh" :size="15" /> 刷新
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
              <th>邮箱</th>
              <th>角色</th>
              <th>余额(元)</th>
              <th>佣金(元)</th>
              <th>流量</th>
              <th>状态</th>
              <th>注册时间</th>
              <th class="text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in list" :key="u.id">
              <td>{{ u.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ u.email }}</td>
              <td>
                <StatusBadge
                  :type="u.role === 1 ? 'primary' : u.role === 2 ? 'marketing' : 'neutral'"
                >
                  {{ ADMIN_ROLE_LABEL[u.role] }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ u.balance.toFixed(2) }}</td>
              <td class="num-font">{{ u.commission_balance.toFixed(2) }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ formatTraffic(u.u) }} ↑ / {{ formatTraffic(u.d) }} ↓
              </td>
              <td>
                <StatusBadge :type="u.is_banned ? 'danger' : 'success'">
                  {{ u.is_banned ? '已封禁' : '正常' }}
                </StatusBadge>
              </td>
              <td class="text-14 text-[var(--c-text-sub)]">
                {{ u.created_at.slice(0, 10) }}
              </td>
              <td>
                <div class="flex justify-end gap-2">
                  <button class="btn-ghost h-7 px-3 text-14" @click="changeRole(u)">角色</button>
                  <button class="btn-ghost h-7 px-3 text-14" @click="openBalance(u)">调余额</button>
                  <button
                    class="h-7 px-3 text-14"
                    :class="u.is_banned ? 'btn-olive' : 'btn-danger'"
                    @click="toggleBan(u)"
                  >
                    {{ u.is_banned ? '解封' : '封禁' }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && list.length === 0">
              <td colspan="9"><EmptyState text="暂无用户" /></td>
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
    <n-modal v-model:show="balanceModal" preset="card" title="调整余额" style="width: 420px">
      <n-form label-placement="left" label-width="90">
        <n-form-item label="用户">
          <span class="text-14">{{ balanceTarget?.email }}</span>
        </n-form-item>
        <n-form-item label="金额(元)">
          <n-input-number
            v-model:value="balanceAmount"
            placeholder="可正可负,如 10 或 -5"
            class="w-full"
          />
        </n-form-item>
        <n-form-item label="备注">
          <n-input v-model:value="balanceRemark" placeholder="选填,写入审计日志" />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-ghost h-9 px-4 text-14" @click="balanceModal = false">取消</button>
          <button
            class="btn-primary h-9 px-4 text-14"
            :disabled="balanceSaving"
            @click="saveBalance"
          >
            保存
          </button>
        </div>
      </template>
    </n-modal>

    <!-- 角色调整弹窗 -->
    <n-modal v-model:show="roleModal" preset="card" title="调整角色" style="width: 360px">
      <n-form label-placement="left" label-width="60">
        <n-form-item label="用户">
          <span class="text-14">{{ roleTarget?.email }}</span>
        </n-form-item>
        <n-form-item label="角色">
          <n-select
            v-model:value="roleValue"
            :options="[
              { label: '普通用户', value: 0 },
              { label: '管理员', value: 1 },
              { label: '代理商', value: 2 },
            ]"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button class="btn-ghost h-9 px-4 text-14" @click="roleModal = false">取消</button>
          <button class="btn-primary h-9 px-4 text-14" :disabled="roleSaving" @click="saveRole">
            保存
          </button>
        </div>
      </template>
    </n-modal>
  </div>
</template>
