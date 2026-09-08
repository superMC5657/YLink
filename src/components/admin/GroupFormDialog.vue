<script setup lang="ts">
/**
 * 节点分组新建/编辑弹窗(呈现层):名称/排序状态由 useAdminNodes 持有。
 */
import { useI18n } from 'vue-i18n'

defineProps<{
  show: boolean
  saving: boolean
  editing: number | null
}>()

const name = defineModel<string>('name', { required: true })
const sort = defineModel<number>('sort', { required: true })

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
    :title="editing === null ? t('adminNodes.createGroup') : t('adminNodes.editGroup')"
    style="width: 360px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <n-form-item :label="t('adminNodes.groupNameLabel')">
        <n-input v-model:value="name" :placeholder="t('adminNodes.groupNamePlaceholder')" />
      </n-form-item>
      <n-form-item :label="t('adminNodes.sortLabel')">
        <n-input-number v-model:value="sort" class="w-full" />
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
