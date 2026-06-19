/**
 * 抽奖元数据：
 *   - 抽奖随机算法已迁移到 Go 后端 `internal/game/lottery.go`（crypto/rand）。
 *   - 抽奖奖品池（11 件，名称 / `prizeId` / 价值 / 稀有度 / 权重）由服务端权威配置，
 *     前端只持有 UI 文案 / 颜色 / 价格常量与展示工具，不预置奖品池静态副本。
 *
 * 当前后端的稀有度档位简化为 HIGH / MEDIUM / LOW；为兼容历史的
 * COMMON/RARE/EPIC/LEGENDARY 数据，文案 / 颜色映射两套都保留。
 */

export type Rarity =
  | 'HIGH'
  | 'MEDIUM'
  | 'LOW'
  | 'COMMON'
  | 'RARE'
  | 'EPIC'
  | 'LEGENDARY'
  | string;

export const SINGLE_DRAW_COST = 2;

export const RARITY_LABEL: Record<string, string> = {
  HIGH: '高价值',
  MEDIUM: '中价值',
  LOW: '低价值',
  COMMON: '普通',
  RARE: '稀有',
  EPIC: '史诗',
  LEGENDARY: '传说',
};

export const RARITY_COLOR: Record<string, string> = {
  HIGH: 'text-red-400',
  MEDIUM: 'text-gold-400',
  LOW: 'text-purple-400',
  COMMON: 'text-slate-300',
  RARE: 'text-sky-400',
  EPIC: 'text-purple-400',
  LEGENDARY: 'text-amber-400',
};

/**
 * 以"W"为单位渲染奖品价值或保底金额。
 * 例：`formatValueW(215)` -> `"215W"`。
 */
export function formatValueW(value: number | undefined | null): string {
  const v = Number(value ?? 0);
  return Number.isFinite(v) ? `${v}W` : '0W';
}

/**
 * 渲染保底金额的变化：`+215W（2500 → 2715）`。
 * `delta == 0` 时返回空串，避免 UI 误显示无变化。
 */
export function formatGuaranteeShift(
  delta: number | undefined | null,
  before: number | undefined | null,
  after: number | undefined | null,
): string {
  const d = Number(delta ?? 0);
  if (!Number.isFinite(d) || d === 0) return '';
  const sign = d > 0 ? '+' : '';
  return `${sign}${d}W（${Number(before ?? 0)} → ${Number(after ?? 0)}）`;
}
