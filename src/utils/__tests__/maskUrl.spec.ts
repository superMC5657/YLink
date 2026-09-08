import { describe, expect, it } from 'vitest'
import { maskUrl } from '../format'

describe('maskUrl 订阅链接掩码', () => {
  it('保留协议+域名,主体以 •••• + 末 4 位示意', () => {
    const url = 'https://sub.example.com/api/v1/sub?token=AbCdEf123456'
    expect(maskUrl(url)).toBe('https://sub.example.com/••••3456')
    expect(maskUrl(url)).not.toContain('AbCdEf')
    expect(maskUrl(url)).not.toContain('/api/v1')
  })

  it('非标准 URL(无 path/无 scheme)原样返回', () => {
    expect(maskUrl('https://sub.example.com')).toBe('https://sub.example.com')
    expect(maskUrl('clash://install-config?url=x')).toBe('clash://install-config?url=x')
  })
})
