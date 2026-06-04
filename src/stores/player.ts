/**
 * 玩家本地身份 Store：
 *   - playerId：稳定 ID，首次访问生成并持久化。
 *   - nickname：玩家昵称，可由用户修改。
 *   - lastRoomCode：上一次进入的房间号（用于刷新页面后引导）。
 *   - session：与某个房间绑定的 sessionToken（用于断线重连）。
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import { generatePlayerId } from '@/utils/id';
import { storage } from '@/utils/storage';

const STORAGE_KEY_ID = 'playerId';
const STORAGE_KEY_NICKNAME = 'nickname';
const STORAGE_KEY_LAST_CODE = 'lastRoomCode';
const STORAGE_KEY_SESSION = 'session';

type SessionRecord = {
  code: string;
  token: string;
  at: number;
};

const DEFAULT_NICKNAME = '花栗鼠的板板';

export const usePlayerStore = defineStore('player', () => {
  const playerId = ref<string>('');
  const nickname = ref<string>('');
  const lastRoomCode = ref<string>('');
  const session = ref<SessionRecord | null>(null);

  function ensureIdentity(): void {
    let id = storage.get<string>(STORAGE_KEY_ID, '');
    if (!id) {
      id = generatePlayerId();
      storage.set(STORAGE_KEY_ID, id);
    }
    playerId.value = id;

    let name = storage.get<string>(STORAGE_KEY_NICKNAME, '');
    if (!name) {
      name = DEFAULT_NICKNAME;
      storage.set(STORAGE_KEY_NICKNAME, name);
    }
    nickname.value = name;

    lastRoomCode.value = storage.get<string>(STORAGE_KEY_LAST_CODE, '');
    session.value = storage.get<SessionRecord | null>(STORAGE_KEY_SESSION, null);
  }

  function setNickname(name: string): void {
    const trimmed = name.trim().slice(0, 16);
    if (!trimmed) return;
    nickname.value = trimmed;
    storage.set(STORAGE_KEY_NICKNAME, trimmed);
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
    lastRoomCode,
    session,
    ensureIdentity,
    setNickname,
    rememberRoomCode,
    forgetRoomCode,
    setSessionToken,
    getSessionToken,
    clearSession,
    setPlayerId,
  };
});
