<script setup lang="ts">
/**
 * 登录页 —— 数据:POST /auth/login(docs/api/README.md §4.2)
 */
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { FormInst, FormRules } from 'naive-ui'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const form = ref({ email: '2734921923@qq.com', password: 'Passw0rd' })

const rules: FormRules = {
  email: [{ required: true, message: t('auth.invalidEmail'), trigger: ['blur', 'input'] }],
  password: [{ required: true, message: t('auth.password'), trigger: ['blur', 'input'] }],
}

async function onSubmit() {
  if (loading.value) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await auth.login(form.value.email, form.value.password)
    message.success(t('auth.loginSuccess'))
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.replace(redirect)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="mb-1 text-20 font-700 text-[var(--c-text)]">{{ t('auth.welcomeBack') }}</h2>
    <p class="mb-6 text-13 text-[var(--c-text-sub)]">{{ t('auth.email') }} / {{ t('auth.password') }}</p>

    <n-form ref="formRef" :model="form" :rules="rules" size="large" @keyup.enter="onSubmit">
      <n-form-item path="email">
        <n-input v-model:value="form.email" :placeholder="t('auth.email')">
          <template #prefix><AppIcon name="user" :size="16" /></template>
        </n-input>
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

      <div class="mb-4 text-right">
        <router-link to="/forgot" class="text-13 text-[var(--c-primary-text)] hover:underline">
          {{ t('auth.forgotLink') }}
        </router-link>
      </div>

      <button class="btn-primary h-11 w-full text-15" :disabled="loading" @click="onSubmit">
        <span v-if="loading" class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white" />
        {{ t('auth.login') }}
      </button>
    </n-form>

    <p class="mt-5 text-center text-13 text-[var(--c-text-sub)]">
      {{ t('auth.toRegister') }}
      <router-link to="/register" class="text-[var(--c-primary-text)] hover:underline">
        {{ t('auth.register') }}
      </router-link>
    </p>
  </div>
</template>
