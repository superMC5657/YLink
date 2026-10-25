<script setup lang="ts">
/**
 * 403 页:无权访问(角色不足)。纯前端呈现,后端 admin 接口仍以 40300 强制兜底。
 * 骨架归 ui/ErrorPage。
 */
import ErrorPage from '@/components/ui/ErrorPage.vue'
</script>

<template>
  <ErrorPage
    code="403"
    gradient="linear-gradient(135deg, var(--c-danger), var(--c-warning))"
    title-key="forbidden.title"
    desc-key="forbidden.desc"
    back-key="forbidden.back"
  />
</template>
