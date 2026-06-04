package game

import (
	"testing"

	"hls.dev/server/pkg/errors"
)

func TestEquipmentList(t *testing.T) {
	list := EquipmentList()
	if len(list) == 0 {
		t.Fatal("equipment list is empty")
	}
	wantPrice := map[string]int{
		EquipmentArmorL4:     1,
		EquipmentArmorL5:     2,
		EquipmentArmorL6:     3,
		EquipmentBundleAW:    4,
		EquipmentBundleM7:    3,
		EquipmentWeaponOther: 2,
		EquipmentWeaponLone:  1,
	}
	got := map[string]int{}
	for _, eq := range list {
		got[eq.ID] = eq.Price
	}
	for id, price := range wantPrice {
		if got[id] != price {
			t.Errorf("equipment %s price=%d, want %d", id, got[id], price)
		}
	}
}

func TestEquipmentListIsDefensiveCopy(t *testing.T) {
	list := EquipmentList()
	for i := range list {
		if list[i].ID == EquipmentBundleAW && len(list[i].BundledLabels) > 0 {
			list[i].BundledLabels[0] = "MUTATED"
			break
		}
	}
	// 重新读取应当不受外部修改影响。
	again := FindEquipment(EquipmentBundleAW)
	if again == nil {
		t.Fatal("BUNDLE_AW not found")
	}
	for _, p := range again.BundledLabels {
		if p == "MUTATED" {
			t.Fatal("equipment list mutation leaked into catalog")
		}
	}
}

func TestValidatePurchase(t *testing.T) {
	tests := []struct {
		name     string
		eqID     string
		pick     string
		wantOK   bool
		wantCode int    // 失败时期望的错误码（成功时忽略）
		wantMsg  string // 失败时期望的错误 Message（成功时忽略）
	}{
		{
			name: "护甲直接购买",
			eqID: EquipmentArmorL5, pick: "",
			wantOK: true,
		},
		{
			name: "护甲传 pick 应拒绝",
			eqID: EquipmentArmorL5, pick: WeaponM416,
			wantCode: errors.CodeInvalidParam, wantMsg: "PICK_NOT_REQUIRED",
		},
		{
			name: "AW 组合直接购买",
			eqID: EquipmentBundleAW, pick: "",
			wantOK: true,
		},
		{
			name: "AW 组合不再接受 pick",
			eqID: EquipmentBundleAW, pick: WeaponM762,
			wantCode: errors.CodeInvalidParam, wantMsg: "PICK_NOT_REQUIRED",
		},
		{
			name: "M7 组合直接购买",
			eqID: EquipmentBundleM7, pick: "",
			wantOK: true,
		},
		{
			name: "M7 组合不再接受 pick",
			eqID: EquipmentBundleM7, pick: WeaponM24,
			wantCode: errors.CodeInvalidParam, wantMsg: "PICK_NOT_REQUIRED",
		},
		{
			name: "其他枪直接购买",
			eqID: EquipmentWeaponOther, pick: "",
			wantOK: true,
		},
		{
			name: "其他枪不再接受 pick",
			eqID: EquipmentWeaponOther, pick: WeaponKar98K,
			wantCode: errors.CodeInvalidParam, wantMsg: "PICK_NOT_REQUIRED",
		},
		{
			name: "单枪匹马直接购买",
			eqID: EquipmentWeaponLone, pick: "",
			wantOK: true,
		},
		{
			name: "单枪匹马不接受 pick",
			eqID: EquipmentWeaponLone, pick: WeaponM416,
			wantCode: errors.CodeInvalidParam, wantMsg: "PICK_NOT_REQUIRED",
		},
		{
			name: "装备不存在",
			eqID: "UNKNOWN", pick: "",
			wantCode: errors.CodeItemNotFound, wantMsg: "EQUIPMENT_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq, err := ValidatePurchase(tt.eqID, tt.pick)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("expected ok, got err=%v", err)
				}
				if eq == nil || eq.ID != tt.eqID {
					t.Fatalf("expected eq id=%s, got %+v", tt.eqID, eq)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got eq=%+v", eq)
			}
			if got := errors.CodeOf(err); got != tt.wantCode {
				t.Errorf("error code=%d, want %d", got, tt.wantCode)
			}
			if got := err.Error(); got != tt.wantMsg {
				t.Errorf("error msg=%q, want %q", got, tt.wantMsg)
			}
			if !errors.IsExposable(err) {
				t.Errorf("expected exposable error")
			}
		})
	}
}

func TestFindWeapon(t *testing.T) {
	tests := []struct {
		id     string
		exists bool
	}{
		{WeaponAWM, true},
		{WeaponM762, true},
		{WeaponS12K, true},
		{"UNKNOWN", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := FindWeapon(tt.id)
			if (got != nil) != tt.exists {
				t.Errorf("FindWeapon(%q) exists=%v, want %v", tt.id, got != nil, tt.exists)
			}
		})
	}
}

func TestWeaponListContainsAW(t *testing.T) {
	for _, w := range WeaponList() {
		if w.ID == WeaponAWM {
			return
		}
	}
	t.Error("WeaponList should contain AWM")
}
