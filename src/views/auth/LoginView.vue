<script setup lang="ts">
/**
 * 登录页 —— 数据:POST /auth/login(docs/api/README.md §4.2)
 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import type { FormInst, FormRules } from 'naive-ui'
import { removeItem } from '@/utils/storage'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const message = useMessage()
const { t } = useI18n()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
// 登录页不预填演示账号(真实后端账号以注册为准)
const form = ref({ email: '', password: '' })

/** localStorage 残留过自定义后端地址时,显示「重置后端地址」自救入口 */
const hasCustomApiBase = computed(() => localStorage.getItem('app:apiBase') !== null)

function resetApiBase() {
  removeItem('apiBase')
  window.location.reload()
}

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
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <h2 class="mb-1 text-20 font-700 text-[var(--c-text)]">{{ t('auth.welcomeBack') }}</h2>
    <p class="mb-2 text-14 text-[var(--c-text-sub)]">
      {{ t('auth.email') }} / {{ t('auth.password') }}
    </p>

    <n-form
      ref="formRef"
      :model="form"
      :rules="rules"
      size="large"
      class="login-form"
      :show-feedback="false"
      @keyup.enter="onSubmit"
    >
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

      <div class="mt-1 mb-3 text-right">
        <router-link to="/forgot" class="text-14 text-[var(--c-primary-text)]">
          {{ t('auth.forgotLink') }}
        </router-link>
      </div>

      <button
        type="button"
        class="btn-primary h-11 w-full text-15"
        :disabled="loading"
        @click="onSubmit"
      >
        <span
          v-if="loading"
          class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
        />
        {{ t('auth.login') }}
      </button>
    </n-form>

    <p class="mt-4 text-center text-14 text-[var(--c-text-sub)]">
      {{ t('auth.toRegister') }}
      <router-link to="/register" class="text-[var(--c-primary-text)]">
        {{ t('auth.register') }}
      </router-link>
    </p>

    <!-- 自救入口:localStorage 残留自定义后端地址导致「网络异常」时,一键重置 -->
    <div v-if="hasCustomApiBase" class="mt-3 text-center">
      <button
        class="text-14 text-[var(--c-text-sub)] hover:text-[var(--c-danger)]"
        @click="resetApiBase"
      >
        {{ t('auth.resetApiBase') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
/* 收紧登录表单纵向间距:naive form-item 默认 feedback 占位 + 大留白 */
.login-form :deep(.n-form-item) {
  margin-bottom: 4px;
}
</style>
