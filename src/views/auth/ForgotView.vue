<script setup lang="ts">
/**
 * 找回密码页 —— 数据:POST /captcha/email、POST /auth/forgot(docs/api/README.md §4.4)
 */
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiAuth } from '@/api/auth'
import { apiCaptcha } from '@/api/user'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import EmailCodeField from './EmailCodeField.vue'
import type { FormInst, FormItemInst, FormRules } from 'naive-ui'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const formRef = ref<FormInst | null>(null)
// email 表单项实例:发送验证码前单独校验邮箱(n-form 实例无 validateField,用 form-item 的 validate)
const emailItemRef = ref<FormItemInst | null>(null)
const loading = ref(false)
const form = ref({ email: '', email_code: '', password: '' })

const rules = computed<FormRules>(() => ({
  email: [
    {
      required: true,
      trigger: ['blur', 'input'],
      validator: (_r, v: string) => {
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return new Error(t('auth.invalidEmail'))
        return true
      },
    },
  ],
  email_code: [
    {
      required: true,
      trigger: ['blur', 'input'],
      validator: (_r, v: string) => {
        if (!/^\d{6}$/.test(v)) return new Error(t('auth.emailCodeInvalid'))
        return true
      },
    },
  ],
  password: [
    {
      required: true,
      trigger: ['blur', 'input'],
      validator: (_r, v: string) => {
        if (!/^(?=.*[A-Za-z])(?=.*\d).{8,}$/.test(v)) return new Error(t('auth.passwordRule'))
        return true
      },
    },
  ],
}))

async function onSubmit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await apiAuth.forgot(form.value)
    message.success(t('auth.forgotSuccess'))
    router.replace('/login')
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="mb-1 text-20 font-700 text-[var(--c-text)]">{{ t('auth.welcomeForgot') }}</h2>
    <p class="mb-6 text-14 text-[var(--c-text-sub)]">
      {{ t('auth.email') }} + {{ t('auth.emailCode') }}
    </p>

    <n-form
      ref="formRef"
      :model="form"
      :rules="rules"
      size="large"
      class="forgot-form"
      :show-feedback="false"
      @keyup.enter="onSubmit"
    >
      <n-form-item ref="emailItemRef" path="email">
        <n-input v-model:value="form.email" :placeholder="t('auth.email')">
          <template #prefix><AppIcon name="user" :size="16" /></template>
        </n-input>
      </n-form-item>

      <EmailCodeField
        v-model:value="form.email_code"
        :email-item="emailItemRef"
        :get-email="() => form.email"
        :send="(email) => apiCaptcha.sendEmail({ email, type: 'forgot' })"
      />

      <n-form-item path="password">
        <n-input
          v-model:value="form.password"
          type="password"
          show-password-on="click"
          :placeholder="t('auth.password')"
        >
          <template #prefix><AppIcon name="shield-check" :size="16" /></template>
        </n-input>
      </n-form-item>

      <button
        type="button"
        class="btn-primary mt-9 h-11 w-full text-15"
        :disabled="loading"
        @click="onSubmit"
      >
        <span
          v-if="loading"
          class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
        />
        {{ t('auth.forgot') }}
      </button>
    </n-form>

    <p class="mt-5 text-center text-14 text-[var(--c-text-sub)]">
      {{ t('auth.toLogin') }}
      <router-link to="/login" class="text-[var(--c-primary-text)]">
        {{ t('auth.login') }}
      </router-link>
    </p>
  </div>
</template>

<style scoped>
/* 收紧找回表单纵向间距,与登录/注册页(login-form/register-form)保持一致(pages.md §3.2) */
.forgot-form :deep(.n-form-item) {
  margin-bottom: 4px;
}
</style>
