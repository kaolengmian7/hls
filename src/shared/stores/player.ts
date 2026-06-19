/**
 * 玩家本地身份 Store：
 *   - playerId：稳定 ID，首次访问生成并持久化。
 *   - nickname：最终展示昵称，由“自定义昵称 + 可选名牌前缀”拼接而成（名牌选填）。
 *   - lastRoomCode：上一次进入的房间号（用于刷新页面后引导）。
 *   - session：与某个房间绑定的 sessionToken（用于断线重连）。
 */

import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { generatePlayerId } from '@shared/utils/id';
import { storage } from '@shared/utils/storage';

const STORAGE_KEY_ID = 'playerId';
const STORAGE_KEY_NICKNAME = 'nickname';
const STORAGE_KEY_NICKNAME_PRESET = 'nicknamePreset';
const STORAGE_KEY_NICKNAME_CUSTOM = 'nicknameCustom';
const STORAGE_KEY_LAST_CODE = 'lastRoomCode';
const STORAGE_KEY_SESSION = 'session';

type SessionRecord = {
  code: string;
  token: string;
  at: number;
};

const DEFAULT_NICKNAME = '花栗鼠的板板';
export const MAX_NICKNAME_LENGTH = 16;
/** 名牌与自定义昵称之间的分隔符。 */
export const NICKNAME_SEPARATOR = '-';

/**
 * 可选名牌（昵称前缀）预设列表。
 *
 * 名牌为「选填」：首页只填昵称即可开始；准备页可挑选名牌拼到昵称前面，也可不选。
 */
export const NICKNAME_PRESETS = [
  '花栗鼠的可爱板板',
  '花栗鼠的尊贵板板',
  '花栗鼠的月光小护卫',
  '花栗鼠的星尘护航使',
  '花栗鼠的秘境萌骑士',
  '花栗鼠的糖霜守望者',
  '花栗鼠的宝匣小勇者',
  '花栗鼠的幻梦护航团',
];

export function maxCustomNicknameLength(preset: string): number {
  const presetLength = preset.trim().slice(0, MAX_NICKNAME_LENGTH).length;
  // 预留分隔符占位，确保「名牌 - 昵称」整体不超过 MAX_NICKNAME_LENGTH。
  return Math.max(0, MAX_NICKNAME_LENGTH - presetLength - NICKNAME_SEPARATOR.length);
}

/**
 * 由「名牌（可选）+ 自定义昵称」拼接出最终昵称。
 *
 * - 自定义昵称为空：返回空串（视为未设置昵称）。
 * - 未选名牌：直接返回自定义昵称（截断到 MAX_NICKNAME_LENGTH）。
 * - 选了名牌：返回「名牌-自定义」，自定义部分按剩余额度截断。
 */
export function composeNickname(preset: string, custom: string): string {
  const normalizedCustom = custom.trim();
  if (!normalizedCustom) return '';
  const normalizedPreset = preset.trim().slice(0, MAX_NICKNAME_LENGTH);
  if (!normalizedPreset) return normalizedCustom.slice(0, MAX_NICKNAME_LENGTH);
  const trimmedCustom = normalizedCustom.slice(0, maxCustomNicknameLength(normalizedPreset));
  if (!trimmedCustom) return normalizedPreset;
  return `${normalizedPreset}${NICKNAME_SEPARATOR}${trimmedCustom}`;
}

export const usePlayerStore = defineStore('player', () => {
  const playerId = ref<string>('');
  const nicknamePreset = ref<string>('');
  const nicknameCustom = ref<string>('');
  const legacyNickname = ref<string>('');
  const lastRoomCode = ref<string>('');
  const session = ref<SessionRecord | null>(null);
  const nickname = computed<string>(() => {
    const combined = composeNickname(nicknamePreset.value, nicknameCustom.value);
    return combined || legacyNickname.value;
  });
  const hasCustomNickname = computed<boolean>(() => !!nicknameCustom.value.trim());

  function ensureIdentity(): void {
    let id = storage.get<string>(STORAGE_KEY_ID, '');
    if (!id) {
      id = generatePlayerId();
      storage.set(STORAGE_KEY_ID, id);
    }
    playerId.value = id;

    let name = storage.get<string>(STORAGE_KEY_NICKNAME, '').trim().slice(0, MAX_NICKNAME_LENGTH);
    if (!name) {
      name = DEFAULT_NICKNAME;
      storage.set(STORAGE_KEY_NICKNAME, name);
    }

    nicknamePreset.value = storage.get<string>(STORAGE_KEY_NICKNAME_PRESET, '').trim();
    nicknameCustom.value = storage.get<string>(STORAGE_KEY_NICKNAME_CUSTOM, '').trim();
    legacyNickname.value = name;

    // 迁移：旧版本仅存了整体昵称（STORAGE_KEY_NICKNAME），没有结构化的自定义昵称时，
    // 把整体昵称（非默认占位）回填为自定义昵称，作为首页输入框的初始值。
    if (!nicknameCustom.value && name && name !== DEFAULT_NICKNAME) {
      nicknameCustom.value = name.slice(0, MAX_NICKNAME_LENGTH);
    }

    persistNickname();

    lastRoomCode.value = storage.get<string>(STORAGE_KEY_LAST_CODE, '');
    session.value = storage.get<SessionRecord | null>(STORAGE_KEY_SESSION, null);
  }

  /** 把当前「名牌 + 自定义昵称」拼接结果回写到本地存储，作为最终昵称。 */
  function persistNickname(): void {
    storage.set(STORAGE_KEY_NICKNAME_PRESET, nicknamePreset.value);
    storage.set(STORAGE_KEY_NICKNAME_CUSTOM, nicknameCustom.value);
    const combined = composeNickname(nicknamePreset.value, nicknameCustom.value);
    if (combined) {
      legacyNickname.value = combined;
      storage.set(STORAGE_KEY_NICKNAME, combined);
    }
  }

  /** 设置自定义昵称（首页输入 / 准备页编辑），保留已选名牌。 */
  function setNicknameCustom(custom: string): void {
    nicknameCustom.value = custom.trim().slice(0, MAX_NICKNAME_LENGTH);
    persistNickname();
  }

  /** 设置或清除名牌（准备页选填）；传空串表示不使用名牌。 */
  function setNicknamePreset(preset: string): void {
    nicknamePreset.value = preset.trim().slice(0, MAX_NICKNAME_LENGTH);
    persistNickname();
  }

  function rememberRoomCode(code: string): void {
    lastRoomCode.value = code;
    storage.set(STORAGE_KEY_LAST_CODE, code);
  }

  function forgetRoomCode(): void {
    lastRoomCode.value = '';
    storage.remove(STORAGE_KEY_LAST_CODE);
  }

  function setSessionToken(code: string, token: string): void {
    if (!token) return;
    session.value = { code, token, at: Date.now() };
    storage.set(STORAGE_KEY_SESSION, session.value);
  }

  function getSessionToken(code: string): string {
    if (session.value && session.value.code === code) return session.value.token;
    return '';
  }

  function clearSession(): void {
    session.value = null;
    storage.remove(STORAGE_KEY_SESSION);
  }

  /** 当 playerId 由服务端在 createRoom 时签发后，覆写本地 ID。 */
  function setPlayerId(id: string): void {
    if (!id) return;
    playerId.value = id;
    storage.set(STORAGE_KEY_ID, id);
  }

  return {
    playerId,
    nickname,
    nicknamePreset,
    nicknameCustom,
    hasCustomNickname,
    lastRoomCode,
    session,
    ensureIdentity,
    setNicknameCustom,
    setNicknamePreset,
    rememberRoomCode,
    forgetRoomCode,
    setSessionToken,
    getSessionToken,
    clearSession,
    setPlayerId,
  };
});
