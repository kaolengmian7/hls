# 神秘商店平均抽奖价值再平衡：80w → 50w

日期：2026-07-29

## 目标

把神秘商店单次抽奖的**加权平均获得价值**从当前的 **80.0w** 下降到约 **50w**，
以收紧商店经济产出。

## 手段与约束

- **Value（奖品展示价值）全部保持不变**：不改任何奖品的估值，玩家看到的奖品目录、
  单价、以及保底累加数值（`GuaranteedAmount`）语义完全一致。
- **只调整 Weight（服务端权重）**：`Weight` 仅服务端可见（`GetPool` 不下发），
  改动对客户端展示无副作用。
- **全局严格单调**：所有 66 件物品统一用同一条反比价值公式，实现"价值越高、爆率越低"
  的全局单调曲线，不再对稀有款做特殊手工权重。

## 权重公式

对**全部 66 件**物品：

```
Weight = round(69.53 / Value^0.85277, 2)
```

- 指数 `p = 0.85277` 是解出的唯一值，使原始（未取整）加权平均恰好 = 50。
  推导：令 `w_i ∝ 1/Value_i^p`，加权平均 `= Σ(Value·w) / Σ(w)`；对 p 做二分求解使其 = 50。
- 系数 `C = 69.53` 只做量级缩放——因 `rollOne` 只按相对权重归一化，C 不影响平均值，
  仅用于让权重落在可读区间（把最稀有的海洋之泪定在 0.10）。
- 取整到 **2 位小数**后实测加权平均 = **49.98**（满足 50 ± 0.3）。
- 权重范围 **0.10 → 7.80**，无 0 权重，对 Value 严格非增（已验证单调）。

## 结果：概率分布变化

单次抽奖落在各价值档的概率（全部 66 件）：

| 价值档 | 调整前 | 调整后 |
|---|---|---|
| <20w | 14.9% | 37.0% |
| 20–60w | 33.1% | 40.1% |
| 60–120w | 31.5% | 15.1% |
| 120–250w | 20.1% | 6.8% |
| >250w | 0.4% | 1.0% |

顶级大奖概率（作为全局单调的必然结果，会较调整前提升 2~4 倍，已确认接受）：

| 奖品 | Value | 调整前 P | 调整后 P |
|---|---|---|---|
| 海洋之泪 | 2152 | 0.014% | 0.058% |
| 非洲之心 | 998 | 0.027% | 0.109% |
| ECMO | 229 | 0.191% | 0.391% |

## 完整权重对照表（66 件）

`Weight_new = round(69.53 / Value^0.85277, 2)`

| ID | 名称 | Value | 旧 Weight | 新 Weight |
|---|---|---|---|---|
| TEAR_OF_THE_OCEAN | 海洋之泪 | 2152 | 0.05 | 0.10 |
| HEART_OF_AFRICA | 非洲之心 | 998 | 0.1 | 0.19 |
| PORTABLE_VENTILATOR | 复苏呼吸机 | 407 | 0.3 | 0.41 |
| MICRO_REACTOR | 微型反应炉 | 361 | 0.4 | 0.46 |
| ARMORED_VEHICLE_BATTERY | 装甲车电池 | 291 | 0.55 | 0.55 |
| ECMO | ECMO | 229 | 0.7 | 0.68 |
| PROTOTYPE_FUSION_POWER_UNIT | 试制聚变供能单元 | 206 | 6.08 | 0.74 |
| VERTICAL_HORIZONTAL | 纵横 | 184 | 6.08 | 0.81 |
| TOP_SECRET_SERVER | 绝密服务器 | 177 | 6.08 | 0.84 |
| GOLDEN_CROWN_OF_TEARS | 万金泪冠 | 177 | 6.08 | 0.84 |
| RADAR_PORTABLE | 便携军用雷达 | 165 | 6.08 | 0.89 |
| IMPRESSIONIST_PAINTING | 印象派名画 | 161 | 6.08 | 0.91 |
| MANDEL_SUPERCOMPUTER | 曼德尔超算单元 | 154 | 6.08 | 0.95 |
| ROBOT_VAC_MOP | 扫拖一体机器人 | 154 | 6.08 | 0.95 |
| REINFORCED_CARBON_FIBER_PLATE | 强化碳纤维板 | 154 | 6.08 | 0.95 |
| DISK_ARRAY_HS | 高速磁盘阵列 | 131 | 6.08 | 1.09 |
| FLIGHT_RECORDER | 飞行记录仪 | 131 | 6.08 | 1.09 |
| REMINGTON_TYPEWRITER | 雷明顿打字机 | 124 | 6.08 | 1.14 |
| FAMOUS_KILN_PORCELAIN | 名窑瓷器 | 115 | 6.08 | 1.22 |
| ROCKET_FUEL | 火箭燃料 | 115 | 6.08 | 1.22 |
| BUOYANCY_COMPENSATOR | 浮力补偿设备 | 115 | 6.08 | 1.22 |
| RANGEFINDER | 测距仪 | 115 | 6.08 | 1.22 |
| BATTERY_PACK | 动力电池组 | 115 | 6.08 | 1.22 |
| MIL_SHELL | 军用炮弹 | 115 | 6.08 | 1.22 |
| DISK_ARRAY_HS_COMPACT | 高速磁盘阵列 | 115 | 6.08 | 1.22 |
| AED | 自动体外除颤仪 | 108 | 6.08 | 1.28 |
| POWERFUL_VACUUM | 强力吸尘器 | 108 | 6.08 | 1.28 |
| BLADE_SERVER | 刀片服务器 | 108 | 6.08 | 1.28 |
| PORTRAIT_BUST | 半身像 | 108 | 6.08 | 1.28 |
| IFV_MODEL | 步战车模型 | 100 | 6.08 | 1.37 |
| GRAMOPHONE | 留声机 | 100 | 6.08 | 1.37 |
| CLOUD_STORAGE_ARRAY | 云储存阵列 | 100 | 6.08 | 1.37 |
| MEDICAL_ROBOT | 医疗机器人 | 92 | 6.08 | 1.47 |
| LAPTOP | 笔记本电脑 | 92 | 6.08 | 1.47 |
| MIL_RADIO | 军用电台 | 77 | 6.08 | 1.71 |
| ELECTRONIC_ANKLET | 电子脚镣 | 77 | 6.08 | 1.71 |
| MIL_INFO_TERMINAL | 军用信息终端 | 60 | 6.08 | 2.12 |
| MUSKET | 滑膛枪 | 54 | 6.08 | 2.32 |
| CAMERA | 摄影机 | 50 | 6.08 | 2.47 |
| PORTABLE_LIFE_SUPPORT_SYSTEM | 便携式生命支持系统 | 50 | 6.08 | 2.47 |
| TIANYUAN_DIFANG | 天圆地方 | 46 | 6.08 | 2.66 |
| FEMTOSECOND_LASER | 飞秒激光器 | 46 | 6.08 | 2.66 |
| HIGH_ENERGY_GAS_CANISTER | 高能瓦斯罐 | 40 | 6.08 | 2.99 |
| MIL_DRONE | 军用无人机 | 38 | 6.08 | 3.13 |
| ARCHIVED_AUDIO_SOURCE | 已封存音源 | 38 | 6.08 | 3.13 |
| GOLDEN_CROCODILE_BUST | 黄金鳄鱼头雕像 | 38 | 6.08 | 3.13 |
| SATELLITE_DISH | 卫星锅 | 38 | 6.08 | 3.13 |
| VENTILATOR | 呼吸机 | 35 | 6.08 | 3.35 |
| GOLDEN_GAZELLE | 黄金瞪羚 | 31 | 6.08 | 3.72 |
| MIL_CONTROL_TERMINAL | 军用控制终端 | 31 | 6.08 | 3.72 |
| PREMIUM_COFFEE_BEANS | 高级咖啡豆 | 29 | 6.08 | 3.94 |
| FOSSIL | 化石 | 27 | 6.08 | 4.18 |
| CHAMPAGNE | 香槟 | 26 | 6.08 | 4.32 |
| TIDAL_PRISON_MAP_4 | 潮汐监狱地图-4 | 26 | 6.08 | 4.32 |
| GPU | 显卡 | 25 | 6.08 | 4.47 |
| QUANTUM_STORAGE | 量子存储 | 23 | 6.08 | 4.80 |
| GOLD_BAR | 万足金条 | 23 | 6.08 | 4.80 |
| GILT_CARD | 鎏金卡牌 | 15 | 6.08 | 6.91 |
| ROLEX | 劳力士 | 15 | 6.08 | 6.91 |
| SAYYID_POCKET_WATCH | 赛义德的怀表 | 15 | 6.08 | 6.91 |
| ARCHIVED_EXPERIMENT_DATA | 实验数据 | 15 | 6.08 | 6.91 |
| ULTRASONIC_CUTTER | 超声波切割刀 | 15 | 6.08 | 6.91 |
| GPS_RECEIVER | 定位接收器 | 14 | 6.08 | 7.32 |
| STAR_SENSOR | 恒星敏感器 | 14 | 6.08 | 7.32 |
| ALPHA_BCI_EXPERIMENT_DATA | Alpha脑机实验数据 | 14 | 6.08 | 7.32 |
| CAVIAR | 鱼子酱 | 13 | 6.08 | 7.80 |

（表按 Value 降序排列以便核对单调性；实际写入时保持 `gameserver.yaml` 中的原有条目顺序，
仅逐条替换 `Weight` 字段。）

## 影响范围

- **唯一改动文件**：`server/etc/gameserver.yaml`——把 `Shop.LotteryPool` 下 66 个条目的
  `Weight` 逐一替换为上表"新 Weight"。**不改任何 Go 代码。**
- **保底机制不受影响**：Value 不变 → `DrawLottery` 中 `guarantee += item.Value` 的累加，
  以及 `MaxGuaranteedAmount`（999999）回滚判定逻辑完全不变。
- **VIP `highBoost` 不受影响**：`applyHighBoost` 仍按 HIGH 稀有度对权重做倍增，机制不变；
  VIP 玩家的平均值会自然高于 50（符合预期，非本次调整目标）。
- **单元测试不受影响**：`server/internal/game/lottery_test.go` 使用自带的
  `defaultTestPool()`（独立硬编码奖池，不读 yaml），断言不涉及本 yaml 的具体权重或平均值。

## 验证方式

1. 用一次性脚本重算 `gameserver.yaml` 里改动后的加权平均，确认落在 **50 ± 0.3**。
2. `cd server && go test ./...` 保持全绿。
3. 目视核对：新 Weight 对 Value 严格非增（价值越高、权重越低）。
