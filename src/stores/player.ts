/**
 * 玩家本地身份 Store：
 *   - playerId：稳定 ID，首次访问生成并持久化。
 *   - nickname：最终展示昵称，由“预设名牌 + 自定义昵称”拼接而成。
 *   - lastRoomCode：上一次进入的房间号（用于刷新页面后引导）。
 *   - session：与某个房间绑定的 sessionToken（用于断线重连）。
 */

import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { generatePlayerId } from '@/utils/id';
import { storage } from '@/utils/storage';

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

export function maxCustomNicknameLength(preset: string): number {
  return Math.max(0, MAX_NICKNAME_LENGTH - preset.trim().slice(0, MAX_NICKNAME_LENGTH).length);
}

export function composeNicknameParts(preset: string, custom: string): string {
  const normalizedPreset = preset.trim().slice(0, MAX_NICKNAME_LENGTH);
  const normalizedCustom = custom.trim().slice(0, maxCustomNicknameLength(normalizedPreset));
  if (!normalizedPreset || !normalizedCustom) return '';
  return `${normalizedPreset}${normalizedCustom}`;
}

export const usePlayerStore = defineStore('player', () => {
  const playerId = ref<string>('');
  const nicknamePreset = ref<string>('');
  const nicknameCustom = ref<string>('');
  const legacyNickname = ref<string>('');
  const lastRoomCode = ref<string>('');
  const session = ref<SessionRecord | null>(null);
  const nickname = computed<string>(() => {
    const combined = composeNicknameParts(nicknamePreset.value, nicknameCustom.value);
    return combined || legacyNickname.value;
  });
  const hasStructuredNickname = computed<boolean>(
    () => !!composeNicknameParts(nicknamePreset.value, nicknameCustom.value),
  );

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

    const combined = composeNicknameParts(nicknamePreset.value, nicknameCustom.value);
    if (combined) {
      legacyNickname.value = combined;
      storage.set(STORAGE_KEY_NICKNAME, combined);
      storage.set(STORAGE_KEY_NICKNAME_PRESET, nicknamePreset.value);
      storage.set(STORAGE_KEY_NICKNAME_CUSTOM, nicknameCustom.value);
    } else {
      nicknamePreset.value = '';
      nicknameCustom.value = '';
    }

    lastRoomCode.value = storage.get<string>(STORAGE_KEY_LAST_CODE, '');
    session.value = storage.get<SessionRecord | null>(STORAGE_KEY_SESSION, null);
  }

  function setNickname(name: string): void {
    const trimmed = name.trim().slice(0, MAX_NICKNAME_LENGTH);
    if (!trimmed) return;
    nicknamePreset.value = '';
    nicknameCustom.value = '';
    legacyNickname.value = trimmed;
    storage.set(STORAGE_KEY_NICKNAME, trimmed);
    storage.remove(STORAGE_KEY_NICKNAME_PRESET);
    storage.remove(STORAGE_KEY_NICKNAME_CUSTOM);
  }

  function setNicknameParts(preset: string, custom: string): void {
    const normalizedPreset = preset.trim().slice(0, MAX_NICKNAME_LENGTH);
    const normalizedCustom = custom.trim().slice(0, maxCustomNicknameLength(normalizedPreset));
    const combined = composeNicknameParts(normalizedPreset, normalizedCustom);
    if (!combined) return;

    nicknamePreset.value = normalizedPreset;
    nicknameCustom.value = normalizedCustom;
    legacyNickname.value = combined;
    storage.set(STORAGE_KEY_NICKNAME_PRESET, normalizedPreset);
    storage.set(STORAGE_KEY_NICKNAME_CUSTOM, normalizedCustom);
    storage.set(STORAGE_KEY_NICKNAME, combined);
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
    hasStructuredNickname,
    lastRoomCode,
    session,
    ensureIdentity,
    setNickname,
    setNicknameParts,
    rememberRoomCode,
    forgetRoomCode,
    setSessionToken,
    getSessionToken,
    clearSession,
    setPlayerId,
  };
});
