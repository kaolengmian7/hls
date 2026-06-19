import { customAlphabet, nanoid } from 'nanoid';

const numericAlphabet = '0123456789';
const numericNano = customAlphabet(numericAlphabet, 6);

/** 生成 6 位随机数字房间 Code（首位不为 0）。 */
export function generateRoomCode(): string {
  let code = numericNano();
  if (code[0] === '0') {
    code = String(1 + Math.floor(Math.random() * 9)) + code.slice(1);
  }
  return code;
}

/** 生成持久化 playerId。 */
export function generatePlayerId(): string {
  return `p_${nanoid(12)}`;
}

/** 生成短随机 ID（用于 reqId / lottery record id 等）。 */
export function shortId(): string {
  return nanoid(10);
}

/** 校验是否为合法 6 位数字 Code。 */
export function isValidRoomCode(code: string): boolean {
  return /^\d{6}$/.test(code);
}
