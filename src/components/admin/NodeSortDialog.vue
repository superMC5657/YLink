<script setup lang="ts">
/**
 * 节点排序弹窗(呈现层):上下移与保存动作交回 useAdminNodes 处理。
 */
import { useI18n } from 'vue-i18n'

defineProps<{
  show: boolean
  items: { id: number; name: string }[]
  saving: boolean
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  move: [index: number, delta: number]
  save: []
}>()

const { t } = useI18n()
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('adminNodes.sortTitle')"
    style="width: 420px"
    @update:show="emit('update:show', $event)"
  >
    <p class="mb-3 text-14 text-[var(--c-text-sub)]">{{ t('adminNodes.sortHint') }}</p>
    <div class="flex flex-col gap-2">
      <div
        v-for="(item, i) in items"
        :key="item.id"
        class="flex items-center justify-between rounded-lg border border-[var(--c-border)] px-3 py-2"
      >
        <span class="num-font mr-2 text-14 text-[var(--c-text-sub)]">{{ i + 1 }}</span>
        <span class="flex-1 truncate text-14 font-500 text-[var(--c-text)]">{{ item.name }}</span>
        <div class="flex gap-1">
          <button
            class="btn-soft-neutral h-7 px-2 text-14"
            :disabled="i === 0"
            @click="emit('move', i, -1)"
          >
            ↑
          </button>
          <button
            class="btn-soft-neutral h-7 px-2 text-14"
            :disabled="i === items.length - 1"
            @click="emit('move', i, 1)"
          >
            ↓
          </button>
        </div>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button class="btn-soft-neutral h-9 px-4 text-14" @click="emit('update:show', false)">
          {{ t('common.cancel') }}
        </button>
        <button class="btn-primary h-9 px-4 text-14" :disabled="saving" @click="emit('save')">
          {{ t('adminNodes.sortSave') }}
        </button>
      </div>
    </template>
  </n-modal>
</template>
