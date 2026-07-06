import { defineStore } from 'pinia';
import { ref } from 'vue';
import { storage } from '@shared/utils/storage';

const STORAGE_KEY_LAST_CODE = 'lastRoomCode';

export const usePlayerStore = defineStore('player', () => {
  const lastRoomCode = ref<string>('');

  function ensureIdentity(): void {
    lastRoomCode.value = storage.get<string>(STORAGE_KEY_LAST_CODE, '');
  }

  function rememberRoomCode(code: string): void {
    lastRoomCode.value = code;
    storage.set(STORAGE_KEY_LAST_CODE, code);
  }

  function forgetRoomCode(): void {
    lastRoomCode.value = '';
    storage.remove(STORAGE_KEY_LAST_CODE);
  }

  return {
    lastRoomCode,
    ensureIdentity,
    rememberRoomCode,
    forgetRoomCode,
  };
});
