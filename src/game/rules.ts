/**
 * 前端规则模块：
 *   服务端是业务规则的权威源；这里只持有少量 *UI 展示* 用常量与判别函数，
 *   用于在按钮 disabled 提示、人数显示等场景给玩家即时反馈。
 *   所有真正的裁决都由后端 `internal/game/rules.go` 完成。
 */

import type { Player, Role } from '@/net/protocol';

export const MAX_PLAYERS = 3;
export const MAX_ESCORTS = 2;
export const MAX_BOSSES = 1;

export const ROLE_LABEL: Record<Role, string> = {
  BOSS: '灰袍法师',
  ESCORT: '守誓骑士',
};

export type RosterSummary = {
  ok: boolean;
  reason?: 'NOT_FULL' | 'ROLE_INCOMPLETE' | 'NOT_ALL_READY';
};

/**
 * 仅用于 UI 提示：在房主点击「开始游戏」前，给出按钮 disabled 文案。
 * 返回 ok=true 表示前端预判可以开始；真正的成败仍以服务端 START_GAME 回执为准。
 */
export function summarizeRoster(players: Player[]): RosterSummary {
  if (players.length !== MAX_PLAYERS) return { ok: false, reason: 'NOT_FULL' };
  const bosses = players.filter((p) => p.role === 'BOSS').length;
  const escorts = players.filter((p) => p.role === 'ESCORT').length;
  if (bosses !== MAX_BOSSES || escorts !== MAX_ESCORTS) {
    return { ok: false, reason: 'ROLE_INCOMPLETE' };
  }
  if (!players.every((p) => p.ready)) return { ok: false, reason: 'NOT_ALL_READY' };
  return { ok: true };
}
