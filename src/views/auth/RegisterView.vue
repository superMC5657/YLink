<script setup lang="ts">
/**
 * 注册页 —— 数据:POST /captcha/email、POST /auth/register(docs/api/README.md §3.2/§4.1)
 */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { apiCaptcha } from '@/api/user'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useCountdown } from '@/composables/useCountdown'
import type { FormInst, FormRules } from 'naive-ui'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()
const { remaining, running, start } = useCountdown(60)

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const sending = ref(false)

const form = ref({
  email: '',
  email_code: '',
  password: '',
  confirm_password: '',
  invite_code: '',
})

// URL ?code= 自动填充邀请码
onMounted(() => {
  const code = route.query.code as string | undefined
  if (code) form.value.invite_code = code
})

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
  confirm_password: [
    {
      required: true,
      trigger: ['blur', 'input'],
      validator: (_r, v: string) => {
        if (v !== form.value.password) return new Error(t('auth.passwordMismatch'))
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
    await apiCaptcha.sendEmail({ email: form.value.email, type: 'register' })
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
    await auth.register({
      email: form.value.email,
      password: form.value.password,
      email_code: form.value.email_code,
      invite_code: form.value.invite_code || undefined,
    })
    message.success(t('auth.registerSuccess'))
    router.replace('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="mb-1 text-20 font-700 text-[var(--c-text)]">{{ t('auth.welcomeRegister') }}</h2>
    <p class="mb-6 text-13 text-[var(--c-text-sub)]">{{ t('auth.email') }} + {{ t('auth.password') }}</p>

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

      <n-form-item path="confirm_password">
        <n-input
          v-model:value="form.confirm_password"
          type="password"
          show-password-on="click"
          :placeholder="t('auth.confirmPassword')"
        >
          <template #prefix><AppIcon name="shield-check" :size="16" /></template>
        </n-input>
      </n-form-item>

      <n-form-item path="invite_code">
        <n-input v-model:value="form.invite_code" :placeholder="t('auth.inviteCode')">
          <template #prefix><AppIcon name="gift" :size="16" /></template>
        </n-input>
      </n-form-item>

      <button class="btn-primary h-11 w-full text-15" :disabled="loading" @click="onSubmit">
        <span v-if="loading" class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
        {{ t('auth.register') }}
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
