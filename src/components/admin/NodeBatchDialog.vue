<script setup lang="ts">
/**
 * 节点批量修改公共字段弹窗(呈现层):留空字段不更新,状态由 useAdminNodes 持有。
 */
import { useI18n } from 'vue-i18n'
import type { AdminServerGroupItem, ServerStatus } from '@/types/api'

defineProps<{
  show: boolean
  groups: AdminServerGroupItem[]
  selectedCount: number
  saving: boolean
}>()

const status = defineModel<ServerStatus | null>('status', { required: true })
const group = defineModel<number | null>('group', { required: true })
const rate = defineModel<number | null>('rate', { required: true })

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
    :title="t('adminNodes.batchUpdate')"
    style="width: 420px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <n-form-item :label="t('adminNodes.batchUpdateHint')">
        <span class="text-14 text-[var(--c-text-sub)]">
          {{ t('adminNodes.selectedCount', { count: selectedCount }) }}
        </span>
      </n-form-item>
      <n-form-item :label="t('adminNodes.status')">
        <n-select
          v-model:value="status"
          clearable
          :options="[
            { label: t('node.normal'), value: 1 },
            { label: t('node.busy'), value: 2 },
            { label: t('node.maintenance'), value: 3 },
          ]"
          :placeholder="t('adminNodes.batchUnchanged')"
        />
      </n-form-item>
      <n-form-item :label="t('adminNodes.group')">
        <n-select
          v-model:value="group"
          clearable
          :options="groups.map((g) => ({ label: g.name, value: g.id }))"
          :placeholder="t('adminNodes.batchUnchanged')"
        />
      </n-form-item>
      <n-form-item :label="t('adminNodes.rate')">
        <n-input-number
          v-model:value="rate"
          clearable
          :min="0.1"
          :step="0.1"
          class="w-full"
          :placeholder="t('adminNodes.batchUnchanged')"
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
