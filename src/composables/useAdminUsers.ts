import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiAdmin } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import type { AdminRole, AdminUserItem } from '@/types/api'
import { useMessage, useDialog } from 'naive-ui'
import { maskUrl } from '@/utils/format'
import { copyText } from '@/utils/platform'

/**
 * 用户管理数据域(F05):搜索/分页、封禁/角色/余额、多选批量、导出/发邮件/重置订阅。
 * 纯逻辑与状态;弹窗呈现拆在 components/admin/*Dialog.vue,表格留在视图。
 */
export function useAdminUsers() {
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

  // ---- F05:多选 / 批量操作 / 导出 / 发邮件 / 重置订阅 ----

  const selectedIds = ref<number[]>([])

  function isSelected(id: number): boolean {
    return selectedIds.value.includes(id)
  }

  function toggleRow(id: number, checked: boolean) {
    if (checked) {
      if (!selectedIds.value.includes(id)) selectedIds.value = [...selectedIds.value, id]
    } else {
      selectedIds.value = selectedIds.value.filter((v) => v !== id)
    }
  }

  const allChecked = computed(
    () => list.value.length > 0 && list.value.every((u) => selectedIds.value.includes(u.id)),
  )

  function toggleAll(checked: boolean) {
    selectedIds.value = checked ? list.value.map((u) => u.id) : []
  }

  function clearSelection() {
    selectedIds.value = []
  }

  /** 批量结果汇总提示:成功 n,失败列出原因 */
  function reportBatch(
    title: string,
    resp: { success: number; failed: { id: number; reason: string }[] },
  ) {
    if (resp.failed.length === 0) {
      message.success(`${title}: ${resp.success}`)
    } else {
      const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
      message.warning(
        `${title}: ${resp.success} / ${t('adminUsers.failed')} ${resp.failed.length} — ${detail}`,
      )
    }
  }

  const batchSaving = ref(false)

  function batchBan(banned: boolean) {
    const ids = [...selectedIds.value]
    if (ids.includes(auth.user?.id ?? -1)) {
      message.warning(t('adminUsers.cannotOperateSelf'))
      return
    }
    dialog.warning({
      title: banned ? t('adminUsers.batchBan') : t('adminUsers.batchUnban'),
      content: t('adminUsers.batchBanConfirm', { count: ids.length }),
      positiveText: t('adminUsers.ok'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        batchSaving.value = true
        try {
          const resp = await apiAdmin.batchUsers({ action: banned ? 'ban' : 'unban', ids })
          reportBatch(t('adminUsers.batchDone'), resp)
          clearSelection()
          void load()
        } finally {
          batchSaving.value = false
        }
      },
    })
  }

  // 批量调余额弹窗
  const batchBalanceModal = ref(false)
  const batchBalanceAmount = ref<number | null>(null)
  const batchBalanceRemark = ref('')

  function openBatchBalance() {
    batchBalanceAmount.value = null
    batchBalanceRemark.value = ''
    batchBalanceModal.value = true
  }

  async function saveBatchBalance() {
    if (batchBalanceAmount.value === null) return
    batchSaving.value = true
    try {
      const resp = await apiAdmin.batchUsers({
        action: 'adjust_balance',
        ids: [...selectedIds.value],
        amount: batchBalanceAmount.value,
        remark: batchBalanceRemark.value,
      })
      reportBatch(t('adminUsers.batchBalanceDone'), resp)
      batchBalanceModal.value = false
      clearSelection()
      void load()
    } finally {
      batchSaving.value = false
    }
  }

  // 发邮件弹窗
  const mailModal = ref(false)
  const mailSubject = ref('')
  const mailBody = ref('')
  const mailSaving = ref(false)

  function openSendMail() {
    mailSubject.value = ''
    mailBody.value = ''
    mailModal.value = true
  }

  async function saveSendMail() {
    if (!mailSubject.value || !mailBody.value) return
    mailSaving.value = true
    try {
      const resp = await apiAdmin.sendMail({
        ids: [...selectedIds.value],
        subject: mailSubject.value,
        body: mailBody.value,
      })
      if (resp.failed.length === 0) {
        message.success(t('adminUsers.mailSent', { sent: resp.sent }))
      } else {
        const detail = resp.failed.map((f) => `#${f.id} ${f.reason}`).join('; ')
        message.warning(
          t('adminUsers.mailPartial', { sent: resp.sent, failed: resp.failed.length }) +
            ` — ${detail}`,
        )
      }
      mailModal.value = false
      clearSelection()
    } finally {
      mailSaving.value = false
    }
  }

  // CSV 导出(blob → 触发浏览器保存)
  const exporting = ref(false)

  async function exportCSV() {
    exporting.value = true
    try {
      const blob = await apiAdmin.exportUsersCSV({ keyword: keyword.value || '' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `users_${new Date().toISOString().slice(0, 10)}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      exporting.value = false
    }
  }

  // 重置订阅密钥
  function resetSub(u: AdminUserItem) {
    dialog.warning({
      title: t('adminUsers.resetSub'),
      content: t('adminUsers.resetSubConfirm', { email: u.email }),
      positiveText: t('adminUsers.ok'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        const resp = await apiAdmin.resetUserSubToken(u.id)
        dialog.info({
          title: t('adminUsers.resetSubDone'),
          // 订阅链接掩码展示:原文含令牌,不整体进弹窗文本;复制按钮复制完整原文
          content: maskUrl(resp.subscribe_url),
          positiveText: t('adminUsers.copy'),
          onPositiveClick: () => copyText(resp.subscribe_url),
        })
      },
    })
  }

  onMounted(() => void load())

  return {
    // 状态
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
    // 工具
    roleLabel,
    allChecked,
    // 动作
    load,
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
  }
}
