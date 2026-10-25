<script setup lang="ts">
/**
 * 404 页:插画 + 返回首页按钮(骨架归 ui/ErrorPage)。
 */
import ErrorPage from '@/components/ui/ErrorPage.vue'
</script>

<template>
  <ErrorPage
    code="404"
    gradient="linear-gradient(135deg, var(--c-primary), var(--c-primary-grad-end))"
    title-key="notFound.title"
    desc-key="notFound.desc"
    back-key="notFound.backHome"
  />
</template>
