const PALETTE = [
  '#f97316', '#3b82f6', '#10b981', '#a855f7', '#ec4899',
  '#facc15', '#06b6d4', '#ef4444', '#22c55e', '#8b5cf6',
];

function hash(str: string): number {
  let h = 5381;
  for (let i = 0; i < str.length; i++) {
    h = ((h << 5) + h) ^ str.charCodeAt(i);
  }
  return Math.abs(h);
}

/** 根据稳定的 ID 生成头像背景色，避免不同会话中漂移。 */
export function avatarColor(seed: string): string {
  return PALETTE[hash(seed) % PALETTE.length];
}

/** 取昵称首字（中英兼容）。 */
export function avatarLetter(nickname: string): string {
  if (!nickname) return '?';
  const trimmed = nickname.trim();
  if (!trimmed) return '?';
  return Array.from(trimmed)[0]!.toUpperCase();
}
