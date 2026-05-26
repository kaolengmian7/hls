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
      <div class="card w-full max-w-2xl max-h-[90vh] flex flex-col animate-slide-up">
        <div class="flex items-center justify-between px-5 py-4 border-b border-white/10">
          <h3 class="text-lg font-semibold text-slate-50">本局结算</h3>
          <button class="btn-ghost py-1 px-3 text-xs" @click="emit('close')">取消</button>
        </div>

        <div class="flex-1 overflow-y-auto px-5 py-4 space-y-5">
          <!-- A. 本局结果 -->
          <section>
            <div class="text-xs uppercase tracking-wider text-slate-400 mb-2">本局结果</div>
            <div class="grid grid-cols-2 gap-3">
              <button
                type="button"
                class="rounded-xl border px-4 py-3 text-left transition-colors"
                :class="
                  outcome === 'EVACUATED'
                    ? 'border-emerald-400/60 bg-emerald-500/10 text-emerald-200'
                    : 'border-white/10 bg-slate-900/50 text-slate-300 hover:border-emerald-400/30'
                "
                @click="outcome = 'EVACUATED'"
              >
                <div class="font-semibold">撤离成功</div>
                <div class="text-[11px] mt-0.5 text-slate-400">扣减保底 · 每人 +3 币</div>
              </button>
              <button
                type="button"
                class="rounded-xl border px-4 py-3 text-left transition-colors"
                :class="
                  outcome === 'BOMBED'
                    ? 'border-rose-400/60 bg-rose-500/10 text-rose-200'
                    : 'border-white/10 bg-slate-900/50 text-slate-300 hover:border-rose-400/30'
                "
                @click="outcome = 'BOMBED'"
              >
                <div class="font-semibold">炸单</div>
                <div class="text-[11px] mt-0.5 text-slate-400">增加保底 · 默认 +60W</div>
              </button>
            </div>
          </section>

          <!-- B. 保底变化输入 -->
          <section>
            <label class="text-xs uppercase tracking-wider text-slate-400 mb-2 block">
              {{ outcome === 'EVACUATED' ? '扣减保底（W）' : '增加保底（W）' }}
            </label>
            <div class="flex items-center gap-2">
              <input
                v-model="guaranteeInput"
                inputmode="numeric"
                pattern="[0-9]*"
                class="input flex-1"
                :placeholder="outcome === 'EVACUATED' ? '例如 200' : '60'"
              />
              <span class="text-amber-300 font-mono text-sm whitespace-nowrap">
                当前 {{ currentGuarantee }}W
              </span>
            </div>
            <div
              v-if="newGuarantee !== null"
              class="text-xs mt-1.5"
              :class="newGuarantee < 0 || newGuarantee > GUARANTEE_MAX ? 'text-rose-400' : 'text-slate-400'"
            >
              结算后保底：
              <span class="font-mono font-semibold text-slate-200">{{ newGuarantee }}W</span>
              <span class="ml-1 text-slate-500">
                ({{ outcome === 'EVACUATED' ? '−' : '+' }}{{ guaranteeDelta }}W)
              </span>
            </div>
          </section>

          <!-- C. 玩家明细表 -->
          <section>
            <div class="text-xs uppercase tracking-wider text-slate-400 mb-2">三位玩家结算明细</div>
            <div class="rounded-xl border border-white/10 overflow-hidden">
              <table class="w-full text-sm">
                <thead class="bg-slate-900/60 text-slate-400">
                  <tr>
                    <th class="text-left font-medium px-3 py-2">玩家</th>
                    <th
                      v-for="h in STAT_HEADERS"
                      :key="h.key"
                      class="text-center font-medium px-2 py-2 w-20"
                    >
                      {{ h.label }}
                    </th>
                    <th class="text-right font-medium px-3 py-2 w-20">本次币</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="p in players"
                    :key="p.id"
                    class="border-t border-white/5"
                  >
                    <td class="px-3 py-2">
                      <div class="flex items-center gap-2 min-w-0">
                        <Avatar :seed="p.id" :name="p.nickname" size="sm" />
                        <div class="min-w-0">
                          <div class="font-semibold text-slate-100 truncate flex items-center gap-1">
                            <span v-if="p.isHost" class="text-amber-400">👑</span>
                            {{ p.nickname }}
                          </div>
                          <span
                            class="badge text-[10px]"
                            :class="
                              p.role === 'BOSS'
                                ? 'bg-orange-500/20 text-orange-300'
                                : 'bg-blue-500/20 text-blue-300'
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
                        class="w-full rounded-lg border border-white/10 bg-slate-900/70 px-2 py-1.5 text-sm text-center text-slate-100 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400/40"
                      />
                    </td>
                    <td class="px-3 py-2 text-right font-mono">
                      <template v-if="previewCoinDelta(p.id) === null">
                        <span class="text-rose-400 text-xs">非法</span>
                      </template>
                      <template v-else>
                        <span class="text-emerald-400">+{{ previewCoinDelta(p.id) }}</span>
                      </template>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <!-- D. 奖励规则提示 -->
          <section class="rounded-xl bg-slate-900/50 border border-white/10 px-4 py-3 text-xs text-slate-400 leading-relaxed">
            <div class="text-slate-300 font-semibold mb-1">奖励规则</div>
            <div>击杀 1 人 +1 币 · 摸 1 个小红 +1 币 · 摸 1 个 6 格 +2 币 · 摸 1 个 9 格及以上 +3 币</div>
            <div class="text-emerald-300/80 mt-0.5">撤离成功额外每人 +3 币</div>
          </section>
        </div>

        <div class="px-5 py-4 border-t border-white/10 flex items-center justify-between gap-3">
          <div class="text-xs text-rose-400 min-h-[1em]">
            {{ validation.ok ? '' : validation.reason }}
          </div>
          <div class="flex items-center gap-2 shrink-0">
            <button class="btn-ghost py-2 px-4 text-sm" @click="emit('close')">取消</button>
            <button
              class="btn-primary py-2 px-4 text-sm"
              :disabled="!canSubmit"
              @click="submit"
            >
              {{ submitting ? '结算中…' : '确认结算' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.18s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
