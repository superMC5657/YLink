<script setup lang="ts">
/**
 * 管理后台 · 节点管理:节点分组 CRUD + 节点 CRUD + 批量操作/复制/排序(F09)。
 * 数据:/admin/servers、/admin/server-groups、/admin/servers/batch|sort|{id}/copy(docs/api/README.md §16)
 * 状态与动作在 composables/useAdminNodes;四个弹窗拆在 components/admin/。
 */
import { useI18n } from 'vue-i18n'
import { useAdminNodes } from '@/composables/useAdminNodes'
import { formatBytes, formatPercent } from '@/utils/format'
import PageHeader from '@/components/ui/PageHeader.vue'
import NodeFormDialog from '@/components/admin/NodeFormDialog.vue'
import GroupFormDialog from '@/components/admin/GroupFormDialog.vue'
import NodeBatchDialog from '@/components/admin/NodeBatchDialog.vue'
import NodeSortDialog from '@/components/admin/NodeSortDialog.vue'
import NodeInstallDialog from '@/components/admin/NodeInstallDialog.vue'

const { t } = useI18n()

const {
  loading,
  servers,
  groups,
  nodeModal,
  editingNode,
  nodeSaving,
  nodeForm,
  groupModal,
  editingGroup,
  groupSaving,
  groupName,
  groupSort,
  selectedIds,
  batchSaving,
  batchModal,
  batchStatus,
  batchGroup,
  batchRate,
  sortModal,
  sortItems,
  sortSaving,
  statusText,
  statusType,
  groupNameOf,
  allChecked,
  load,
  openNodeCreate,
  openNodeEdit,
  saveNode,
  removeNode,
  resetNodeKey,
  isSelected,
  toggleRow,
  toggleAll,
  clearSelection,
  batchDelete,
  batchShow,
  openBatchUpdate,
  saveBatchUpdate,
  copyNode,
  openSortModal,
  moveSortItem,
  saveSort,
  openGroupCreate,
  openGroupEdit,
  saveGroup,
  removeGroup,
  installModal,
  installingNode,
  openInstallDialog,
} = useAdminNodes()
</script>

<template>
  <div>
    <PageHeader :title="t('adminNodes.pageTitle')" :subtitle="t('adminNodes.subtitle')">
      <template #actions>
        <div class="flex items-center gap-2">
          <button class="btn-soft-neutral h-9 px-3 text-14" @click="load">
            <AppIcon name="refresh" :size="15" /> {{ t('common.refresh') }}
          </button>
          <button class="btn-soft-primary h-9 px-4 text-14" @click="openGroupCreate">
            {{ t('adminNodes.newGroup') }}
          </button>
          <button class="btn-soft-primary h-9 px-4 text-14" @click="openSortModal">
            {{ t('adminNodes.sortTitle') }}
          </button>
          <button class="btn-primary h-9 px-4 text-14" @click="openNodeCreate">
            <AppIcon name="plus" :size="15" /> {{ t('adminNodes.newNode') }}
          </button>
        </div>
      </template>
    </PageHeader>

    <!-- F09 批量操作工具栏(选中 > 0 时显示) -->
    <div v-if="selectedIds.length > 0" class="card-base mb-4 flex flex-wrap items-center gap-2 p-3">
      <span class="text-14 text-[var(--c-text-sub)]">
        {{ t('adminNodes.selectedCount', { count: selectedIds.length }) }}
      </span>
      <button
        class="btn-soft-success h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchShow(true)"
      >
        {{ t('adminNodes.batchShow') }}
      </button>
      <button
        class="btn-soft-warning h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="batchShow(false)"
      >
        {{ t('adminNodes.batchHide') }}
      </button>
      <button
        class="btn-soft-primary h-8 px-3 text-14"
        :disabled="batchSaving"
        @click="openBatchUpdate"
      >
        {{ t('adminNodes.batchUpdate') }}
      </button>
      <button class="btn-soft-danger h-8 px-3 text-14" :disabled="batchSaving" @click="batchDelete">
        {{ t('adminNodes.batchDelete') }}
      </button>
      <button class="btn-soft-neutral h-8 px-3 text-14" @click="clearSelection">
        {{ t('adminNodes.clearSelection') }}
      </button>
    </div>

    <div class="card-base overflow-x-auto">
      <n-spin :show="loading">
        <n-table :bordered="false" :single-line="false" class="min-w-[1060px]">
          <thead>
            <tr>
              <th class="w-10">
                <n-checkbox :checked="allChecked" @update:checked="toggleAll" />
              </th>
              <th>ID</th>
              <th>{{ t('adminNodes.name') }}</th>
              <th>{{ t('adminNodes.group') }}</th>
              <th>{{ t('adminNodes.protocol') }}</th>
              <th>{{ t('adminNodes.host') }}</th>
              <th>{{ t('adminNodes.rate') }}</th>
              <th>{{ t('adminNodes.status') }}</th>
              <th>{{ t('adminNodes.show') }}</th>
              <th>{{ t('adminNodes.sort') }}</th>
              <th>{{ t('adminNodes.nodeKey') }}</th>
              <th>{{ t('common.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in servers" :key="s.id">
              <td>
                <n-checkbox
                  :checked="isSelected(s.id)"
                  @update:checked="(v: boolean) => toggleRow(s.id, v)"
                />
              </td>
              <td>{{ s.id }}</td>
              <td class="font-500 text-[var(--c-text)]">{{ s.name }}</td>
              <td class="text-14 text-[var(--c-text-sub)]">{{ groupNameOf(s.group_id) }}</td>
              <td class="text-14">{{ s.type }}</td>
              <td class="num-font text-14 text-[var(--c-text-sub)]">{{ s.host }}:{{ s.port }}</td>
              <td class="num-font">{{ s.rate }}</td>
              <td>
                <div class="flex flex-col gap-1 items-start">
                  <StatusBadge :type="statusType(s.status)">
                    {{ statusText(s.status) }}
                  </StatusBadge>
                  <n-popover v-if="s.is_online" trigger="hover" placement="top">
                    <template #trigger>
                      <span
                        class="inline-flex items-center gap-1 text-12 font-medium text-[var(--c-success)] cursor-pointer"
                      >
                        <span class="relative flex h-2 w-2">
                          <span
                            class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"
                          ></span>
                          <span
                            class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"
                          ></span>
                        </span>
                        {{ t('adminNodes.online') }}
                      </span>
                    </template>
                    <div class="space-y-1.5 text-12 p-1 min-w-[170px]">
                      <div
                        class="font-600 text-[var(--c-text)] border-b border-[var(--c-border)] pb-1 mb-1"
                      >
                        {{ t('adminNodes.probeTitle') }}
                      </div>
                      <div class="flex justify-between gap-4">
                        <span class="text-[var(--c-text-sub)]">CPU:</span>
                        <span class="font-mono font-500">{{
                          formatPercent(s.system?.cpu_percent)
                        }}</span>
                      </div>
                      <div class="flex justify-between gap-4">
                        <span class="text-[var(--c-text-sub)]">{{ t('adminNodes.memory') }}:</span>
                        <span class="font-mono font-500">{{
                          formatPercent(s.system?.memory_percent)
                        }}</span>
                      </div>
                      <div class="flex justify-between gap-4">
                        <span class="text-[var(--c-text-sub)]"
                          >{{ t('adminNodes.bandwidth') }}:</span
                        >
                        <span class="font-mono font-500"
                          >↓{{ formatBytes(s.system?.net_speed_in) }}/s ↑{{
                            formatBytes(s.system?.net_speed_out)
                          }}/s</span
                        >
                      </div>
                      <div class="flex justify-between gap-4">
                        <span class="text-[var(--c-text-sub)]"
                          >{{ t('adminNodes.connections') }}:</span
                        >
                        <span class="font-mono font-500">{{ s.system?.connections ?? 0 }}</span>
                      </div>
                      <div v-if="s.system?.uptime" class="flex justify-between gap-4">
                        <span class="text-[var(--c-text-sub)]">{{ t('adminNodes.uptime') }}:</span>
                        <span class="font-mono font-500"
                          >{{ Math.floor(s.system.uptime / 3600) }}h
                          {{ Math.floor((s.system.uptime % 3600) / 60) }}m</span
                        >
                      </div>
                    </div>
                  </n-popover>
                  <span
                    v-else
                    class="inline-flex items-center gap-1 text-12 text-[var(--c-text-sub)]"
                  >
                    <span class="h-1.5 w-1.5 rounded-full bg-gray-400"></span>
                    {{ t('adminNodes.offline') }}
                  </span>
                </div>
              </td>
              <td>
                <StatusBadge :type="s.is_show ? 'success' : 'neutral'">
                  {{ s.is_show ? t('adminNodes.show') : t('adminNodes.hidden') }}
                </StatusBadge>
              </td>
              <td class="num-font">{{ s.sort }}</td>
              <td>
                <CopyText v-if="s.node_key" :text="s.node_key" :max-chars="10" />
                <span v-else class="text-14 text-[var(--c-text-sub)]">—</span>
              </td>
              <td>
                <div class="flex gap-2">
                  <button class="btn-soft-primary h-7 px-2.5 text-14" @click="openInstallDialog(s)">
                    {{ t('adminNodes.install') }}
                  </button>
                  <button class="btn-soft-primary h-7 px-2.5 text-14" @click="openNodeEdit(s)">
                    {{ t('adminNodes.edit') }}
                  </button>
                  <button class="btn-soft-neutral h-7 px-2.5 text-14" @click="copyNode(s)">
                    {{ t('adminNodes.copy') }}
                  </button>
                  <button class="btn-soft-neutral h-7 px-2.5 text-14" @click="resetNodeKey(s)">
                    {{ t('adminNodes.resetKey') }}
                  </button>
                  <button class="btn-danger h-7 px-2.5 text-14" @click="removeNode(s)">
                    {{ t('adminNodes.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!loading && servers.length === 0">
              <td colspan="12"><EmptyState :text="t('adminNodes.emptyNodes')" /></td>
            </tr>
          </tbody>
        </n-table>
      </n-spin>
    </div>

    <!-- 分组列表 -->
    <div class="card-base mt-5 p-5">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-16 font-600 text-[var(--c-text)]">{{ t('adminNodes.groupTitle') }}</h2>
        <button class="btn-soft-primary h-8 px-3 text-14" @click="openGroupCreate">
          {{ t('adminNodes.newGroup') }}
        </button>
      </div>
      <div v-if="groups.length === 0" class="py-4">
        <EmptyState :text="t('adminNodes.emptyGroups')" />
      </div>
      <div v-else class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
        <div
          v-for="g in groups"
          :key="g.id"
          class="flex items-center justify-between rounded-[var(--r-card)] border border-[var(--c-border)] px-4 py-3"
        >
          <div>
            <div class="text-14 font-500 text-[var(--c-text)]">{{ g.name }}</div>
            <div class="text-14 text-[var(--c-text-sub)]">
              {{
                t('adminNodes.nodeCount', {
                  count: servers.filter((s) => s.group_id === g.id).length,
                })
              }}
            </div>
          </div>
          <div class="flex gap-2">
            <button class="btn-soft-primary h-7 px-2.5 text-14" @click="openGroupEdit(g)">
              {{ t('adminNodes.edit') }}
            </button>
            <button class="btn-danger h-7 px-2.5 text-14" @click="removeGroup(g)">
              {{ t('adminNodes.delete') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 节点弹窗 -->
    <NodeFormDialog
      v-model:show="nodeModal"
      :form="nodeForm"
      :groups="groups"
      :saving="nodeSaving"
      :editing="editingNode"
      @save="saveNode"
    />

    <!-- 分组弹窗 -->
    <GroupFormDialog
      v-model:show="groupModal"
      v-model:name="groupName"
      v-model:sort="groupSort"
      :saving="groupSaving"
      :editing="editingGroup"
      @save="saveGroup"
    />

    <!-- F09 批量修改公共字段弹窗 -->
    <NodeBatchDialog
      v-model:show="batchModal"
      v-model:status="batchStatus"
      v-model:group="batchGroup"
      v-model:rate="batchRate"
      :groups="groups"
      :selected-count="selectedIds.length"
      :saving="batchSaving"
      @save="saveBatchUpdate"
    />

    <!-- F09 排序弹窗 -->
    <NodeSortDialog
      v-model:show="sortModal"
      :items="sortItems"
      :saving="sortSaving"
      @move="moveSortItem"
      @save="saveSort"
    />

    <!-- F17 节点一键部署与配置弹窗 -->
    <NodeInstallDialog v-model:show="installModal" :node="installingNode" />
  </div>
</template>
