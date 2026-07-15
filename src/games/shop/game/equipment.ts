/**
 * 装备 / 武器数据，与后端 internal/game/equipment.go 保持一一对齐。
 *
 * 设计要点：
 *   - 护甲（ARMOR）：单件物品，直接购买。
 *   - 武器组合（BUNDLE）：可附带一组展示文案（bundledLabels），整组直接购买。
 *   - 散件武器（WEAPON）：单件武器，直接购买。
 *
 * 所有条目均为直接扣费购买，无需二级选枪。真正的合法性裁决在服务端，前端表只用于渲染与本地预校验。
 */

export type EquipmentCategory = 'ARMOR' | 'BUNDLE' | 'WEAPON';

export type Equipment = {
  id: string;
  name: string;
  icon: string;
  price: number;
  category: EquipmentCategory;
  description: string;
  bundledLabels?: string[];
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

export const EQUIPMENT_LIST: Equipment[] = [
  {
    id: 'ARMOR_L5',
    name: '5 级护甲',
    icon: '🛡️',
    price: 0,
    category: 'ARMOR',
    description: '5级重突',
  },
  {
    id: 'ARMOR_L6',
    name: '6 级护甲',
    icon: '🛡️',
    price: 0,
    category: 'ARMOR',
    description: '刘涛的力量',
  },
  {
    id: 'BUNDLE_AW',
    name: 'AW套装',
    icon: '🎯',
    price: 0,
    category: 'BUNDLE',
    description: 'AWM + 任意一把枪',
    bundledLabels: ['AWM'],
  },
];

export function getEquipment(id: string): Equipment | undefined {
  return EQUIPMENT_LIST.find((e) => e.id === id);
}

export function getWeapon(id: string): Weapon | undefined {
  return WEAPON_LIST.find((w) => w.id === id);
}
