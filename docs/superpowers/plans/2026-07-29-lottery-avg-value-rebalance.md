# 神秘商店平均抽奖价值再平衡（80w→50w）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 `server/etc/gameserver.yaml` 中 `Shop.LotteryPool` 的 66 个 `Weight` 按反比价值公式重算，使单次抽奖加权平均获得价值从 80.0w 降到 ~50w（实测 49.98w），Value 全部不变。

**Architecture:** 纯配置改动。用一次性 Python 脚本读取每个奖池条目的 `Value`，按 `Weight = round(69.53 / Value^0.85277, 2)` 逐条重写该条目的 `Weight` 行，保留文件其余部分（顺序、缩进、注释、ImageURL 引号）不变。不改任何 Go 代码。

**Tech Stack:** YAML 配置（go-zero conf 加载）、Python 3（一次性转换/校验脚本，不入库）、Go test。

## Global Constraints

- **只改 `Weight` 字段**：Value、ID、Name、Rarity、ImageURL 及条目顺序一律不动。
- **不改任何 Go 代码**：`server/internal/game/lottery.go`、`svc`、测试均不动。
- **权重公式（逐条、对全部 66 件统一）**：`Weight = round(69.53 / Value^0.85277, 2)`
  - `p = 0.85277`，`C = 69.53`。
- **验收阈值**：改后加权平均落在 **50 ± 0.3**；权重对 Value 严格非增；无 0 权重。
- **脚本产物不入库**：转换脚本与校验脚本写在 scratchpad，最终提交只含 `gameserver.yaml` 的改动。
- 详见 spec：`docs/superpowers/specs/2026-07-29-lottery-avg-value-rebalance-design.md`。

---

### Task 1: 重算奖池权重并落盘到 gameserver.yaml

**Files:**
- Modify: `server/etc/gameserver.yaml`（`Shop.LotteryPool` 下 66 个条目的 `Weight` 行）
- Create（scratchpad，不入库）：`<scratchpad>/rebalance_weights.py`、`<scratchpad>/check_avg.py`

**Interfaces:**
- Consumes: 无（起始任务）。
- Produces: 无下游代码依赖；产出是改后的 `gameserver.yaml`。

- [ ] **Step 1: 写校验脚本，先确认当前基线 = 80w（"失败态"）**

在 scratchpad 建 `check_avg.py`（`<scratchpad>` = 本会话 scratchpad 目录）：

```python
import re, sys, math
path = sys.argv[1]
txt = open(path).read()
blocks = re.findall(
    r"- ID:\s*(\S+).*?Rarity:\s*(\w+).*?Value:\s*(\d+).*?Weight:\s*([\d.]+)",
    txt, re.S)
items = [(id, r, int(v), float(w)) for id, r, v, w in blocks]
assert len(items) == 66, f"expected 66 items, got {len(items)}"
tot = sum(w for _, _, _, w in items)
avg = sum(v * w for _, _, v, w in items) / tot
print(f"items={len(items)} total_weight={tot:.2f} weighted_avg={avg:.3f}")
# monotonic check: higher Value must not have higher Weight
by_val = {}
for _, _, v, w in items:
    by_val.setdefault(v, w)
vals = sorted(by_val, reverse=True)
mono = all(by_val[vals[i]] <= by_val[vals[i+1]] for i in range(len(vals)-1))
print("monotonic_non_increasing_in_value:", mono)
print("zero_weight_items:", [i for i in items if i[3] == 0])
```

Run: `python3 <scratchpad>/check_avg.py server/etc/gameserver.yaml`
Expected: `weighted_avg` ≈ **80.0**（确认基线；这是我们要改掉的状态）。

- [ ] **Step 2: 写转换脚本（逐条重写 Weight）**

在 scratchpad 建 `rebalance_weights.py`：

```python
import re, sys
P, C = 0.85277, 69.53
path = sys.argv[1]
txt = open(path).read()

def rewrite_item(block):
    m = re.search(r"Value:\s*(\d+)", block)
    v = int(m.group(1))
    new_w = round(C / (v ** P), 2)
    # 替换该条目内的 Weight 行，保留缩进
    return re.sub(r"(\n\s*Weight:\s*)[\d.]+", rf"\g<1>{new_w}", block, count=1)

# 按 "- ID:" 切块（保留分隔符），只处理 LotteryPool 条目块
parts = re.split(r"(?=\n\s*- ID:)", txt)
out = [rewrite_item(p) if re.search(r"\n\s*- ID:", p) and "Weight:" in p else p
       for p in parts]
open(path, "w").write("".join(out))
print("done")
```

- [ ] **Step 3: 运行转换脚本，改写 yaml**

Run: `python3 <scratchpad>/rebalance_weights.py server/etc/gameserver.yaml`
Expected: 打印 `done`。

- [ ] **Step 4: 校验加权平均落在 50 ± 0.3，且单调、无 0 权重（"通过态"）**

Run: `python3 <scratchpad>/check_avg.py server/etc/gameserver.yaml`
Expected:
- `weighted_avg` ≈ **49.98**（在 50 ± 0.3 内）
- `monotonic_non_increasing_in_value: True`
- `zero_weight_items: []`

- [ ] **Step 5: 目视核对 diff——只有 Weight 变了**

Run: `git diff --stat && git diff server/etc/gameserver.yaml | grep '^[+-]' | grep -v Weight`
Expected: `--stat` 只显示 `gameserver.yaml` 一个文件；第二条命令除 diff 头（`+++`/`---`）外**无输出**（即所有变更行都是 `Weight:`）。抽查几条对照 spec 表：海洋之泪 `0.05→0.1`、试制聚变供能单元 `6.08→0.74`、鱼子酱 `6.08→7.8`。

- [ ] **Step 6: 跑 Go 测试确认无回归**

Run: `cd server && go test ./...`
Expected: 全部 PASS（`lottery_test.go` 用独立 `defaultTestPool()`，不读该 yaml，应不受影响）。

- [ ] **Step 7: 提交（仅 yaml）**

```bash
git add server/etc/gameserver.yaml
git commit -m "feat(config): rebalance lottery weights to ~50w avg draw value

Reweight all 66 Shop.LotteryPool items via Weight=round(69.53/Value^0.85277,2),
dropping weighted-average draw value from 80w to 49.98w. Values unchanged;
globally monotonic (higher value -> lower weight). Spec:
docs/superpowers/specs/2026-07-29-lottery-avg-value-rebalance-design.md

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

Expected: 提交成功，`git status` clean。

---

## Self-Review

**1. Spec coverage:**
- 权重公式 `round(69.53/Value^0.85277, 2)` → Task 1 Step 2 脚本实现。✓
- 只改 Weight、Value 不变 → Step 2 脚本只替换 Weight 行；Step 5 diff 校验。✓
- 全局严格单调、无 0 权重 → Step 4 校验。✓
- 平均 50 ± 0.3 → Step 1（基线 80）/ Step 4（结果 ~49.98）。✓
- 保底/highBoost/测试不受影响 → Value 不变（无代码改动）；Step 6 `go test` 兜底。✓
- 唯一改动文件 `gameserver.yaml` → Step 5 `git diff --stat` 校验。✓

**2. Placeholder scan:** 无 TBD/TODO；所有脚本、命令、期望输出均为具体内容。✓

**3. Type consistency:** 仅一个任务，公式常量 `p=0.85277`/`C=69.53` 在 Global Constraints 与脚本中一致。✓
