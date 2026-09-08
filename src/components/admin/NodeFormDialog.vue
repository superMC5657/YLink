<script setup lang="ts">
/* eslint-disable vue/no-mutating-props --
   白名单:本弹窗为呈现层,直接绑定 useAdminNodes 持有的同一 reactive 表单对象,
   字段更新即状态更新,不存在"子组件私改父数据"的边界问题 */
/**
 * 节点新建/编辑弹窗(呈现层):表单状态由 useAdminNodes 持有,经 props 传入。
 */
import { useI18n } from 'vue-i18n'
import type { AdminServerGroupItem, AdminServerReq } from '@/types/api'

defineProps<{
  show: boolean
  form: AdminServerReq
  groups: AdminServerGroupItem[]
  saving: boolean
  editing: number | null
}>()

const emit = defineEmits<{
  'update:show': [boolean]
  save: []
}>()

const { t } = useI18n()

const PROTOCOLS = [
  { label: 'Shadowsocks', value: 'shadowsocks' },
  { label: 'VMess', value: 'vmess' },
  { label: 'VLESS', value: 'vless' },
  { label: 'Trojan', value: 'trojan' },
  { label: 'Hysteria2', value: 'hysteria2' },
  { label: 'Tuic', value: 'tuic' },
]
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="editing === null ? t('adminNodes.createNode') : t('adminNodes.editNode')"
    style="width: 560px"
    @update:show="emit('update:show', $event)"
  >
    <n-form label-placement="top">
      <div class="grid grid-cols-2 gap-x-4">
        <n-form-item :label="t('adminNodes.name')">
          <n-input v-model:value="form.name" :placeholder="t('adminNodes.namePlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.group')">
          <n-select
            v-model:value="form.group_id"
            :options="groups.map((g) => ({ label: g.name, value: g.id }))"
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.protocol')">
          <n-select v-model:value="form.type" :options="PROTOCOLS" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.rate')">
          <n-input-number v-model:value="form.rate" :min="0.1" :step="0.1" class="w-full" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.hostLabel')">
          <n-input v-model:value="form.host" :placeholder="t('adminNodes.hostPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.portLabel')">
          <n-input-number v-model:value="form.port" :min="1" :max="65535" class="w-full" />
        </n-form-item>
        <n-form-item :label="t('adminNodes.status')">
          <n-select
            v-model:value="form.status"
            :options="[
              { label: t('node.normal'), value: 1 },
              { label: t('node.busy'), value: 2 },
              { label: t('node.maintenance'), value: 3 },
            ]"
          />
        </n-form-item>
        <n-form-item :label="t('adminNodes.sort')">
          <n-input-number v-model:value="form.sort" class="w-full" />
        </n-form-item>
      </div>
      <n-form-item :label="t('adminNodes.tagsLabel')">
        <n-select
          v-model:value="form.tags"
          multiple
          filterable
          tag
          :options="[]"
          :placeholder="t('adminNodes.tagsPlaceholder')"
        />
      </n-form-item>
      <n-form-item :label="t('adminNodes.configLabel')">
        <n-input
          v-model:value="form.config"
          type="textarea"
          :rows="4"
          placeholder='{"password":"...","method":"aes-256-gcm"}'
        />
      </n-form-item>
      <n-form-item :label="t('adminNodes.showLabel')">
        <n-switch v-model:value="form.is_show" />
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
