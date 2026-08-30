<script setup lang="ts">
/**
 * 申请代理(截图5):左侧申请卡 + 右侧条件/特权/注意事项。
 * 数据:GET /agent/status、POST /agent/apply(契约 §12);内容取站点配置 agent_policy
 */
import { computed, onMounted } from 'vue'
import { useAgentStore } from '@/stores/agent'
import { useConfigStore } from '@/stores/config'
import { useMessage } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import StatusBadge from '@/components/ui/StatusBadge.vue'

const agent = useAgentStore()
const config = useConfigStore()
const message = useMessage()
const { t } = useI18n()

const policy = computed(() => config.config?.agent_policy)

const status = computed(() => agent.status)

/** 按钮状态:未满足=灰禁用 / 可申请=主色 / 审核中=warning / 已是代理=success */
const applyBtn = computed(() => {
  const s = status.value
  if (!s) return { disabled: true, text: t('agent.apply'), type: 'primary' as const }
  if (s.is_agent) return { disabled: true, text: t('agent.approved'), type: 'success' as const }
  if (s.apply_status === 'pending')
    return { disabled: true, text: t('agent.applying'), type: 'warning' as const }
  if (s.apply_status === 'rejected')
    return { disabled: false, text: t('agent.apply'), type: 'primary' as const }
  if (!s.qualified)
    return { disabled: true, text: t('agent.notQualified'), type: 'neutral' as const }
  return { disabled: false, text: t('agent.apply'), type: 'primary' as const }
})

async function onApply() {
  try {
    await agent.apply()
    message.success(t('agent.applied'))
  } catch {
    // 错误提示由 http 层统一 toast,这里仅阻止异常冒泡为 unhandled error
  }
}

onMounted(() => {
  void agent.fetchStatus()
  // 政策文案(特权/注意事项)来自站点配置,进页强制刷新以反映管理后台最新改动
  void config.fetchConfig(true)
})
</script>

<template>
  <div>
    <h1 class="mb-5 text-20 font-600 text-[var(--c-text)]">{{ t('agent.title') }}</h1>

    <div class="flex flex-col gap-5 lg:flex-row">
      <!-- 左:申请卡 -->
      <div class="w-full shrink-0 lg:w-90">
        <div class="card-base flex flex-col items-center p-8 text-center">
          <span
            class="flex h-18 w-18 items-center justify-center rounded-full text-white shadow-lg"
            style="background: linear-gradient(135deg, #6558f5, #8b5cf6)"
          >
            <AppIcon name="award" :size="34" />
          </span>
          <h2 class="mt-4 text-20 font-700 text-[var(--c-text)]">{{ t('agent.becomeAgent') }}</h2>
          <p class="mt-2 text-14 text-[var(--c-text-sub)]">{{ t('agent.agentDesc') }}</p>

          <div class="mt-3 flex flex-wrap items-center justify-center gap-2">
            <StatusBadge v-if="status?.is_agent" type="success">{{
              t('agent.approved')
            }}</StatusBadge>
            <StatusBadge v-if="status?.apply_status === 'pending'" type="warning">{{
              t('agent.applying')
            }}</StatusBadge>
          </div>

          <div
            class="mt-6 w-full rounded-xl p-4 text-left"
            style="background-color: var(--c-bg-hover)"
          >
            <div class="flex justify-between text-14">
              <span class="text-[var(--c-text-sub)]">{{ t('invite.registeredUsers') }}</span>
              <span class="num font-600 text-[var(--c-text)]">
                {{ status?.valid_invites ?? 0 }} / {{ status?.required_valid_invites ?? 0 }}
              </span>
            </div>
            <div
              class="mt-2 h-2 w-full overflow-hidden rounded-full"
              style="background-color: var(--c-border)"
            >
              <div
                class="h-full rounded-full transition-all duration-500"
                :style="{
                  width: `${Math.min(100, ((status?.valid_invites ?? 0) / Math.max(1, status?.required_valid_invites ?? 1)) * 100)}%`,
                  background: 'linear-gradient(90deg, #6558F5, #8B5CF6)',
                }"
              />
            </div>
          </div>

          <button
            class="mt-6 h-11 w-full rounded-[var(--r-control)] text-14 font-500 transition-all active:scale-98"
            :class="{
              'bg-[var(--c-bg-hover)] text-[var(--c-text-sub)] cursor-not-allowed':
                applyBtn.disabled,
              'btn-primary': !applyBtn.disabled,
            }"
            :style="
              applyBtn.type === 'warning'
                ? { backgroundColor: 'var(--c-warning-bg)', color: 'var(--c-warning)' }
                : applyBtn.type === 'success'
                  ? { backgroundColor: 'var(--c-success-bg)', color: 'var(--c-success)' }
                  : {}
            "
            :disabled="applyBtn.disabled"
            @click="onApply"
          >
            {{ applyBtn.text }}
          </button>
        </div>
      </div>

      <!-- 右:条件 / 特权 / 注意事项 -->
      <div class="min-w-0 flex-1 space-y-5">
        <!-- 加盟条件 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('agent.conditions') }}</h3>
          <div class="space-y-3">
            <div
              v-for="(cond, i) in status?.conditions ?? []"
              :key="i"
              class="flex items-start gap-3"
            >
              <span
                class="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full"
                :style="
                  cond.met
                    ? { backgroundColor: 'var(--c-success-bg)', color: 'var(--c-success)' }
                    : { backgroundColor: 'var(--c-bg-hover)', color: 'var(--c-text-sub)' }
                "
              >
                <AppIcon :name="cond.met ? 'check' : 'close'" :size="14" :stroke-width="2.5" />
              </span>
              <span class="text-14 leading-6 text-[var(--c-text)]">{{ cond.text }}</span>
            </div>
            <div v-if="!status?.conditions?.length" class="text-14 text-[var(--c-text-sub)]">-</div>
          </div>
        </div>

        <!-- 代理商特权 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('agent.benefits') }}</h3>
          <div class="grid gap-3 sm:grid-cols-2">
            <div
              v-for="(b, i) in policy?.benefits ?? []"
              :key="i"
              class="flex items-center gap-3 rounded-xl p-3.5"
              style="background-color: var(--c-bg-hover)"
            >
              <span
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full"
                style="background: var(--c-primary-soft); color: var(--c-primary-text)"
              >
                <AppIcon name="award" :size="17" />
              </span>
              <span class="text-14 font-500 text-[var(--c-text)]">{{ b }}</span>
            </div>
          </div>
        </div>

        <!-- 注意事项 -->
        <div class="card-base p-5 md:p-6">
          <h3 class="mb-4 text-16 font-600 text-[var(--c-text)]">{{ t('agent.notes') }}</h3>
          <ol class="space-y-2.5">
            <li
              v-for="(n, i) in policy?.notes ?? []"
              :key="i"
              class="flex items-start gap-3 text-14 leading-6 text-[var(--c-text-sub)]"
            >
              <span
                class="num mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-14 font-600"
                style="background: var(--c-primary-soft); color: var(--c-primary-text)"
              >
                {{ i + 1 }}
              </span>
              {{ n }}
            </li>
          </ol>
        </div>
      </div>
    </div>
  </div>
</template>
