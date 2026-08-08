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
import { useCountdown } from '@/composables/useCountdown'
import type { FormInst, FormRules } from 'naive-ui'

const router = useRouter()
const message = useMessage()
const { t } = useI18n()
const { remaining, running, start } = useCountdown(60)

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const sending = ref(false)
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
        if (!/^\d{6}$/.test(v)) return new Error('请输入 6 位验证码')
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

async function sendCode() {
  if (sending.value || running.value) return
  // naive-ui 2.42 类型未暴露 validateField,运行时存在,局部断言调用
  const inst = formRef.value as unknown as { validateField: (p: string) => Promise<void> } | null
  try {
    await inst?.validateField('email')
  } catch {
    return
  }
  sending.value = true
  try {
    await apiCaptcha.sendEmail({ email: form.value.email, type: 'forgot' })
    message.success(t('auth.codeSent'))
    start(60)
  } finally {
    sending.value = false
  }
}

async function onSubmit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await apiAuth.forgot(form.value)
    message.success(t('auth.forgotSuccess'))
    router.replace('/login')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="mb-1 text-20 font-700 text-[var(--c-text)]">{{ t('auth.welcomeForgot') }}</h2>
    <p class="mb-6 text-13 text-[var(--c-text-sub)]">
      {{ t('auth.email') }} + {{ t('auth.emailCode') }}
    </p>

    <n-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="onSubmit">
      <n-form-item path="email">
        <n-input v-model:value="form.email" :placeholder="t('auth.email')">
          <template #prefix><AppIcon name="user" :size="16" /></template>
        </n-input>
      </n-form-item>

      <n-form-item path="email_code">
        <div class="flex w-full gap-2">
          <n-input v-model:value="form.email_code" :placeholder="t('auth.emailCode')" maxlength="6">
            <template #prefix><AppIcon name="shield-check" :size="16" /></template>
          </n-input>
          <button
            class="btn-ghost h-10 shrink-0 px-4 text-13"
            :disabled="sending || running"
            @click="sendCode"
          >
            {{ running ? t('auth.resendAfter', { n: remaining }) : t('auth.sendCode') }}
          </button>
        </div>
      </n-form-item>

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

      <button class="btn-primary h-11 w-full text-15" :disabled="loading" @click="onSubmit">
        <span
          v-if="loading"
          class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
        />
        {{ t('auth.forgot') }}
      </button>
    </n-form>

    <p class="mt-5 text-center text-13 text-[var(--c-text-sub)]">
      {{ t('auth.toLogin') }}
      <router-link to="/login" class="text-[var(--c-primary-text)] hover:underline">
        {{ t('auth.login') }}
      </router-link>
    </p>
  </div>
</template>
