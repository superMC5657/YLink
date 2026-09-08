import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import prettier from 'eslint-config-prettier'

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'coverage/**',
      'test-results/**',
      'playwright-report/**',
      'src-tauri/**',
      'server/**',
      '*.config.ts',
      'src/vite-env.d.ts',
      'scripts/**',
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: ['.vue'],
      },
    },
  },
  {
    files: ['**/*.{ts,vue}'],
    languageOptions: {
      globals: {
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        localStorage: 'readonly',
        fetch: 'readonly',
        console: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
        crypto: 'readonly',
        btoa: 'readonly',
        AbortController: 'readonly',
        RequestInit: 'readonly',
        URLSearchParams: 'readonly',
      },
    },
    rules: {
      // TS 编译器已做类型检查(含 DOM 类型),关闭 no-undef 避免误报 HTMLElement 等
      'no-undef': 'off',
      'vue/multi-word-component-names': 'off',
      'vue/require-default-prop': 'off',
      // v-html 白名单制:仅限已 DOMPurify 过滤的 markdown 渲染与 AppIcon 内联 path,
      // 新增使用必须带 eslint-disable-next-line vue/no-v-html + 安全理由注释
      'vue/no-v-html': 'error',
      // 先 warn 起步:现有代码零 any,新增 any 会在 CI lint(--max-warnings 0)暴露
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      'no-unused-vars': 'off',
    },
  },
  prettier,
)
