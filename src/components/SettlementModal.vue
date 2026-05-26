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
import { useRoomStore } from '@/stores/room';
import type { SettlementOutcome } from '@/net/protocol';
import { ROLE_LABEL } from '@/game/rules';
import Avatar from '@/components/Avatar.vue';

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
const submitting = ref(false);

const players = computed(() => roomStore.players);
const currentGuarantee = computed(() => roomStore.guaranteedAmount);

function freshRow(): StatRow {
  return { kills: '0', smallReds: '0', sixGrids: '0', nineGrids: '0' };
}

function resetForm(): void {
  outcome.value = 'EVACUATED';
  // 撤离成功无默认值，强制房主输入扣减额；炸单默认 60。
  guaranteeInput.value = '';
  submitting.value = false;
  const next: Record<string, StatRow> = {};
  for (const p of players.value) {
    next[p.id] = freshRow();
  }
  stats.value = next;
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
  for (const k of STAT_KEYS) {
    const v = statValue(playerId, k);
    if (v < 0 || v > STAT_MAX) return null;
    delta += v * coinWeight(k);
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
    return { ok: false, reason: '请输入合法的保底变化（非负整数）' };
  }
  if (guaranteeDelta.value > GUARANTEE_MAX) {
    return { ok: false, reason: `保底变化不能超过 ${GUARANTEE_MAX}` };
  }
  if (newGuarantee.value === null) {
    return { ok: false, reason: '' };
  }
  if (newGuarantee.value < 0) {
    return { ok: false, reason: '保底金额不足，扣减后不能为负' };
  }
  if (newGuarantee.value > GUARANTEE_MAX) {
    return { ok: false, reason: `结算后保底超过上限 ${GUARANTEE_MAX}W` };
  }
  for (const p of players.value) {
    for (const k of STAT_KEYS) {
      const v = statValue(p.id, k);
      if (v < 0) {
        return { ok: false, reason: `${p.nickname} 的字段必须为 0 ~ ${STAT_MAX} 整数` };
      }
      if (v > STAT_MAX) {
        return { ok: false, reason: `${p.nickname} 的字段超过 ${STAT_MAX}` };
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
    perPlayer: players.value.map((p) => ({
      playerId: p.id,
      kills: statValue(p.id, 'kills'),
      smallReds: statValue(p.id, 'smallReds'),
      sixGrids: statValue(p.id, 'sixGrids'),
      nineGrids: statValue(p.id, 'nineGrids'),
    })),
  });
  emit('close');
}

const STAT_HEADERS: { key: StatKey; label: string; suffix: string }[] = [
  { key: 'kills', label: '击杀', suffix: '人' },
  { key: 'smallReds', label: '小红', suffix: '个' },
  { key: 'sixGrids', label: '6 格', suffix: '个' },
  { key: 'nineGrids', label: '9+ 格', suffix: '个' },
];
</script>

<template>
  <transition name="fade">
    <div
      v-if="open"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      @click.self="emit('close')"
    >
      <div class="card rpg-corners w-full max-w-2xl max-h-[92vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between px-5 py-4 border-b border-gold-500/25">
          <h3 class="rpg-title text-xl">
            <span class="rune-mark mr-1">⚖</span>盘点结账
          </h3>
          <button class="btn-ghost py-1.5 px-3 text-xs" @click="emit('close')">作罢</button>
        </div>

        <div class="flex-1 overflow-y-auto px-5 py-5 space-y-6">
          <section>
            <div class="font-serif text-[11px] uppercase tracking-[0.32em] text-gold-300/85 mb-2">
              <span class="rune-mark mr-1">❖</span>本场结果
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
                  <div class="font-display tracking-[0.2em] text-lg uppercase text-forest-300">
                    顺利出仓
                  </div>
                </div>
                <div class="font-body text-[11px] mt-1 text-parchment-300/85">
                  保底 −X · 全员各入袋 +3 金
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
                  <div class="font-display tracking-[0.2em] text-lg uppercase text-oxblood-300">
                    炸单覆灭
                  </div>
                </div>
                <div class="font-body text-[11px] mt-1 text-parchment-300/85">
                  保底 +X · 默认 +60W
                </div>
              </button>
            </div>
          </section>

          <section>
            <label class="font-serif text-[11px] uppercase tracking-[0.32em] text-gold-300/85 mb-2 block">
              <span class="rune-mark mr-1">❖</span>{{ outcome === 'EVACUATED' ? '保底扣减（W）' : '保底增加（W）' }}
            </label>
            <div class="flex items-center gap-2">
              <input
                v-model="guaranteeInput"
                inputmode="numeric"
                pattern="[0-9]*"
                class="input flex-1"
                :placeholder="outcome === 'EVACUATED' ? '例如 200' : '60'"
              />
              <span class="font-serif text-sm text-gold-200 whitespace-nowrap">
                现保底 {{ currentGuarantee }}W
              </span>
            </div>
            <div
              v-if="newGuarantee !== null"
              class="font-body text-xs mt-2"
              :class="newGuarantee < 0 || newGuarantee > GUARANTEE_MAX ? 'text-oxblood-300' : 'text-parchment-300/85'"
            >
              结账之后保底：
              <span class="font-serif font-bold text-parchment-50">{{ newGuarantee }}W</span>
              <span class="ml-1 text-parchment-400/80">
                ({{ outcome === 'EVACUATED' ? '−' : '+' }}{{ guaranteeDelta }}W)
              </span>
            </div>
          </section>

          <section>
            <div class="font-serif text-[11px] uppercase tracking-[0.32em] text-gold-300/85 mb-2">
              <span class="rune-mark mr-1">❖</span>三鼠战绩
            </div>
            <div class="rounded-md overflow-hidden border border-gold-500/30">
              <table class="w-full text-sm">
                <thead>
                  <tr class="bg-ink-700/80">
                    <th class="text-left font-serif text-[11px] uppercase tracking-[0.18em] text-gold-300/85 px-3 py-2">伙计</th>
                    <th
                      v-for="h in STAT_HEADERS"
                      :key="h.key"
                      class="text-center font-serif text-[11px] uppercase tracking-[0.18em] text-gold-300/85 px-2 py-2 w-20"
                    >
                      {{ h.label }}
                    </th>
                    <th class="text-right font-serif text-[11px] uppercase tracking-[0.18em] text-gold-300/85 px-3 py-2 w-20">入袋</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="p in players"
                    :key="p.id"
                    class="border-t border-gold-500/15"
                  >
                    <td class="px-3 py-2">
                      <div class="flex items-center gap-2 min-w-0">
                        <Avatar :seed="p.id" :name="p.nickname" size="sm" />
                        <div class="min-w-0">
                          <div class="font-serif font-semibold text-parchment-50 truncate flex items-center gap-1">
                            <span v-if="p.isHost" class="text-gold-300">👑</span>
                            {{ p.nickname }}
                          </div>
                          <span
                            class="badge text-[10px]"
                            :class="
                              p.role === 'BOSS'
                                ? '!border-ember-400/60 !text-ember-300'
                                : '!border-sapphire-300/60 !text-sapphire-300'
                            "
                          >
                            {{ p.role ? ROLE_LABEL[p.role] : '' }}
                          </span>
                        </div>
                      </div>
                    </td>
                    <td v-for="h in STAT_HEADERS" :key="h.key" class="px-2 py-2">
                      <input
                        v-model="stats[p.id][h.key]"
                        inputmode="numeric"
                        pattern="[0-9]*"
                        class="stat-input"
                      />
                    </td>
                    <td class="px-3 py-2 text-right font-serif font-bold">
                      <template v-if="previewCoinDelta(p.id) === null">
                        <span class="text-oxblood-300 text-xs">非法</span>
                      </template>
                      <template v-else>
                        <span class="text-forest-300">+{{ previewCoinDelta(p.id) }}</span>
                      </template>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="rounded-md border border-gold-500/25 bg-ink-700/55 px-4 py-3 text-xs leading-relaxed">
            <div class="font-serif tracking-[0.18em] uppercase text-gold-300/90 mb-1">
              <span class="rune-mark mr-1">❖</span>分账规矩
            </div>
            <div class="font-body text-parchment-200/90">
              击杀 1 人 +1 金 · 摸 1 个小红 +1 金 · 摸 1 件 6 格 +2 金 · 摸 1 件 9+ 格 +3 金
            </div>
            <div class="font-body text-forest-300/90 mt-0.5">
              顺利出仓 · 全员各加封 +3 金
            </div>
          </section>
        </div>

        <div class="px-5 py-4 border-t border-gold-500/25 flex items-center justify-between gap-3">
          <div class="font-body text-xs text-oxblood-300 min-h-[1em]">
            {{ validation.ok ? '' : validation.reason }}
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <button class="btn-ghost py-2 px-4" @click="emit('close')">作罢</button>
            <button
              class="btn-primary py-2 px-4"
              :disabled="!canSubmit"
              @click="submit"
            >
              {{ submitting ? '盘账中…' : '⚖ 一锤定账' }}
            </button>
          </div>
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

.stat-input {
  width: 100%;
  border-radius: 4px;
  background:
    linear-gradient(180deg, rgba(0, 0, 0, 0.35), rgba(0, 0, 0, 0)),
    linear-gradient(180deg, #1a0f0a, #120804);
  border: 1px solid rgba(212, 175, 55, 0.3);
  color: #f4e6c2;
  text-align: center;
  font-family: 'Cinzel', 'Cardo', 'Noto Serif SC', serif;
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
