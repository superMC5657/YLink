<script setup lang="ts">
/**
 * 用户调余额弹窗(呈现层):金额/备注状态由 useAdminUsers 持有。
 */
import { useI18n } from 'vue-i18n'
import type { AdminUserItem } from '@/types/api'

defineProps<{
  show: boolean
  target: AdminUserItem | null
  saving: boolean
}>()

const amount = defineModel<number | null>('amount', { required: true })
const remark = defineModel<string>('remark', { required: true })

const emit = defineEmits<{
  'update:show': [boolean]
  save: []
}>()

const { t } = useI18n()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('adminUsers.adjustBalanceTitle')"
    style="width: 420px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="left" label-width="90">
      <n-form-item :label="t('adminUsers.user')">
        <span class="text-14">{{ target?.email }}</span>
      </n-form-item>
      <n-form-item :label="t('adminUsers.amount')">
        <n-input-number
          v-model:value="amount"
          :placeholder="t('adminUsers.amountPlaceholder')"
          class="w-full"
        />
      </n-form-item>
      <n-form-item :label="t('adminUsers.remark')">
        <n-input v-model:value="remark" :placeholder="t('adminUsers.remarkPlaceholder')" />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn-soft-neutral h-9 px-4 text-14" @click="emit('update:show', false)">
          {{ t('common.cancel') }}
        </button>
        <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="emit('save')">
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </n-modal>
</template>
