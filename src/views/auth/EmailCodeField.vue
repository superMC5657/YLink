<script setup lang="ts">
/**
 * 认证页邮箱验证码输入行(注册/找回共用):验证码输入 + 发送按钮。
 * 发送前先校验 email 表单项(实例经 prop 传入),倒计时/发送态内置;
 * apiCaptcha 直调留在引导页视图(data-layer.md §2 认证视图例外),经 send 注入。
 */
import { useI18n } from 'vue-i18n'
import type { FormItemInst } from 'naive-ui'
import { useEmailCode } from '@/composables/useEmailCode'

const value = defineModel<string>('value', { required: true })

const props = defineProps<{
  /** email 表单项实例:发送验证码前单独校验邮箱 */
  emailItem: FormItemInst | null
  getEmail: () => string
  send: (email: string) => Promise<unknown>
}>()

const { t } = useI18n()
const { remaining, running, sending, sendCode } = useEmailCode({
  getEmailItem: () => props.emailItem,
  getEmail: props.getEmail,
  send: props.send,
})
</script>

<template>
  <n-form-item path="email_code">
    <div class="flex w-full gap-2">
      <n-input v-model:value="value" :placeholder="t('auth.emailCode')" maxlength="6">
        <template #prefix><AppIcon name="shield-check" :size="16" /></template>
      </n-input>
      <button
        type="button"
        class="btn-soft-primary h-10 shrink-0 px-4 text-14"
        :disabled="sending || running"
        @click="sendCode"
      >
        {{ running ? t('auth.resendAfter', { n: remaining }) : t('auth.sendCode') }}
      </button>
    </div>
  </n-form-item>
</template>
