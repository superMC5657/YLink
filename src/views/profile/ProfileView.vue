<script setup lang="ts">
/**
 * 个人信息(截图7):左侧 Banner + 通知设置 + Telegram + 会话管理;右侧 改密 + 重置订阅。
 * 数据:PUT /user/profile、POST /user/password/change、POST /user/subscribe/reset、
 * GET/DELETE /user/sessions(契约 §5,F14)
 */
import { computed, onMounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { useConfigStore } from '@/stores/config'
import { useMessage, useDialog } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { openExternal } from '@/utils/platform'
import { copyText } from '@/utils/platform'
import { isTauri } from '@/utils/platform'
import { checkForUpdate, requestCheckUpdate } from '@/utils/updater'
import { apiUser } from '@/api/user'
import { formatTime } from '@/utils/format'
import type { UserSessionItem } from '@/types/api'
import BannerStatCard from '@/components/business/BannerStatCard.vue'

const user = useUserStore()
const config = useConfigStore()
const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

// ---------- 通知设置 ----------
const remindExpire = ref(true)
const remindTraffic = ref(false)
const savingNotify = ref(false)

async function onNotifyChange() {
  if (savingNotify.value) return
  savingNotify.value = true
  try {
    const data = await user.updateProfile({
      remind_expire: remindExpire.value,
      remind_traffic: remindTraffic.value,
    })
    remindExpire.value = data.remind_expire
    remindTraffic.value = data.remind_traffic
    message.success(t('common.save') + ' ✓')
  } finally {
    savingNotify.value = false
  }
}

// ---------- 修改密码 ----------
const pwdForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const savingPwd = ref(false)

function clearPwd() {
  pwdForm.value = { old_password: '', new_password: '', confirm_password: '' }
}

const canSavePwd = computed(
  () =>
    pwdForm.value.old_password.length >= 6 &&
    pwdForm.value.new_password.length >= 8 &&
    pwdForm.value.new_password === pwdForm.value.confirm_password,
)

async function savePwd() {
  if (savingPwd.value) return
  savingPwd.value = true
  try {
    await user.changePassword({
      old_password: pwdForm.value.old_password,
      new_password: pwdForm.value.new_password,
    })
    message.success(t('profile.passwordChanged'))
    clearPwd()
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    savingPwd.value = false
  }
}

// ---------- 重置订阅 ----------
const showResetModal = ref(false)
const resetPwd = ref('')
const resetting = ref(false)
const newSubscribeUrl = ref('')

async function onResetSubscribe() {
  if (resetting.value) return
  if (!resetPwd.value) {
    message.warning(t('profile.resetConfirm'))
    return
  }
  resetting.value = true
  try {
    const data = await user.resetSubscribe({ password: resetPwd.value })
    newSubscribeUrl.value = data.subscribe_url
    showResetModal.value = false
    resetPwd.value = ''
    message.success(t('profile.resetSuccess'))
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    resetting.value = false
  }
}

async function copyNewUrl() {
  await copyText(newSubscribeUrl.value)
  message.success(t('common.copied'))
}

function openGroup() {
  openExternal(config.config?.telegram?.group_url ?? '')
}
function openBot() {
  openExternal(config.config?.telegram?.bot_url ?? '')
}

// ---------- 关于与更新(仅桌面端) ----------
const appVersion = ref('')
const checkingUpdate = ref(false)

async function onCheckUpdate() {
  if (checkingUpdate.value) return
  checkingUpdate.value = true
  try {
    const update = await checkForUpdate()
    if (update) {
      requestCheckUpdate()
    } else {
      message.success(t('update.upToDate'))
    }
  } finally {
    checkingUpdate.value = false
  }
}

// ---------- Telegram 绑定（F12） ----------
const telegramBound = ref(false)
const bindModal = ref(false)
const bindCode = ref('')
const bindBot = ref('')
const gettingCode = ref(false)
const unbinding = ref(false)

async function openBindModal() {
  if (gettingCode.value) return
  gettingCode.value = true
  bindCode.value = ''
  try {
    const data = await apiUser.telegramBindCode()
    bindCode.value = data.code
    bindBot.value = data.bot_username
    bindModal.value = true
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    gettingCode.value = false
  }
}

async function copyBindCommand() {
  await copyText(`/bind ${bindCode.value}`)
  message.success(t('common.copied'))
}

function onUnbind() {
  dialog.warning({
    title: t('profile.tgUnbind'),
    content: t('profile.tgUnbindConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      if (unbinding.value) return
      unbinding.value = true
      try {
        await apiUser.telegramUnbind()
        telegramBound.value = false
        message.success(t('profile.tgUnboundDone'))
      } catch (e) {
        message.error((e as Error).message)
      } finally {
        unbinding.value = false
      }
    },
  })
}

// ---------- 会话管理（F14） ----------
const sessions = ref<UserSessionItem[]>([])
const sessionsLoading = ref(false)

async function loadSessions() {
  sessionsLoading.value = true
  try {
    const data = await apiUser.sessions()
    sessions.value = data?.list ?? []
  } finally {
    sessionsLoading.value = false
  }
}

function onRevokeSession(s: UserSessionItem) {
  dialog.warning({
    title: t('profile.sessionRevokeTitle'),
    content: t('profile.sessionRevokeConfirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await apiUser.revokeSession(s.jti)
        message.success(t('profile.sessionRevoked'))
        void loadSessions()
      } catch (e) {
        message.error((e as Error).message)
      }
    },
  })
}

onMounted(() => {
  void user.fetchStat()
  void config.fetchConfig()
  void loadSessions()
  void user.fetchProfile().then((profile) => {
    if (profile) {
      remindExpire.value = profile.remind_expire
      remindTraffic.value = profile.remind_traffic
      telegramBound.value = profile.telegram_bound ?? false
    }
  })
  // 桌面端展示当前应用版本(Web 端无此概念,保持空)
  if (isTauri()) {
    void import('@tauri-apps/api/app').then(({ getVersion }) => {
      void getVersion().then((v) => (appVersion.value = v))
    })
  }
})
</script>

<template>
  <div>
    <h1 class="mb-5 text-20 font-600 text-[var(--c-text)]">{{ t('profile.title') }}</h1>

    <div class="flex flex-col gap-5 lg:flex-row">
      <!-- 左列 -->
      <div class="min-w-0 flex-1 space-y-5">
        <BannerStatCard />

        <!-- 通知设置 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">
            {{ t('profile.notifySettings') }}
          </h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-14 font-500 text-[var(--c-text)]">
                  {{ t('profile.remindExpire') }}
                </div>
                <div class="text-14 text-[var(--c-text-sub)]">
                  {{ t('profile.remindExpireDesc') }}
                </div>
              </div>
              <n-switch v-model:value="remindExpire" @update:value="onNotifyChange" />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <div class="text-14 font-500 text-[var(--c-text)]">
                  {{ t('profile.remindTraffic') }}
                </div>
                <div class="text-14 text-[var(--c-text-sub)]">
                  {{ t('profile.remindTrafficDesc') }}
                </div>
              </div>
              <n-switch v-model:value="remindTraffic" @update:value="onNotifyChange" />
            </div>
          </div>
        </div>

        <!-- 会话管理（F14） -->
        <div class="card-base p-5 md:p-6">
          <div class="mb-4 flex items-center justify-between">
            <h3 class="text-16 font-600 text-[var(--c-text)]">{{ t('profile.sessions') }}</h3>
            <button class="btn-soft-neutral h-8 px-3 text-14" @click="loadSessions">
              <AppIcon name="refresh" :size="14" /> {{ t('common.refresh') }}
            </button>
          </div>
          <n-spin :show="sessionsLoading">
            <div class="space-y-2">
              <div
                v-for="s in sessions"
                :key="s.jti"
                class="flex items-center justify-between gap-3 rounded-xl p-3"
                style="background-color: var(--c-bg-hover)"
              >
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-14 font-500 text-[var(--c-text)]">
                      {{ s.user_agent || t('profile.sessionUnknownDevice') }}
                    </span>
                    <StatusBadge v-if="s.current" type="success">
                      {{ t('profile.sessionCurrent') }}
                    </StatusBadge>
                  </div>
                  <div class="mt-0.5 text-14 text-[var(--c-text-sub)]">
                    {{ s.ip || '--' }} · {{ s.created_at ? formatTime(s.created_at, false) : '--' }}
                  </div>
                </div>
                <button
                  v-if="!s.current"
                  class="btn-soft-danger h-8 shrink-0 px-3 text-14"
                  @click="onRevokeSession(s)"
                >
                  {{ t('profile.sessionRevoke') }}
                </button>
              </div>
              <EmptyState
                v-if="!sessionsLoading && sessions.length === 0"
                :text="t('profile.sessionEmpty')"
                icon="user"
              />
            </div>
          </n-spin>
          <p class="mt-3 text-13 text-[var(--c-text-sub)]">{{ t('profile.sessionsTip') }}</p>
        </div>

        <!-- Telegram -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('profile.telegram') }}</h3>
          <div class="grid gap-3 sm:grid-cols-2">
            <button
              class="flex cursor-pointer items-center gap-3 rounded-xl border border-[var(--c-border)] p-3.5 transition-colors hover:border-[var(--c-primary)] hover:bg-[var(--c-bg-hover)]"
              @click="openGroup"
            >
              <span
                class="flex h-10 w-10 items-center justify-center rounded-full"
                style="background: var(--c-primary-soft); color: var(--c-primary-text)"
              >
                <AppIcon name="users" :size="19" />
              </span>
              <span class="text-14 font-500 text-[var(--c-text)]">{{
                t('profile.joinGroup')
              }}</span>
              <AppIcon name="external-link" :size="15" class="ml-auto text-[var(--c-text-sub)]" />
            </button>
            <button
              class="flex cursor-pointer items-center gap-3 rounded-xl border border-[var(--c-border)] p-3.5 transition-colors hover:border-[var(--c-primary)] hover:bg-[var(--c-bg-hover)]"
              @click="openBot"
            >
              <span
                class="flex h-10 w-10 items-center justify-center rounded-full"
                style="background: var(--c-success-bg); color: var(--c-success)"
              >
                <AppIcon name="send" :size="19" />
              </span>
              <span class="text-14 font-500 text-[var(--c-text)]">{{
                t('profile.contactBot')
              }}</span>
              <AppIcon name="external-link" :size="15" class="ml-auto text-[var(--c-text-sub)]" />
            </button>
          </div>

          <!-- 账号绑定（F12） -->
          <div
            class="mt-4 flex items-center gap-3 rounded-xl border border-[var(--c-border)] p-3.5"
          >
            <span
              class="flex h-10 w-10 items-center justify-center rounded-full"
              style="background: var(--c-primary-soft); color: var(--c-primary-text)"
            >
              <AppIcon name="send" :size="19" />
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-14 font-500 text-[var(--c-text)]">{{
                  t('profile.tgBindTitle')
                }}</span>
                <StatusBadge :type="telegramBound ? 'success' : 'neutral'">
                  {{ telegramBound ? t('profile.tgBound') : t('profile.tgUnbound') }}
                </StatusBadge>
              </div>
              <p class="mt-0.5 truncate text-12 text-[var(--c-text-sub)]">
                {{ t('profile.contactBot') }}
              </p>
            </div>
            <button
              v-if="!telegramBound"
              class="btn-soft-primary h-8 shrink-0 px-3 text-14"
              :disabled="gettingCode"
              @click="openBindModal"
            >
              {{ t('profile.tgGetCode') }}
            </button>
            <button
              v-else
              class="btn-soft-neutral h-8 shrink-0 px-3 text-14"
              :disabled="unbinding"
              @click="onUnbind"
            >
              {{ t('profile.tgUnbind') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 右列 -->
      <div class="w-full shrink-0 space-y-5 lg:w-95">
        <!-- 修改密码 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-5 text-16 font-600 text-[var(--c-text)]">
            {{ t('profile.resetPassword') }}
          </h3>
          <div class="space-y-4">
            <div>
              <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{
                t('profile.oldPassword')
              }}</label>
              <input
                v-model="pwdForm.old_password"
                type="password"
                class="h-10 w-full border-b border-[var(--c-border)] bg-transparent text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
              />
            </div>
            <div>
              <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{
                t('profile.newPassword')
              }}</label>
              <input
                v-model="pwdForm.new_password"
                type="password"
                class="h-10 w-full border-b border-[var(--c-border)] bg-transparent text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
              />
            </div>
            <div>
              <label class="mb-1 block text-14 text-[var(--c-text-sub)]">{{
                t('profile.confirmNewPassword')
              }}</label>
              <input
                v-model="pwdForm.confirm_password"
                type="password"
                class="h-10 w-full border-b border-[var(--c-border)] bg-transparent text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
              />
            </div>
            <p
              v-if="pwdForm.new_password && pwdForm.new_password !== pwdForm.confirm_password"
              class="text-14 text-[var(--c-danger)]"
            >
              {{ t('auth.passwordMismatch') }}
            </p>
            <div class="flex gap-2">
              <button class="btn-soft-neutral h-9 flex-1 text-14" @click="clearPwd">
                {{ t('profile.clearForm') }}
              </button>
              <button
                class="btn-olive h-9 flex-1 text-14"
                :disabled="!canSavePwd || savingPwd"
                @click="savePwd"
              >
                {{ t('common.save') }}
              </button>
            </div>
          </div>
        </div>

        <!-- 重置订阅 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">
            {{ t('profile.resetSubscribe') }}
          </h3>
          <div
            class="mb-4 flex items-start gap-2.5 rounded-xl p-3.5"
            style="background: var(--c-warning-bg)"
          >
            <AppIcon
              name="alert"
              :size="17"
              class="mt-0.5 shrink-0"
              :style="{ color: 'var(--c-warning)' }"
            />
            <p class="text-14 leading-5 text-[var(--c-text)]">
              {{ t('profile.resetSubscribeTip') }}
            </p>
          </div>
          <button class="btn-danger h-10 w-full text-14" @click="showResetModal = true">
            <AppIcon name="refresh" :size="15" />
            {{ t('profile.resetSubscribeBtn') }}
          </button>
        </div>

        <!-- 关于与更新(仅桌面端) -->
        <div v-if="isTauri()" class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('update.checkUpdate') }}</h3>
          <div class="mb-4 flex items-center justify-between text-14">
            <span class="text-[var(--c-text-sub)]">{{ t('update.currentVersion') }}</span>
            <span class="num text-[var(--c-text)]">{{ appVersion || '--' }}</span>
          </div>
          <button
            class="btn-primary h-10 w-full text-14"
            :disabled="checkingUpdate"
            @click="onCheckUpdate"
          >
            <AppIcon name="download" :size="15" />
            {{ t('update.checkUpdate') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 重置确认弹窗 -->
    <n-modal
      v-model:show="showResetModal"
      preset="card"
      :title="t('profile.resetSubscribe')"
      class="max-w-95"
    >
      <div class="space-y-4">
        <p class="text-14 text-[var(--c-text-sub)]">{{ t('profile.resetConfirm') }}</p>
        <input
          v-model="resetPwd"
          type="password"
          :placeholder="t('auth.password')"
          class="h-10 w-full rounded-[var(--r-control)] border border-[var(--c-border)] bg-[var(--c-bg-card)] px-3 text-14 text-[var(--c-text)] outline-none transition-colors focus:border-[var(--c-primary)]"
        />
        <button
          class="btn-danger h-10 w-full text-14"
          :disabled="resetting"
          @click="onResetSubscribe"
        >
          {{ t('profile.resetSubscribeBtn') }}
        </button>
      </div>
    </n-modal>

    <!-- 新订阅链接展示 -->
    <n-modal
      :show="!!newSubscribeUrl"
      preset="card"
      :title="t('profile.newSubscribeUrl')"
      class="max-w-105"
      @update:show="
        (v: boolean) => {
          if (!v) newSubscribeUrl = ''
        }
      "
    >
      <div
        class="flex items-center gap-2 rounded-xl p-3"
        style="background-color: var(--c-bg-hover)"
      >
        <span class="num min-w-0 flex-1 break-all text-14 text-[var(--c-text)]">{{
          newSubscribeUrl
        }}</span>
        <button class="btn-primary h-9 shrink-0 px-4 text-14" @click="copyNewUrl">
          <AppIcon name="copy" :size="14" />
          {{ t('common.copy') }}
        </button>
      </div>
      <p class="mt-3 text-14 text-[var(--c-warning)]">{{ t('profile.resetSubscribeTip') }}</p>
    </n-modal>

    <!-- Telegram 绑定验证码弹窗（F12） -->
    <n-modal
      v-model:show="bindModal"
      preset="card"
      :title="t('profile.tgCodeTitle')"
      style="width: 460px"
    >
      <div class="space-y-4">
        <p class="text-14 text-[var(--c-text-sub)]">
          {{ t('profile.tgCodeTip', { bot: '@' + bindBot }) }}
        </p>
        <div class="flex items-center justify-between gap-3 rounded-xl bg-[var(--c-bg)] px-4 py-3">
          <span class="num-font text-22 font-600 tracking-widest text-[var(--c-text)]">{{
            bindCode
          }}</span>
          <button class="btn-soft-neutral h-8 px-3 text-14" @click="copyBindCommand">
            <AppIcon name="copy" :size="14" /> /bind {{ bindCode }}
          </button>
        </div>
      </div>
    </n-modal>
  </div>
</template>
