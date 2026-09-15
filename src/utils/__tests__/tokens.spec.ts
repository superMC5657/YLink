import { beforeEach, describe, expect, it } from 'vitest'
import {
  applyTokens,
  clearTokens,
  getAccessToken,
  getRefreshToken,
  restorePersisted,
  setMemoryTokens,
} from '../tokens'
import { getItem, removeItem, setItem } from '../storage'

describe('tokens 最小持久面', () => {
  beforeEach(() => {
    removeItem('auth')
    clearTokens()
  })

  it('applyTokens:持久化只含 refreshToken/user/savedAt,不含 accessToken', () => {
    applyTokens('access-1', 'refresh-1', { id: 1, email: 'a@b.c', role: 0 } as never)
    expect(getAccessToken()).toBe('access-1')
    expect(getRefreshToken()).toBe('refresh-1')
    const persisted = getItem<Record<string, unknown>>('auth')
    expect(persisted?.refreshToken).toBe('refresh-1')
    expect(persisted?.accessToken).toBeUndefined()
    expect(typeof persisted?.savedAt).toBe('number')
  })

  it('restorePersisted:恢复 refresh 与 user,不恢复 access', () => {
    // 模拟刷新后:内存令牌已失,持久化仍在
    setItem('auth', {
      refreshToken: 'refresh-1',
      user: { id: 1, email: 'a@b.c', role: 1 },
      savedAt: Date.now(),
    })
    const restored = restorePersisted()
    expect(restored.refreshToken).toBe('refresh-1')
    expect(restored.user?.email).toBe('a@b.c')
    expect(getAccessToken()).toBe('')
    expect(getRefreshToken()).toBe('refresh-1')
  })

  it('过期清理:savedAt 超过保留期,恢复时丢弃并清持久化', () => {
    const expired = Date.now() - 31 * 24 * 60 * 60 * 1000
    setItem('auth', { refreshToken: 'old', user: null, savedAt: expired })
    const restored = restorePersisted()
    expect(restored.refreshToken).toBe('')
    expect(getItem('auth')).toBeNull()
  })

  it('无 savedAt 视为过期脏数据:恢复时丢弃并清持久化(需重新登录)', () => {
    setItem('auth', { refreshToken: 'legacy', user: null })
    const restored = restorePersisted()
    expect(restored.refreshToken).toBe('')
    expect(restored.user).toBeNull()
    expect(getItem('auth')).toBeNull()
  })

  it('clearTokens 清内存与持久化;setMemoryTokens 不写持久化', () => {
    applyTokens('a', 'r')
    setMemoryTokens('a2', 'r2')
    expect(getAccessToken()).toBe('a2')
    clearTokens()
    expect(getAccessToken()).toBe('')
    expect(getRefreshToken()).toBe('')
    expect(getItem('auth')).toBeNull()
  })
})
