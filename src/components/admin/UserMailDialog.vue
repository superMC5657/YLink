<script setup lang="ts">
/**
 * 批量发送邮件弹窗(呈现层):主题/正文状态由 useAdminUsers 持有。
 */
import { useI18n } from 'vue-i18n'

defineProps<{
  show: boolean
  selectedCount: number
  saving: boolean
  subject: string
  body: string
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  'update:subject': [string]
  'update:body': [string]
  save: []
}>()

const { t } = useI18n()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('adminUsers.sendMail')"
    style="width: 520px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="left" label-width="90">
      <n-form-item :label="t('adminUsers.users')">
        <span class="text-14">{{ t('adminUsers.selectedCount', { count: selectedCount }) }}</span>
      </n-form-item>
      <n-form-item :label="t('adminUsers.mailSubject')">
        <n-input :value="subject" :maxlength="200" @update:value="emit('update:subject', $event)" />
      </n-form-item>
      <n-form-item :label="t('adminUsers.mailBody')">
        <n-input
          :value="body"
          type="textarea"
          :rows="6"
          :maxlength="10000"
          show-count
          @update:value="emit('update:body', $event)"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn-soft-neutral h-9 px-4 text-14" @click="emit('update:show', false)">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn-primary h-9 px-4 text-14"
          :disabled="saving || !subject || !body"
          @click="emit('save')"
        >
          <AppIcon name="send" :size="14" /> {{ t('common.submit') }}
        </button>
      </div>
    </template>
  </n-modal>
</template>
