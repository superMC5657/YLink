<script setup lang="ts">
/**
 * 用户角色调整弹窗(呈现层):目标用户与角色值由 useAdminUsers 持有。
 */
import { useI18n } from 'vue-i18n'
import type { AdminRole, AdminUserItem } from '@/types/api'

defineProps<{
  show: boolean
  target: AdminUserItem | null
  saving: boolean
}>()

const role = defineModel<AdminRole>('role', { required: true })

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
    :title="t('adminUsers.adjustRoleTitle')"
    style="width: 360px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="left" label-width="60">
      <n-form-item :label="t('adminUsers.user')">
        <span class="text-14">{{ target?.email }}</span>
      </n-form-item>
      <n-form-item :label="t('adminUsers.role')">
        <n-select
          v-model:value="role"
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
