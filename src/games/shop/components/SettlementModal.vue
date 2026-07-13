<script setup lang="ts">
/**
 * 房主专属：本局结算 Modal。
 *
 * - 选择「撤离成功」/「炸单」（炸单默认 +60W 保底，撤离需手动填扣减额）。
 * - 为房间内全部 3 位玩家（含老板）填写本局战绩：击杀 / 小红 / 6 格 / 9 格及以上。
 * - 实时预览每位玩家本次结算的金币奖励与新的保底金额。
 * - 「确认结算」发送 `SETTLE_ROUND` Intent，关闭 Modal；服务端拒绝时通过全局 toast 反馈。
 *
 * 设计要点：
 *  - 仅在 props.open=true 时渲染，并在每次打开时重置表单，避免上次输入残留。
 *  - 所有金币 / 保底变化都由服务端计算，前端仅做预览（最终以 ROUND_SETTLED 为准）。
 *  - 不阻塞底层服务端事件（FeedStream / 资产卡片仍会实时更新）。
 */
import { computed, ref, watch } from 'vue';
import { useRoomStore } from '../stores/room';
import type { SettlementOutcome } from '../net/protocol';
import { ROLE_LABEL } from '../game/rules';
import Avatar from '@shared/components/Avatar.vue';

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();

const roomStore = useRoomStore();

type StatKey = 'kills' | 'smallReds' | 'sixGrids' | 'nineGrids';
type StatRow = Record<StatKey, string>;
const STAT_KEYS: StatKey[] = ['kills', 'smallReds', 'sixGrids', 'nineGrids'];

const STAT_MAX = 99;
const GUARANTEE_MAX = 999_999;

const outcome = ref<SettlementOutcome>('EVACUATED');
const guaranteeInput = ref<string>('');
const stats = ref<Record<string, StatRow>>({});
// 护航专属：是否救助板板（true 时 +1 币）。
const rescues = ref<Record<string, boolean>>({});
const submitting = ref(false);

const players = computed(() => roomStore.players);
const currentGuarantee = computed(() => roomStore.guaranteedAmount);

function isEscort(playerId: string): boolean {
  return players.value.find((p) => p.id === playerId)?.role === 'ESCORT';
}

function freshRow(): StatRow {
  return { kills: '0', smallReds: '0', sixGrids: '0', nineGrids: '0' };
}

function resetForm(): void {
  outcome.value = 'EVACUATED';
  // 撤离成功无默认值，强制房主输入扣减额；炸单默认 60。
  guaranteeInput.value = '';
  submitting.value = false;
  const next: Record<string, StatRow> = {};
  const nextRescues: Record<string, boolean> = {};
  for (const p of players.value) {
    next[p.id] = freshRow();
    nextRescues[p.id] = false;
  }
  stats.value = next;
  rescues.value = nextRescues;
}

watch(
  () => props.open,
  (open) => {
    if (open) resetForm();
  },
  { immediate: true },
);

watch(outcome, (next) => {
  guaranteeInput.value = next === 'BOMBED' ? '60' : '';
});

/** 解析输入为非负整数；空串 / 非整数返回 NaN。 */
function parseNonNegInt(v: string): number {
  const trimmed = v.trim();
  if (trimmed === '') return Number.NaN;
  if (!/^\d+$/.test(trimmed)) return Number.NaN;
  return Number(trimmed);
}

const guaranteeDelta = computed<number | null>(() => {
  const n = parseNonNegInt(guaranteeInput.value);
  return Number.isNaN(n) ? null : n;
});

const newGuarantee = computed<number | null>(() => {
  if (guaranteeDelta.value === null) return null;
  return outcome.value === 'EVACUATED'
    ? currentGuarantee.value - guaranteeDelta.value
    : currentGuarantee.value + guaranteeDelta.value;
});

function statValue(playerId: string, key: StatKey): number {
  const s = stats.value[playerId];
  if (!s) return 0;
  const v = parseNonNegInt(s[key]);
  return Number.isNaN(v) ? -1 : v;
}

function previewCoinDelta(playerId: string): number | null {
  let delta = 0;
  if (isEscort(playerId)) {
    // 护航：仅击杀 + 救助板板计币，不再通过小红 / 6 格 / 9 格获得币。
    const kills = statValue(playerId, 'kills');
    if (kills < 0 || kills > STAT_MAX) return null;
    delta = kills * coinWeight('kills');
    if (rescues.value[playerId]) delta += 1;
  } else {
    // 板板（老板）：完整战绩计币。
    for (const k of STAT_KEYS) {
      const v = statValue(playerId, k);
      if (v < 0 || v > STAT_MAX) return null;
      delta += v * coinWeight(k);
    }
  }
  if (outcome.value === 'EVACUATED') delta += 3;
  return delta;
}

function coinWeight(k: StatKey): number {
  switch (k) {
    case 'kills':
      return 1;
    case 'smallReds':
      return 1;
    case 'sixGrids':
      return 2;
    case 'nineGrids':
      return 3;
  }
}

const validation = computed<{ ok: boolean; reason: string }>(() => {
  if (guaranteeDelta.value === null) {
    return { ok: false, reason: '请输入合法的保底变动（非负整数）' };
  }
  if (guaranteeDelta.value > GUARANTEE_MAX) {
    return { ok: false, reason: `保底变动不可逾越 ${GUARANTEE_MAX}` };
  }
  if (newGuarantee.value === null) {
    return { ok: false, reason: '' };
  }
  // 撤离扣减允许结算后保底为负数，不做下限校验。
  if (newGuarantee.value > GUARANTEE_MAX) {
    return { ok: false, reason: `裁决后保底超过上限 ${GUARANTEE_MAX}W` };
  }
  for (const p of players.value) {
    for (const k of STAT_KEYS) {
      const v = statValue(p.id, k);
      if (v < 0) {
        return { ok: false, reason: `${p.nickname} 的字段须填 0 ~ ${STAT_MAX} 之整数` };
      }
      if (v > STAT_MAX) {
        return { ok: false, reason: `${p.nickname} 的字段逾越 ${STAT_MAX}` };
      }
    }
  }
  return { ok: true, reason: '' };
});

const canSubmit = computed(
  () =>
    !submitting.value &&
    validation.value.ok &&
    roomStore.isHost &&
    roomStore.roomStatus === 'PLAYING',
);

function submit(): void {
  if (!canSubmit.value) return;
  submitting.value = true;
  roomStore.sendIntent('SETTLE_ROUND', {
    outcome: outcome.value,
    guaranteeDelta: guaranteeDelta.value!,
    perPlayer: players.value.map((p) => {
      const escort = isEscort(p.id);
      return {
        playerId: p.id,
        kills: statValue(p.id, 'kills'),
        // 护航不再通过小红 / 6 格 / 9 格计币，统一上报 0。
        smallReds: escort ? 0 : statValue(p.id, 'smallReds'),
        sixGrids: escort ? 0 : statValue(p.id, 'sixGrids'),
        nineGrids: escort ? 0 : statValue(p.id, 'nineGrids'),
        rescue: escort ? !!rescues.value[p.id] : false,
      };
    }),
  });
  emit('close');
}

const STAT_HEADERS: { key: StatKey; label: string; suffix: string }[] = [
  { key: 'kills', label: '击杀', suffix: '人' },
  { key: 'smallReds', label: '小红', suffix: '个' },
  { key: 'sixGrids', label: '6 格', suffix: '件' },
  { key: 'nineGrids', label: '9+ 格', suffix: '件' },
];
</script>

<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4"
      @click.self="emit('close')"
    >
      <div class="hls-panel w-[95%] max-w-2xl max-h-[90vh] flex flex-col animate-pixel-slide">
        <div class="hls-panel-title">本轮结算</div>

        <div class="flex-1 overflow-y-auto px-5 py-5 space-y-6">
          <section>
            <div class="font-pixel text-[11px] uppercase tracking-[0.32em] text-hls-gold/85 mb-2">
              ❖ 本轮结果
            </div>
            <div class="grid grid-cols-2 gap-3">
              <button
                type="button"
                class="judgment-card"
                :class="outcome === 'EVACUATED' ? 'is-evacuated' : ''"
                @click="outcome = 'EVACUATED'"
              >
                <div class="flex items-center gap-2">
                  <span class="text-2xl">🛡️</span>
                  <div class="font-pixel tracking-[0.2em] text-lg uppercase text-hls-btn-green">
                    撤离成功
                  </div>
                </div>
                <div class="font-body text-[11px] mt-1 text-hls-text-muted">
                  保底 -X · 全员奖励 3 鼠币
                </div>
              </button>
              <button
                type="button"
                class="judgment-card"
                :class="outcome === 'BOMBED' ? 'is-bombed' : ''"
                @click="outcome = 'BOMBED'"
              >
                <div class="flex items-center gap-2">
                  <span class="text-2xl">💀</span>
                  <div class="font-pixel tracking-[0.2em] text-lg uppercase text-red-400">
                    炸单翻车
                  </div>
                </div>
                <div class="font-body text-[11px] mt-1 text-hls-text-muted">
                  保底 +X · 默认 +60W
                </div>
              </button>
            </div>
          </section>

          <section>
            <label class="font-pixel text-[11px] uppercase tracking-[0.32em] text-hls-gold/85 mb-2 block">
              ❖ {{ outcome === 'EVACUATED' ? '保底扣减（W）' : '保底增加（W）' }}
            </label>
            <div class="flex items-center gap-2">
              <input
                v-model="guaranteeInput"
                inputmode="numeric"
                pattern="[0-9]*"
                class="input flex-1"
                :placeholder="outcome === 'EVACUATED' ? '例如 200' : '60'"
              />
              <span class="font-pixel text-sm text-hls-gold whitespace-nowrap">
                当前保底 {{ currentGuarantee }}W
              </span>
            </div>
            <div
              v-if="newGuarantee !== null"
              class="font-body text-xs mt-2"
              :class="newGuarantee > GUARANTEE_MAX ? 'text-red-400' : 'text-hls-text-muted'"
            >
              结算后保底：
              <span class="font-pixel font-bold text-hls-text-light">{{ newGuarantee }}W</span>
              <span class="ml-1 text-hls-text-muted">
                ({{ outcome === 'EVACUATED' ? '−' : '+' }}{{ guaranteeDelta }}W)
              </span>
            </div>
          </section>

          <section>
            <div class="font-pixel text-[11px] uppercase tracking-[0.32em] text-hls-gold/85 mb-2">
              ❖ 三人本轮记录
            </div>
            <div class="rounded-md overflow-hidden border border-hls-border/30">
              <table class="w-full text-sm">
                <thead>
                  <tr class="bg-hls-bg/80">
                    <th class="text-left font-pixel text-[11px] uppercase tracking-[0.18em] text-hls-gold/85 px-3 py-2">成员</th>
                    <th
                      v-for="h in STAT_HEADERS"
                      :key="h.key"
                      class="text-center font-pixel text-[11px] uppercase tracking-[0.18em] text-hls-gold/85 px-2 py-2 w-20"
                    >
                      {{ h.label }}
                    </th>
                    <th class="text-right font-pixel text-[11px] uppercase tracking-[0.18em] text-hls-gold/85 px-3 py-2 w-20">所得</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="p in players"
                    :key="p.id"
                    class="border-t border-hls-border/15"
                  >
                    <td class="px-3 py-2">
                      <div class="flex items-center gap-2 min-w-0">
                        <Avatar :seed="p.id" :name="p.nickname" size="sm" />
                        <div class="min-w-0">
                          <div class="font-pixel font-semibold text-hls-text-light truncate flex items-center gap-1">
                            <span v-if="p.isHost" class="text-hls-gold">👑</span>
                            {{ p.nickname }}
                          </div>
                          <span
                            class="hls-badge text-[10px]"
                            :class="
                              p.role === 'BOSS'
                                ? '!border-hls-gold/60 !text-hls-gold'
                                : '!border-hls-text-nav/60 !text-hls-text-nav'
                            "
                          >
                            {{ p.role ? ROLE_LABEL[p.role] : '' }}
                          </span>
                        </div>
                      </div>
                    </td>
                    <template v-if="p.role === 'ESCORT'">
                      <td class="px-2 py-2">
                        <input
                          v-model="stats[p.id].kills"
                          inputmode="numeric"
                          pattern="[0-9]*"
                          class="stat-input"
                        />
                      </td>
                      <td colspan="3" class="px-2 py-2">
                        <div class="flex items-center justify-center gap-2">
                          <span class="font-pixel text-[11px] text-hls-text-muted whitespace-nowrap">救助板板</span>
                          <button
                            type="button"
                            class="rescue-toggle"
                            :class="rescues[p.id] ? 'is-yes' : ''"
                            @click="rescues[p.id] = true"
                          >
                            是
                          </button>
                          <button
                            type="button"
                            class="rescue-toggle"
                            :class="!rescues[p.id] ? 'is-no' : ''"
                            @click="rescues[p.id] = false"
                          >
                            否
                          </button>
                        </div>
                      </td>
                    </template>
                    <template v-else>
                      <td v-for="h in STAT_HEADERS" :key="h.key" class="px-2 py-2">
                        <input
                          v-model="stats[p.id][h.key]"
                          inputmode="numeric"
                          pattern="[0-9]*"
                          class="stat-input"
                        />
                      </td>
                    </template>
                    <td class="px-3 py-2 text-right font-pixel font-bold">
                      <template v-if="previewCoinDelta(p.id) === null">
                        <span class="text-red-400 text-xs">逆数</span>
                      </template>
                      <template v-else>
                        <span class="text-hls-btn-green">+{{ previewCoinDelta(p.id) }}</span>
                      </template>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="rounded-md border border-hls-border/25 bg-hls-bg/55 px-4 py-3 text-xs leading-relaxed">
            <div class="font-pixel tracking-[0.18em] uppercase text-hls-gold/90 mb-1">
              ❖ 结算规则
            </div>
            <div class="font-body text-hls-gold/90">
              <span class="font-pixel tracking-wide">板板</span>：击杀 1 人奖励 1 鼠币，摸到小红奖励 1 鼠币，6 格奖励 2 鼠币，9 格及以上奖励 3 鼠币，非洲之心 / 海洋之泪奖励 10 鼠币
            </div>
            <div class="font-body text-hls-text-nav/90 mt-0.5">
              <span class="font-pixel tracking-wide">护航</span>：击杀 1 人奖励 1 鼠币，救助板板奖励 1 鼠币
            </div>
            <div class="font-body text-hls-btn-green/90 mt-0.5">
              撤离成功 · 全员奖励 3 鼠币
            </div>
          </section>
        </div>

        <div class="p-4 flex gap-3 justify-end border-t border-hls-border/30">
          <div class="font-body text-xs text-red-400 min-h-[1em] flex items-center mr-auto">
            {{ validation.ok ? '' : validation.reason }}
          </div>
          <button class="hls-btn-secondary py-2 px-4" @click="emit('close')">取消</button>
          <button
            class="hls-btn py-2 px-4"
            :disabled="!canSubmit"
            @click="submit"
          >
            {{ submitting ? '结算中…' : '⚖ 确认结算' }}
          </button>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.judgment-card {
  border-radius: 6px;
  padding: 14px 16px;
  text-align: left;
  transition: all 0.2s ease;
  background:
    linear-gradient(180deg, rgba(212, 175, 55, 0.05), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.25);
  box-shadow: inset 0 1px 0 rgba(244, 230, 194, 0.06);
}
.judgment-card:hover {
  border-color: rgba(232, 200, 124, 0.6);
}
.judgment-card.is-evacuated {
  border-color: rgba(90, 122, 50, 0.85);
  background:
    linear-gradient(180deg, rgba(90, 122, 50, 0.18), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #14200a, #0a1505);
  box-shadow:
    inset 0 1px 0 rgba(136, 160, 79, 0.18),
    0 0 0 1px rgba(90, 122, 50, 0.55),
    0 0 18px -4px rgba(90, 122, 50, 0.5);
}
.judgment-card.is-bombed {
  border-color: rgba(160, 35, 35, 0.85);
  background:
    linear-gradient(180deg, rgba(160, 35, 35, 0.18), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #2a0c0c, #1a0606);
  box-shadow:
    inset 0 1px 0 rgba(193, 58, 58, 0.2),
    0 0 0 1px rgba(160, 35, 35, 0.55),
    0 0 18px -4px rgba(160, 35, 35, 0.5);
}

.rescue-toggle {
  min-width: 44px;
  border-radius: 4px;
  padding: 5px 12px;
  font-family: 'VT323', monospace;
  font-size: 0.85rem;
  color: #cbb78a;
  background:
    linear-gradient(180deg, rgba(0, 0, 0, 0.35), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.3);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.45);
  transition: all 0.18s ease;
}
.rescue-toggle:hover {
  border-color: rgba(232, 200, 124, 0.7);
}
.rescue-toggle.is-yes {
  color: #bfe0a0;
  border-color: rgba(90, 122, 50, 0.85);
  background:
    linear-gradient(180deg, rgba(90, 122, 50, 0.22), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #14200a, #0a1505);
  box-shadow:
    inset 0 1px 0 rgba(136, 160, 79, 0.18),
    0 0 0 1px rgba(90, 122, 50, 0.5);
}
.rescue-toggle.is-no {
  color: #e6c0c0;
  border-color: rgba(160, 35, 35, 0.7);
  background:
    linear-gradient(180deg, rgba(160, 35, 35, 0.18), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #2a0c0c, #1a0606);
  box-shadow:
    inset 0 1px 0 rgba(193, 58, 58, 0.18),
    0 0 0 1px rgba(160, 35, 35, 0.45);
}

.stat-input {
  width: 100%;
  border-radius: 4px;
  background:
    linear-gradient(180deg, rgba(0, 0, 0, 0.35), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.3);
  color: #f4e6c2;
  text-align: center;
  font-family: 'VT323', monospace;
  font-size: 0.875rem;
  padding: 6px 8px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.45);
}
.stat-input:focus {
  outline: none;
  border-color: rgba(232, 200, 124, 0.85);
  box-shadow:
    inset 0 2px 4px rgba(0, 0, 0, 0.45),
    0 0 0 1px rgba(212, 175, 55, 0.4);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
