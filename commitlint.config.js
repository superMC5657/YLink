// commitlint 配置：遵循 Conventional Commits（docs/README §5）
// 项目为 ESM（package.json "type": "module"），配置文件用 export default。
// 允许 scope 为空（如 `fix: xxx`）；类型白名单由 @commitlint/config-conventional 提供
export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,
      'always',
      [
        'feat',
        'fix',
        'docs',
        'style',
        'refactor',
        'perf',
        'test',
        'build',
        'ci',
        'chore',
        'revert',
      ],
    ],
  },
}
