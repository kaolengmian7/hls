# 教学系统设计（Teaching System）

- 日期：2026-08-21
- 分支：`feat/enforcer-payroll-management`
- 状态：设计已确认，待进入实现计划（writing-plans）

## 1. 背景与目标

在现有平台上新增一个**长周期教学子系统**。业务模型：

- **打手（`enforcer`）作为老师**，**客户（`customer`）作为学员**。
- 一个教学单售卖一段较长的总课时（如 6 小时、14 小时、30 小时）。
- 老师每次上完课：**录入本节课时长（从总时长扣减）**，并按**固定教学大纲**给学员逐项打分（每项 0–10，作为成绩单），可附一段评语。
- 老师能看到该教学单的**成绩单列表**。
- 学员能在自己后台的**「我的教学」**里看到该教学单的成绩单列表。
- 店长（`owner`）可以为教学单**续时长**。
- 全部课时用完后走**结单流程发工资**，工资**完全复用现有 payroll 流**（`payroll_items` + 押金/罚款扣减 + 「工资管理」面板）。
- 教学单**有学员评价**，学员评价驱动奉金达标（与订单同一套 `total>=8` 规则）。

### 已确认的关键决策

| 决策点 | 结论 |
| --- | --- |
| 实体建模 | 独立 `teaching_orders` 实体（不复用 `orders` 表） |
| 分数存储 | 成绩单分数存 JSON（`{item_key: 0-10}`），留空的项不出现 |
| 学员账号 | 店长/客服建单时指定已有 `customer` 或顺手新建一个 |
| 大纲 | 写死在**配置文件**里，全局共用，后端出接口给前端 |
| 成绩单规则 | 每节**必填时长**，16 项评分可留空 |
| 老师分配 | 单个老师，**打手自领** |
| 建单权限 | 店长 + 客服（`owner` / `service`）；续时长仅店长 |
| 时长台账 | 保留完整台账（`teaching_time_logs`） |
| 工资算法 | 订单价 × 分成比例（含奉金），公式与订单一致 |
| 结单触发 | 打手（该单老师）触发，**剩余课时=0** 才可结 |
| 工资流 | 完全复用现有 payroll 流 |
| 奉金 | 有奉金，由**学员评价**驱动（`total>=8` 达标） |

### 非目标（v1 不做）

- 教学单退款/作废流程（预留状态位即可，暂不实现）。
- 排课/日程（只事后记录已上的课，不预约未来课程）。
- 大纲可配置化的增删改管理界面（仅通过改配置文件调整）。
- 教学单的投诉态（订单的 `complaint_status` 语义对教学无意义）。
- 多老师协作教学。

## 2. 现有代码约定（依据）

- 后端 Go 1.25 + go-zero，分层 `handler → repo → db`，DI 容器 `svc.ServiceContext`。
- MySQL；迁移是 `server/migrations/` 下**手工执行的编号 SQL 文件**，无迁移工具；`server/db/schema.sql` 为全量快照。
- 约定：`id BIGINT AUTO_INCREMENT PK`；金额 `DECIMAL(10,2)`、比率 `DECIMAL(5,2)`、名字 `VARCHAR(50)`；状态用 `ENUM`；每表 `created_at/updated_at`；**无外键约束、无软删除**，跨表按业务键（`code`）或 id 关联，完整性靠事务保证。
- repo 模式：`const xCols` 复用列清单；读用 `QueryRow(s)Ctx`、写用 `ExecCtx`；多步写用 `TransactCtx` + `sqlx.Session`；并发保护用条件式 `UPDATE ... WHERE ...` 判 `RowsAffected==0` 返回哨兵错误（如 `ErrSlotTaken`、`ErrNotSettleable`）。
- 路由集中在 `server/cmd/gameserver/main.go` 的 `AddRoutes`；鉴权用 JWT bearer + `requireOwner` / `requireAnyRole` 包装 + 单据内归属校验。
- 角色：`owner`(店长) / `service`(客服) / `enforcer`(打手) / `customer`(学员)，同存 `users.role`，无独立打手/学员表。
- 前端 Vue3 + TS + Pinia；后台是单路由 `/admin`（`AdminPage.vue`）按角色过滤 tab；每个 tab 一个 `src/pages/admin/*Panel.vue`；API 客户端在 `src/api/*.ts`，共享 axios 实例 `src/http.ts`（注入 bearer、401 跳登录）。
- `/admin` 路由 guard（`src/router/index.ts`）当前只放行 `owner/service/enforcer`，**`customer` 尚无后台入口**。

### 关键集成点（本设计必须处理）

现有工资流**硬绑定 `orders` 表**：

- `PayrollRepo.Pay()`（`server/internal/repo/payroll.go`）发工资时 `SELECT ... FROM orders WHERE code = item.OrderCode` 并调 `o.Earnings()` 实时重算 gross，还检查 `o.Status == "refunded"`。
- 工资面板 `ListPayrollHandler`（`server/internal/handler/payroll.go`）用 `ordersByCode` 反查订单标题、`projectPayrollView` 里调 `order.Earnings()` 重算待发金额。

若教学单是独立表，这两处按 code 反查 `orders` 会落空。解决方式见 §5「工资流集成」。

现有奉金机制：`reviewEffects()`（`server/internal/handler/review.go`）规则 `total = attitude + technical`，`total>=8` → `bonus_qualified`；`ReviewRepo.UpsertWithEffects` 在一个事务里写 `reviews` + 回写 `orders.bonus_qualified`。教学镜像此机制，见 §6。

## 3. 数据模型

### 3.1 配置：教学大纲

大纲写死在 `server/etc/gameserver.yaml`，config struct 新增 `TeachingSyllabus`。每项一个**稳定 `key`**（分数按 key 存，故改 label/desc 不影响历史成绩单，仅增删 key 有影响）：

```yaml
TeachingSyllabus:
  - key: gunplay
    label: 枪法与身法
    items:
      - { key: pre_aim,        label: 预瞄,        desc: "移动时准星始终保持在敌方头部高度与常见拉出点，做到出掩体即开枪。" }
      - { key: peek,           label: 闪身枪与小身位Peek, desc: "利用掩体极快速拉出并收回身位，分为探头Peek与提速闪身枪。" }
      - { key: jump_slide_peek,label: 跳拉与铲拉,   desc: "利用跳跃或滑铲大身位快速拉出掩体，打破敌方架枪的爆头线与预瞄节奏。" }
      - { key: prone_dodge,    label: 拜佛,        desc: "近距离贴脸对枪时突然卧倒，躲避对方第一波爆头线。" }
      - { key: bunny_hop,      label: 兔子跳,      desc: "跳拉出去后连续跳跃，躲避敌方枪线，增加敌方跟枪难度。" }
      - { key: recoil_control, label: 控枪,        desc: "熟练掌握主流枪械的前20发后坐力规律。" }
      - { key: tracking,       label: 跟枪,        desc: "瞄准移动靶的能力。" }
      - { key: spray_transfer, label: 扫射转移,    desc: "同时面对多个敌人时的枪械控制能力。" }
  - key: awareness
    label: 意识与信息收集
    items:
      - { key: map_headlines,  label: 地图地形与爆头线, desc: "熟记各地图高频交战点、优势掩体、高地位以及撤离/复活路线。" }
      - { key: sound_sense,    label: 听声辨位与细节感知, desc: "通过音效判断脚声、换弹声、修甲声及技能音效的距离与空间感知。" }
      - { key: info_integration, label: 地图与信息流整合, desc: "收集地图信息，推算敌方小队的人数与包抄路径。" }
      - { key: rotate_bait,    label: 转场与拉扯,   desc: "利用道具或地形转场，创造侧面偷袭机会。" }
  - key: utility_teamwork
    label: 道具与干员协同
    items:
      - { key: utility_clear, label: 道具清点,     desc: "熟练运用道具将死角架枪的敌人逼出掩体。" }
      - { key: oneway_skill,  label: 单向视线与技能组合, desc: "运用道具制造单向输出环境，发挥不同兵种的功能定位。" }
      - { key: crossfire,     label: 交叉火力与补枪, desc: "与队友形成不同角度的交叉架枪线，第一时间拉出补枪。" }
      - { key: callout,       label: 高效战术报点,  desc: "遵循方向+参照物+状态原则，拒绝无效废话。" }
```

共 3 类 16 项。后端出接口 `GET /api/teaching/syllabus` 供前端渲染，前端不重复维护一份。

### 3.2 表 `teaching_orders`

钱字段刻意对齐 `orders`，以复用同一套 earnings 公式。

```sql
CREATE TABLE teaching_orders (
  id                BIGINT AUTO_INCREMENT PRIMARY KEY,
  code              VARCHAR(20) NOT NULL,              -- 带前缀（如 JX+时间戳），与订单 code 命名空间隔离
  title             VARCHAR(50) NOT NULL,
  student_id        BIGINT NOT NULL,
  student_name      VARCHAR(50) NOT NULL,
  dispatcher_id     BIGINT NOT NULL,                   -- 建单人（owner/service）
  dispatcher_name   VARCHAR(50) NOT NULL,
  teacher_id        BIGINT NULL,                       -- enforcer，自领前为空
  teacher_name      VARCHAR(50) NULL,
  price             DECIMAL(10,2) NOT NULL DEFAULT 0,  -- 随续时长累加
  commission_rate   DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_rate        DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_qualified   TINYINT(1)    NOT NULL DEFAULT 0,  -- 由学员评价回写
  total_minutes     INT NOT NULL DEFAULT 0,            -- 累计总课时（含续时）
  remaining_minutes INT NOT NULL DEFAULT 0,            -- 剩余；= 台账 delta 求和，冗余存一份做快速判断/并发保护
  status            ENUM('dispatching','ongoing','settled') NOT NULL DEFAULT 'dispatching',
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_code (code),
  INDEX idx_status (status),
  INDEX idx_teacher (teacher_id),
  INDEX idx_student (student_id)
) DEFAULT CHARSET=utf8mb4;
```

`player_count` 恒为 1（教学单老师唯一），故复用 earnings 公式中的 `/playerCount` 天然成立（实现时以常量 1 传入）。

状态流转：`dispatching`（已建、无老师）→ `ongoing`（被自领）→ `settled`（结单发工资）。

### 3.3 表 `teaching_scorecards`（一节课一张，分数 JSON）

```sql
CREATE TABLE teaching_scorecards (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  duration_minutes    INT NOT NULL,                    -- 本节时长，必填、必须 >0、≤ 剩余
  scores              JSON NULL,                       -- {"pre_aim":8,"peek":6,...}；留空项不出现
  comment             TEXT NULL,
  teacher_id          BIGINT NOT NULL,
  teacher_name        VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;
```

JSON 扫描：go-zero sqlx 不自动解 JSON，Go 侧把该列扫成 `string`，用 `json.Unmarshal` 进 `map[string]int`；写入前 `json.Marshal`。分数值域 0–10，写入前做范围校验（超范围或非法 key 拒绝）。

### 3.4 表 `teaching_time_logs`（时长完整台账）

```sql
CREATE TABLE teaching_time_logs (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  delta_minutes       INT NOT NULL,                    -- +初始 / +续时 / -上课
  reason              ENUM('initial','topup','lesson') NOT NULL,
  ref_scorecard_id    BIGINT NULL,                     -- lesson 关联成绩单
  price_delta         DECIMAL(10,2) NOT NULL DEFAULT 0,-- 续时同时加的钱，留痕
  operator_id         BIGINT NOT NULL,
  operator_name       VARCHAR(50) NOT NULL,
  operator_role       VARCHAR(20) NOT NULL,
  note                VARCHAR(255) NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;
```

不变式：`teaching_orders.remaining_minutes == SUM(delta_minutes)`；`teaching_orders.total_minutes == SUM(delta_minutes WHERE delta>0)`。冗余字段与台账在同一事务内一起更新，保证一致。

### 3.5 表 `teaching_reviews`（学员评价，一单一条）

```sql
CREATE TABLE teaching_reviews (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  attitude_score      INT NOT NULL,                    -- 学员给老师：态度分
  technical_score     INT NOT NULL,                    -- 学员给老师：教学/技术分
  comment             TEXT NULL,
  reviewer_id         BIGINT NOT NULL,                 -- 学员
  reviewer_name       VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;
```

分值语义与订单评价一致（attitude/technical 各自沿用订单现有量程）。不带订单那批游戏单专属布尔 flag（`no_backpack`/`grab_container` 等）——与教学无关，YAGNI 掉。

### 3.6 `payroll_items` 加一列（复用工资流所需）

```sql
ALTER TABLE payroll_items
  ADD COLUMN source ENUM('order','teaching') NOT NULL DEFAULT 'order';
```

现有行默认 `order`，向后兼容。教学结单插入的工资条 `source='teaching'`。`UNIQUE(order_code, enforcer_id)` 不变——教学 code 带前缀，与订单 code 不冲突，可安全共用该唯一键。

## 4. 后端行为与事务

所有多步写走 `TransactCtx`；并发保护用条件式 `UPDATE` 判 `RowsAffected`。新增哨兵错误：`ErrTeacherTaken`、`ErrInsufficientTime`、`ErrTeachingNotSettleable`、`ErrNotTeacher`、`ErrTeachingNotFound`。

### 4.1 建单（owner / service）

事务：
1. 若表单选择"新建学员"，先创建 `role=customer` 用户（复用现有 UsersPanel 建用户逻辑），拿到 `student_id/name`。
2. 生成带前缀的 `code`，插 `teaching_orders`（`total=remaining=初始分钟`，`status='dispatching'`，`price/commission_rate/bonus_rate` 来自表单）。
3. 插 `teaching_time_logs(reason='initial', delta=+初始分钟, operator=建单人)`。

校验：初始分钟 > 0；`commission_rate` ∈ [0,100]；学员存在且 `role=customer`。

### 4.2 自领为老师（enforcer）

```sql
UPDATE teaching_orders
   SET teacher_id=?, teacher_name=?, status='ongoing'
 WHERE code=? AND teacher_id IS NULL AND status='dispatching';
```

`RowsAffected==0` → `ErrTeacherTaken`。

### 4.3 录成绩单（enforcer = 该单老师）

前置校验：caller == `teacher_id`（否则 `ErrNotTeacher`）；`status='ongoing'`；`duration_minutes > 0`；`scores` 里 key 合法、值 ∈ [0,10]。

事务：
1. 插 `teaching_scorecards`（`scores` 序列化为 JSON）。
2. 插 `teaching_time_logs(reason='lesson', delta=-duration, ref_scorecard_id=新id, operator=老师)`。
3. 条件式扣减：
   ```sql
   UPDATE teaching_orders SET remaining_minutes = remaining_minutes - ?
    WHERE code=? AND remaining_minutes >= ?;
   ```
   `RowsAffected==0` → `ErrInsufficientTime`（这就是并发保护，防止两次并发录课把剩余扣成负）。

### 4.4 续时长（owner）

请求：`addMinutes > 0`，`addPrice >= 0`。事务：
1. 插 `teaching_time_logs(reason='topup', delta=+addMinutes, price_delta=+addPrice, operator=店长)`。
2. `UPDATE teaching_orders SET total_minutes+=addMinutes, remaining_minutes+=addMinutes, price=price+addPrice WHERE code=?`。

price 同步累加，保证多上的课有对应工资口径（工资算法 = 订单价 × 分成）。

### 4.5 结单发工资（enforcer = 该单老师）

前置校验：caller == `teacher_id`。事务：
```sql
UPDATE teaching_orders SET status='settled'
 WHERE code=? AND status='ongoing' AND remaining_minutes=0;
```
`RowsAffected==0` → `ErrTeachingNotSettleable`（剩余非 0 或状态不符）。同一事务内：
```sql
INSERT IGNORE INTO payroll_items (order_code, enforcer_id, enforcer_name, source, status)
VALUES (<教学code>, teacher_id, teacher_name, 'teaching', 'pending');
```
gross 此刻不写、发放时实时算，与订单一致；结单冷静期（`Payroll.SettleDelayDays`）天然继承（`payroll_items.created_at` 驱动的冻结/待发/已发分类无需改动）。`INSERT IGNORE` 保证重复结单幂等。

### 4.6 学员评价（customer = 该单学员）

`PUT /api/teaching-orders/:code/review`：校验 caller == `student_id`。镜像 `ReviewRepo.UpsertWithEffects`，事务内：
1. `INSERT ... ON DUPLICATE KEY UPDATE` 写 `teaching_reviews`（一单一条，可重复提交覆盖）。
2. 用共享的奉金判定（见 §6）算出 `bonusQualified`，`UPDATE teaching_orders SET bonus_qualified=? WHERE code=?`。

## 5. 工资流集成（改动最敏感处）

目标：教学工资与订单工资进**同一个「工资管理」面板、同一套押金/罚款扣减、同一个发放按钮**。需要动的共享代码：

1. **抽取纯函数** `ComputeEarnings(price, commissionRate, bonusRate, bonusQualified float64/bool, playerCount int) float64`，把 `Order.Earnings()` 的公式搬进去；`Order.Earnings()` 改为调它。教学单用同一函数（`playerCount=1`）。公式不变：
   `per = (price*(1-commission/100) + (bonusQualified ? price*bonus/100 : 0)) / playerCount`，四舍五入两位。
2. **`payroll_items` 加 `source` 列**（§3.6）。
3. **`PayrollRepo.Pay()` 按 `item.source` 分支**：
   - `order`：走现有 `SELECT FROM orders`，逻辑不变。
   - `teaching`：改 `SELECT FROM teaching_orders`，用同一 `ComputeEarnings` 算 gross；`refunded` 检查换成教学作废态（v1 教学无退款，恒 false）。
   - 押金/罚款扣减（`ComputeDeduction` + `enforcer_accounts` 更新）**完全不动**——老师本身是 enforcer，同一套押金账户。
4. **工资面板 `ListPayrollHandler`**：把 `ordersByCode` 换成一个同时装订单与教学单的合并结构 `map[string]payrollSource{ Title string; Gross float64; Refunded bool }`（订单来源用 `Order.Earnings()`，教学来源用 `ComputeEarnings`）；`projectPayrollView` 接这个结构而非死绑 `*repo.Order`。`paidPayrollView` 走冻结快照，天然无关来源。面板 Vue 组件无需改动——教学工资条自然出现在同一列表，标题=教学单标题。

`Pay()` 需要能在其事务内查 `teaching_orders`：在同一 `session` 上内联查询即可（无需额外 repo 依赖）。

## 6. 奉金判定复用

`reviewEffects()` 现同时算 `total`、`bonusQualified`、`newComplaint`。教学只需要 `bonusQualified = (attitude+technical) >= 8`。

- 抽一个可共享的小判定（如 `BonusQualifies(attitude, technical int) bool { return attitude+technical >= 8 }`），`reviewEffects` 内部改调它，教学评价 upsert 也调它，保证师徒两套口径一致。
- 教学不计算/存储 `complaint_status`。

奉金在**发工资时实时算**（`Pay()`/面板读 `teaching_orders.bonus_qualified` 再经 `ComputeEarnings`），所以只要学员在发放前提交评价即计入，与订单"后补评价影响奉金"一致。

## 7. API 一览

| 方法 | 路径 | 权限 |
| --- | --- | --- |
| POST | `/api/teaching-orders` | owner, service |
| GET | `/api/teaching-orders` | owner, service（全部列表） |
| GET | `/api/teaching-orders/:code` | owner, service, 该单 teacher, 该单 student |
| POST | `/api/teaching-orders/:code/claim` | enforcer |
| POST | `/api/teaching-orders/:code/topup` | owner |
| POST | `/api/teaching-orders/:code/settle` | enforcer（= 该单 teacher） |
| GET | `/api/teaching-orders/:code/scorecards` | owner, 该单 teacher, 该单 student |
| POST | `/api/teaching-orders/:code/scorecards` | enforcer（= 该单 teacher） |
| GET | `/api/teaching-orders/:code/timelogs` | owner |
| PUT | `/api/teaching-orders/:code/review` | customer（= 该单 student） |
| GET | `/api/teaching-orders/:code/review` | owner, 该单 teacher, 该单 student |
| GET | `/api/my-teaching` | enforcer（看我教的）/ customer（看我学的） |
| GET | `/api/teaching/syllabus` | 任意登录 |
| GET | `/api/customers` | owner, service（建单选学员用；返回 id/昵称/手机号） |

鉴权沿用 `requireAnyRole` + 单据内归属校验（照搬 `SettleOrderHandler` 的"caller 必须占该单槽位"写法）。`/api/my-teaching` 按 caller 角色分支：`enforcer` 查 `teacher_id=caller`，`customer` 查 `student_id=caller`。

响应 view 用 map builder 投影（照搬 `toOrderView`），不直接序列化 DB struct。

## 8. 前端

- `src/api/teaching.ts`：教学相关接口客户端（复用共享 `http` 实例）。
- `src/stores/`：一个 syllabus store，从 `GET /api/teaching/syllabus` 拉取并缓存，师生共用同一份渲染。
- `src/router/index.ts`：`/admin` guard 放行 `customer`。
- `src/pages/AdminPage.vue`：tab 列表按角色新增 3 个 tab（复用现有 `allTabs` + `roles` 过滤模式）。
- 课时单位：库存分钟（int，避免浮点）；前端按小时输入（0.5 步进）显示与换算。

三个新 Panel（`src/pages/admin/`）：

1. **教学管理**（`owner`, `service`）：建单（选/新建学员、填初始课时+价+分成率+奉金率）、教学单列表、续时长（仅 `owner` 可见入口）、查看成绩单、查看时长台账、查看学员评价。
2. **我的教学·老师**（`enforcer`）：可领教学单 + 我教的；自领；录成绩单（按大纲渲染 16 项 0–10 输入、可留空 + 时长必填 + 评语）；结单（剩余=0 才可点）；看该单成绩单列表与学员评价。
3. **我的教学·学员**（`customer`）：我学的教学单列表 + 剩余课时 + 成绩单列表（只读，看每节各项得分+评语+进度）；「评价老师」表单（态度分+技术分+评语，可重复提交覆盖）。

## 9. 测试

沿用现有风格（纯函数单测 + repo 事务测试，参考 `payroll_repo_test.go`、`ComputeDeduction`）：

- **纯函数**：`ComputeEarnings` 单测；`BonusQualifies` 单测；证明 `Order.Earnings()` 抽取后无回归（同输入同输出）。
- **repo 事务**：
  - 录成绩单条件式扣减的并发保护：剩余不足报 `ErrInsufficientTime`、绝不扣负。
  - 续时正确累加 `total_minutes`/`remaining_minutes`/`price` 并写台账。
  - 自领并发：第二个自领得 `ErrTeacherTaken`。
  - 结单只在 `remaining=0 && status=ongoing` 成功，生成恰好一条 `payroll_items(source='teaching')`，`INSERT IGNORE` 幂等。
  - 台账不变式：`remaining_minutes == SUM(delta_minutes)`。
- **集成点**：`Pay()` 对 `source='teaching'` 的工资条能从 `teaching_orders` 正确算 gross 并走押金/罚款扣减；学员评价 upsert 正确回写 `bonus_qualified` 并在发放时体现于 gross。

## 10. 代码落点

- 迁移：`server/migrations/005_create_teaching.sql`（4 张新表）、`server/migrations/006_payroll_add_source.sql`（加列）；手工执行；同步 `server/db/schema.sql`。
- 后端：
  - `server/internal/repo/teaching.go`（`TeachingRepo`：教学单/成绩单/台账/评价的读写与事务）。
  - `server/internal/handler/teaching.go`（各接口 + view 投影）。
  - `server/internal/svc/service_context.go` 挂 `TeachingRepo`。
  - `server/cmd/gameserver/main.go` 注册路由。
  - `server/internal/repo/order.go`（抽 `ComputeEarnings`）、`server/internal/repo/payroll.go`（`Pay()` 按 source 分支）、`server/internal/handler/payroll.go`（面板合并来源）、`server/internal/handler/review.go`（抽 `BonusQualifies`）。
  - `server/internal/config`（加 `TeachingSyllabus`）+ `server/etc/gameserver.yaml`。
- 前端：`src/api/teaching.ts`、`src/pages/admin/` 下 3 个新 Panel、`src/pages/AdminPage.vue`、`src/router/index.ts`、syllabus store。
- 分支：`feat/enforcer-payroll-management`（沿用当前分支）。

## 11. 风险与注意

- **教学 code 前缀**：必须与订单 code 命名空间隔离，避免 `payroll_items(order_code, enforcer_id)` 唯一键在两个来源间碰撞。
- **共享工资代码改动**：`ComputeEarnings` 抽取与 `Pay()` 分支要有回归测试护栏，确保现有订单工资行为不变。
- **JSON 列**：确认目标 MySQL 版本支持 `JSON` 类型（5.7+）；否则退化为 `TEXT` 存 JSON 字符串（对本设计的 Go 侧 marshal/unmarshal 无影响）。
- **手工迁移**：与现有约定一致，无自动 runner；上线需人工在库上执行 005/006。
