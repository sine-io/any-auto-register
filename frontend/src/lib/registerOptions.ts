export const EXECUTOR_OPTIONS = [
  { value: 'protocol', label: '纯协议' },
  { value: 'headless', label: '无头浏览器' },
  { value: 'headed', label: '有头浏览器' },
] as const

export interface PlatformMeta {
  name: string
  display_name: string
  version: string
  supported_executors?: string[]
  available?: boolean
  availability_reason?: string
}

function findPlatform(platform: string | undefined, platforms: PlatformMeta[] = []) {
  if (!platform) return undefined
  return platforms.find((item) => item.name === platform)
}

export function getDefaultPlatform(platforms: PlatformMeta[] = [], preferred = 'trae') {
  const availablePlatforms = platforms.filter((item) => item.available !== false)
  if (availablePlatforms.some((item) => item.name === preferred)) return preferred
  if (availablePlatforms[0]?.name) return availablePlatforms[0].name
  if (platforms[0]?.name) return platforms[0].name
  return preferred
}

export function getSupportedExecutors(platform?: string, platforms: PlatformMeta[] = []) {
  if (!platform) return ['protocol']
  return findPlatform(platform, platforms)?.supported_executors || ['protocol']
}

export function getExecutorOptions(platform?: string, platforms: PlatformMeta[] = []) {
  const supported = new Set(getSupportedExecutors(platform, platforms))
  return EXECUTOR_OPTIONS.filter((option) => supported.has(option.value))
}

export function normalizeExecutorForPlatform(
  platform: string | undefined,
  executor: string | undefined,
  platforms: PlatformMeta[] = [],
) {
  const supported = getSupportedExecutors(platform, platforms)
  if (executor && supported.includes(executor)) return executor
  return supported[0] || 'protocol'
}
