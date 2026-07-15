package game

import (
	"hls.dev/server/pkg/errors"
)

// EquipmentCategory 标识装备大类，供前端做分组渲染与显示用。
type EquipmentCategory string

const (
	// CategoryArmor 护甲（单件）。
	CategoryArmor EquipmentCategory = "ARMOR"
	// CategoryBundle 武器组合（包含一把固定枪 + 任意搭配，整组直接带走）。
	CategoryBundle EquipmentCategory = "BUNDLE"
	// CategoryWeapon 散件武器。
	CategoryWeapon EquipmentCategory = "WEAPON"
)

// Equipment 装备元数据。
//
// 字段说明：
//   - BundledLabels 仅 BUNDLE 类使用，表示组合随卡片展示的一组文案。
//   - PickPool   预留字段：非空时表示购买需要二级 pick。当前所有装备均直接购买，
//     该字段保持为空（即无需二级选枪）。
type Equipment struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Icon          string            `json:"icon"`
	Price         int               `json:"price"`
	Category      EquipmentCategory `json:"category"`
	Description   string            `json:"description"`
	BundledLabels []string          `json:"bundledLabels,omitempty"`
	PickPool      []string          `json:"pickPool,omitempty"`
}

// Weapon 武器元数据（用于二级选枪面板）。
type Weapon struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	Category string `json:"category"` // SNIPER | RIFLE | SMG | SHOTGUN
}

// 武器 ID 常量。
const (
	WeaponAWM    = "AWM"
	WeaponM24    = "M24"
	WeaponKar98K = "KAR98K"
	WeaponM762   = "M762"
	WeaponM416   = "M416"
	WeaponSCARL  = "SCAR_L"
	WeaponAKM    = "AKM"
	WeaponUZI    = "UZI"
	WeaponVector = "VECTOR"
	WeaponS686   = "S686"
	WeaponS12K   = "S12K"
	WeaponM7     = "M7"
	WeaponM14    = "M14"
	WeaponAsval  = "Asval"
	WeaponMK47   = "MK47"
)

// weaponCatalog 是当前局可见的所有武器。
//
// 设计约定：AW(AWM) 只能通过 BUNDLE_AW 获得。
var weaponCatalog = []Weapon{
	{ID: WeaponAWM, Name: "AWM", Icon: "🎯", Category: "SNIPER"},
	{ID: WeaponM24, Name: "M24", Icon: "🎯", Category: "SNIPER"},
	{ID: WeaponKar98K, Name: "Kar98K", Icon: "🎯", Category: "SNIPER"},
	{ID: WeaponM762, Name: "M762", Icon: "🔫", Category: "RIFLE"},
	{ID: WeaponM416, Name: "M416", Icon: "🔫", Category: "RIFLE"},
	{ID: WeaponSCARL, Name: "SCAR-L", Icon: "🔫", Category: "RIFLE"},
	{ID: WeaponAKM, Name: "AKM", Icon: "🔫", Category: "RIFLE"},
	{ID: WeaponUZI, Name: "UZI", Icon: "💥", Category: "SMG"},
	{ID: WeaponVector, Name: "Vector", Icon: "💥", Category: "SMG"},
	{ID: WeaponS686, Name: "S686", Icon: "🩸", Category: "SHOTGUN"},
	{ID: WeaponS12K, Name: "S12K", Icon: "🩸", Category: "SHOTGUN"},
}

// 装备 ID 常量。
const (
	EquipmentArmorL5  = "ARMOR_L5"
	EquipmentArmorL6  = "ARMOR_L6"
	EquipmentBundleAW = "BUNDLE_AW"
)

// equipmentCatalog 是默认装备目录。
//
// 所有条目均为直接购买（扣费即可），无需二级选枪。
var equipmentCatalog = []Equipment{
	{
		ID: EquipmentArmorL5, Name: "5 级护甲", Icon: "🛡️", Price: 0, Category: CategoryArmor,
		Description: "进阶护甲，中等减伤。",
	},
	{
		ID: EquipmentArmorL6, Name: "6 级护甲", Icon: "🛡️", Price: 0, Category: CategoryArmor,
		Description: "顶级护甲，最大减伤。",
	},
	{
		ID: EquipmentBundleAW, Name: "AW套装", Icon: "🎯", Price: 0, Category: CategoryBundle,
		Description:   "AWM + 任意一把枪",
		BundledLabels: []string{WeaponAWM},
	},
}

// EquipmentList 返回完整装备目录（拷贝避免外部修改）。
func EquipmentList() []Equipment {
	out := make([]Equipment, len(equipmentCatalog))
	for i, eq := range equipmentCatalog {
		out[i] = cloneEquipment(eq)
	}
	return out
}

// WeaponList 返回武器目录（拷贝避免外部修改）。
func WeaponList() []Weapon {
	out := make([]Weapon, len(weaponCatalog))
	copy(out, weaponCatalog)
	return out
}

// FindEquipment 按 ID 查找装备；不存在返回 nil。
func FindEquipment(id string) *Equipment {
	for i := range equipmentCatalog {
		if equipmentCatalog[i].ID == id {
			eq := cloneEquipment(equipmentCatalog[i])
			return &eq
		}
	}
	return nil
}

// FindWeapon 按 ID 查找武器；不存在返回 nil。
func FindWeapon(id string) *Weapon {
	for i := range weaponCatalog {
		if weaponCatalog[i].ID == id {
			w := weaponCatalog[i]
			return &w
		}
	}
	return nil
}

// ValidatePurchase 校验一次购买是否合法。
//
// 返回值：
//   - 命中且通过校验：返回装备指针，error 为 nil。
//   - 装备不存在：返回 CodeItemNotFound。
//   - 当前目录下所有装备都直接购买；若 caller 多传了 pick，返回 CodeInvalidParam("PICK_NOT_REQUIRED")。
//   - 若未来恢复二级选枪，PickPool 非空但 pick 为空或不在池中时，返回 CodeInvalidParam("INVALID_PICK")。
func ValidatePurchase(equipmentID, pick string) (*Equipment, error) {
	eq := FindEquipment(equipmentID)
	if eq == nil {
		return nil, errors.NewExposable(errors.CodeItemNotFound, "EQUIPMENT_NOT_FOUND")
	}
	if len(eq.PickPool) == 0 {
		if pick != "" {
			return nil, errors.NewExposable(errors.CodeInvalidParam, "PICK_NOT_REQUIRED")
		}
		return eq, nil
	}
	if pick == "" {
		return nil, errors.NewExposable(errors.CodeInvalidParam, "INVALID_PICK")
	}
	for _, allowed := range eq.PickPool {
		if allowed == pick {
			return eq, nil
		}
	}
	return nil, errors.NewExposable(errors.CodeInvalidParam, "INVALID_PICK")
}

// cloneEquipment 拷贝 Equipment 的可变切片字段。
func cloneEquipment(src Equipment) Equipment {
	dst := src
	if src.BundledLabels != nil {
		dst.BundledLabels = append([]string(nil), src.BundledLabels...)
	}
	if src.PickPool != nil {
		dst.PickPool = append([]string(nil), src.PickPool...)
	}
	return dst
}
