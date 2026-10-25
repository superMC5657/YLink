<script setup lang="ts">
/**
 * 列表排序弹窗(呈现层,节点/公告/知识库共用):上下移与保存动作交回调用方处理。
 * 文案(标题/提示/保存钮)由调用方传入,组件不绑定具体域的 i18n 键。
 */
import { useI18n } from 'vue-i18n'

defineProps<{
  show: boolean
  items: { id: number; name: string }[]
  saving: boolean
  title: string
  hint: string
  saveText: string
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
    :title="title"
    style="width: 420px"
    @update:show="emit('update:show', $event)"
  >
    <p class="mb-3 text-14 text-[var(--c-text-sub)]">{{ hint }}</p>
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
          {{ saveText }}
        </button>
      </div>
    </template>
  </n-modal>
</template>
