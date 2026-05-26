package game

import (
	"hls.dev/server/pkg/errors"
)

// EquipmentCategory 标识装备大类，供前端做分组渲染与显示用。
type EquipmentCategory string

const (
	// CategoryArmor 护甲（单件，不需要二级选枪）。
	CategoryArmor EquipmentCategory = "ARMOR"
	// CategoryBundle 武器组合（包含一把固定枪，且需要二级 pick）。
	CategoryBundle EquipmentCategory = "BUNDLE"
	// CategoryWeapon 散件武器（自选一把）。
	CategoryWeapon EquipmentCategory = "WEAPON"
)

// Equipment 装备元数据。
//
// 字段说明：
//   - BundledIDs 仅 BUNDLE 类使用，表示组合里"必带"的武器 ID。
//   - PickPool   非空时表示购买时需要二级 pick，且 pick 必须出现在该池中。
//     ARMOR 类没有 PickPool；BUNDLE/WEAPON 类必有 PickPool。
type Equipment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Icon        string            `json:"icon"`
	Price       int               `json:"price"`
	Category    EquipmentCategory `json:"category"`
	Description string            `json:"description"`
	BundledIDs  []string          `json:"bundledIds,omitempty"`
	PickPool    []string          `json:"pickPool,omitempty"`
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
)

// weaponCatalog 是当前局可见的所有武器。
//
// 设计约定：AW(AWM) 只能通过 BUNDLE_AW 获得；其他枪可以通过 WEAPON_SINGLE 单买。
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
	EquipmentArmorL4     = "ARMOR_L4"
	EquipmentArmorL5     = "ARMOR_L5"
	EquipmentArmorL6     = "ARMOR_L6"
	EquipmentBundleAW    = "BUNDLE_AW"
	EquipmentBundleM7    = "BUNDLE_M7"
	EquipmentWeaponOther = "WEAPON_SINGLE"
)

// equipmentCatalog 是默认装备目录。
//
// AW 不进入「其他枪」单买池；M7 与其他狙击枪可单买。
var equipmentCatalog = []Equipment{
	{
		ID: EquipmentArmorL4, Name: "4 级护甲", Icon: "🛡️", Price: 1, Category: CategoryArmor,
		Description: "基础护甲，少量减伤。",
	},
	{
		ID: EquipmentArmorL5, Name: "5 级护甲", Icon: "🛡️", Price: 2, Category: CategoryArmor,
		Description: "进阶护甲，中等减伤。",
	},
	{
		ID: EquipmentArmorL6, Name: "6 级护甲", Icon: "🛡️", Price: 3, Category: CategoryArmor,
		Description: "顶级护甲，最大减伤。",
	},
	{
		ID: EquipmentBundleAW, Name: "AW + 任意枪", Icon: "🎯", Price: 3, Category: CategoryBundle,
		Description: "必带 AWM，外加自选 1 把其他枪。",
		BundledIDs:  []string{WeaponAWM},
		PickPool: []string{
			WeaponM24, WeaponKar98K,
			WeaponM762, WeaponM416, WeaponSCARL, WeaponAKM,
			WeaponUZI, WeaponVector, WeaponS686, WeaponS12K,
		},
	},
	{
		ID: EquipmentBundleM7, Name: "M7 + 任意狙击枪", Icon: "🔫", Price: 2, Category: CategoryBundle,
		Description: "必带 M762，外加自选 1 把非 AW 狙击枪。",
		BundledIDs:  []string{WeaponM762},
		PickPool:    []string{WeaponM24, WeaponKar98K},
	},
	{
		ID: EquipmentWeaponOther, Name: "其他枪（自选 1 把）", Icon: "🔫", Price: 1, Category: CategoryWeapon,
		Description: "除 AW 以外任选 1 把单枪。",
		PickPool: []string{
			WeaponM24, WeaponKar98K,
			WeaponM762, WeaponM416, WeaponSCARL, WeaponAKM,
			WeaponUZI, WeaponVector, WeaponS686, WeaponS12K,
		},
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
//   - PickPool 为空（如护甲）但 caller 多传了 pick：返回 CodeInvalidParam("PICK_NOT_REQUIRED")。
//   - PickPool 非空但 pick 为空或不在池中：返回 CodeInvalidParam("INVALID_PICK")。
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
	if src.BundledIDs != nil {
		dst.BundledIDs = append([]string(nil), src.BundledIDs...)
	}
	if src.PickPool != nil {
		dst.PickPool = append([]string(nil), src.PickPool...)
	}
	return dst
}
