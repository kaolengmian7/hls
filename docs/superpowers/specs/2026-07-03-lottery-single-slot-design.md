# 神秘商店老虎机 · 单格大图 + CSGO 开箱动效 设计文档

**日期**：2026-07-03  
**文件**：`src/games/shop/components/LotteryPanel.vue`

---

## 目标

将老虎机从 3 行窗口改为单格全幅展示，配合 CSGO 风格的「快速飞过 → 极慢停稳」减速曲线，放大奖品视觉冲击力。

---

## 变更范围

只修改 `LotteryPanel.vue`，不涉及后端、store、协议或其他组件。

---

## 动画参数

| 参数 | 旧值 | 新值 | 说明 |
|------|------|------|------|
| `VISIBLE_ROWS` | `3` | `1` | 单格窗口 |
| `CENTER_ROW` | `1` | `0` | 自动随 `VISIBLE_ROWS` 变化 |
| `ITEM_H` | `96` | `240` | 格高（px） |
| `VIEWPORT_H` | `288` | `240` | `ITEM_H × VISIBLE_ROWS` |
| `SPIN_SPEED` | `ITEM_H × 17` | `ITEM_H × 22` | 快速阶段速度（px/s） |
| `MIN_SPIN_MS` | `700` | `1000` | 快速飞过时长（ms） |
| `LAND_MS` | `2300` | `3000` | 减速着陆时长（ms） |
| `LAND_ITEMS` | `16` | `22` | 着陆阶段额外滚过格数 |
| 缓动函数 | `easeOutQuart`（4次方） | `easeOutSextic`（6次方） | 更剧烈的慢尾 |

---

## 图片尺寸

| | 旧 | 新 |
|--|--|--|
| 图片 / 占位符 | 64 × 64 px | 160 × 160 px |
| border-radius | 10px | 16px |

---

## 视觉变化

### 滚动中（spinning / landing）

- viewport 固定 240 × 240 px，overflow hidden，单格铺满
- 上下渐隐遮罩缩小至 28px（仅遮边缘，不遮奖品主体）
- 图片**不加模糊**，用户可辨认每个经过的奖品
- 结果框（中间行指示线）保持，spinning 时用低透明度金边

### 开奖停稳（revealed）

- 结果框边框加亮至 `rgba(212,175,55,0.95)`，伴随 `inset` + 外部发光
- 中奖格图片添加 `revealPulse` 脉冲光晕动效（1.8s 无限循环）
- 四角出现 `✦` 装饰（`revealed` 时 CSS transition 从透明淡入）

### 缓动曲线

```
easeOutSextic(t) = 1 - (1 - t)^6
```

相比旧的 4 次方，6 次方在最后 20% 时间内运动量极少，制造「就是这个吗？」的悬疑感。

---

## 模板结构变化

```
旧：
  <div v-else class="relative" :style="{ height: VIEWPORT_H }">
    <div :style="stripStyle">          ← 3 行滚动条
      <div v-for="row" ...>
        <img style="width:64px;height:64px" />
      </div>
    </div>
    <!-- 上下遮罩 height: ITEM_H (96px) -->
    <!-- 结果框 top: CENTER_ROW * ITEM_H = 96px -->
  </div>

新：
  <div v-else class="relative" :style="{ height: VIEWPORT_H }">
    <div :style="stripStyle">          ← 单列滚动条不变
      <div v-for="row" ...>
        <img style="width:160px;height:160px" />  ← 放大
      </div>
    </div>
    <!-- 上下遮罩 height: 28px（缩小） -->
    <!-- 结果框 top: 0（CENTER_ROW=0，inset:0） -->
    <!-- 四角 ✦ span × 4，revealed 时淡入 -->
  </div>
```

---

## 不变的部分

- 抽奖流程（`drawOne` → `tick` → `beginLanding` → `revealed`）完全不变
- `RECYCLE_AT = 20`、`RESULT_TIMEOUT_MS = 6000` 不变
- 开奖信息面板（价值、保底变化文字）不变
- `failedImages` 占位符逻辑不变
