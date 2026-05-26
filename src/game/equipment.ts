/**
 * 装备 / 武器数据，与后端 internal/game/equipment.go 保持一一对齐。
 *
 * 设计要点：
 *   - 护甲（ARMOR）：单件物品，购买时不需要 pick。
 *   - 武器组合（BUNDLE）：自带固定武器（bundledIds），并需要从 pickPool 二级选 1 把枪。
 *   - 散件武器（WEAPON）：单买 1 把，从 pickPool 中二级选 1。
 *
 * 真正的合法性裁决在服务端，前端表只用于渲染与本地预校验。
 */

export type EquipmentCategory = 'ARMOR' | 'BUNDLE' | 'WEAPON';

export type Equipment = {
  id: string;
  name: string;
  icon: string;
  price: number;
  category: EquipmentCategory;
  description: string;
  bundledIds?: string[];
  pickPool?: string[];
};

export type Weapon = {
  id: string;
  name: string;
  icon: string;
  category: 'SNIPER' | 'RIFLE' | 'SMG' | 'SHOTGUN';
};

export const WEAPON_LIST: Weapon[] = [
  { id: 'AWM', name: 'AWM', icon: '🎯', category: 'SNIPER' },
  { id: 'M24', name: 'M24', icon: '🎯', category: 'SNIPER' },
  { id: 'KAR98K', name: 'Kar98K', icon: '🎯', category: 'SNIPER' },
  { id: 'M762', name: 'M762', icon: '🔫', category: 'RIFLE' },
  { id: 'M416', name: 'M416', icon: '🔫', category: 'RIFLE' },
  { id: 'SCAR_L', name: 'SCAR-L', icon: '🔫', category: 'RIFLE' },
  { id: 'AKM', name: 'AKM', icon: '🔫', category: 'RIFLE' },
  { id: 'UZI', name: 'UZI', icon: '💥', category: 'SMG' },
  { id: 'VECTOR', name: 'Vector', icon: '💥', category: 'SMG' },
  { id: 'S686', name: 'S686', icon: '🩸', category: 'SHOTGUN' },
  { id: 'S12K', name: 'S12K', icon: '🩸', category: 'SHOTGUN' },
];

const NON_AW_GUNS = [
  'M24',
  'KAR98K',
  'M762',
  'M416',
  'SCAR_L',
  'AKM',
  'UZI',
  'VECTOR',
  'S686',
  'S12K',
];

export const EQUIPMENT_LIST: Equipment[] = [
  {
    id: 'ARMOR_L4',
    name: '4 级护甲',
    icon: '🛡️',
    price: 1,
    category: 'ARMOR',
    description: '基础护甲，少量减伤。',
  },
  {
    id: 'ARMOR_L5',
    name: '5 级护甲',
    icon: '🛡️',
    price: 2,
    category: 'ARMOR',
    description: '进阶护甲，中等减伤。',
  },
  {
    id: 'ARMOR_L6',
    name: '6 级护甲',
    icon: '🛡️',
    price: 3,
    category: 'ARMOR',
    description: '顶级护甲，最大减伤。',
  },
  {
    id: 'BUNDLE_AW',
    name: 'AW + 任意枪',
    icon: '🎯',
    price: 3,
    category: 'BUNDLE',
    description: '必带 AWM，外加自选 1 把其他枪。',
    bundledIds: ['AWM'],
    pickPool: NON_AW_GUNS,
  },
  {
    id: 'BUNDLE_M7',
    name: 'M7 + 任意狙击枪',
    icon: '🔫',
    price: 2,
    category: 'BUNDLE',
    description: '必带 M762，外加自选 1 把非 AW 狙击枪。',
    bundledIds: ['M762'],
    pickPool: ['M24', 'KAR98K'],
  },
  {
    id: 'WEAPON_SINGLE',
    name: '其他枪（自选 1 把）',
    icon: '🔫',
    price: 1,
    category: 'WEAPON',
    description: '除 AW 以外任选 1 把单枪。',
    pickPool: NON_AW_GUNS,
  },
];

export function getEquipment(id: string): Equipment | undefined {
  return EQUIPMENT_LIST.find((e) => e.id === id);
}

export function getWeapon(id: string): Weapon | undefined {
  return WEAPON_LIST.find((w) => w.id === id);
}

/** 是否需要弹出二级选枪面板。 */
export function needsPick(eq: Equipment): boolean {
  return (eq.pickPool?.length ?? 0) > 0;
}

/** 按装备的 pickPool 过滤出真实武器对象，保留 WEAPON_LIST 中的顺序。 */
export function pickableWeapons(eq: Equipment): Weapon[] {
  const pool = eq.pickPool;
  if (!pool || pool.length === 0) return [];
  const set = new Set(pool);
  return WEAPON_LIST.filter((w) => set.has(w.id));
}
