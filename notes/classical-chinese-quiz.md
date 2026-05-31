# 文言文选择题产品实现计划

## 1. 变更后的核心规则

- 角色拆分为三类：`admin` 管理员、`teacher` 教师、`student` 学生。
- 学生和教师账号统一由管理员在 Admin 后台创建；教师 MVP 阶段不能创建账号。
- 系统需要班级域模型：管理员创建班级，并把教师、学生绑定到班级；教师只能查看自己班级内的学生数据。
- 管理员和教师都可以录入、编辑、停用题目；学生不能管理题目。
- 学生每次测试固定为 10 道题；同一次测试内题目不允许重复，题目顺序不重要。
- 学生同一时间只能有 1 个未完成测试；存在 `in_progress` 测试时不能创建新测试，只能继续当前测试。
- 学生提交每道题后立即返回反馈：是否正确、本题得分、当前总积分；答错时返回正确答案和解析。
- 每题答对最多获得 1 次积分：同一学生同一题首次答对时加 1 分；后续测试再次遇到同题即使答对也不再加分。
- 积分只增不减，除管理员执行重置分数外，普通答题行为不会扣分。
- 错题记录每个学生每道题只保留最新一条；同题再次答错时覆盖最新错误答案、测试来源和时间。
- 题目支持 2-4 个选项，默认 4 个选项；正确答案必须落在实际配置的选项范围内。
- 管理员需要具备重置能力：重置用户密码、重置学生分数、清空/重置学生测试记录。

## 2. 技术方案总览

- 后端：Go + Gin + SQLite。
- 前端：React + TypeScript + Ant Design，建议用 Vite 初始化。
- 轻量辅助库：
  - 前端：React Router、Axios、Zustand。
  - 后端：GORM、JWT、bcrypt。
- 推荐目录：
  - `backend/cmd/server/main.go`
  - `backend/internal/{config,db,models,middleware,handlers,services,routes}`
  - `frontend/src/{api,router,store,types,layouts,pages,components}`
- 认证方式：登录成功返回 JWT，前端保存 token；后端中间件解析用户和角色。
- 测试设计：使用 `test_attempts` 表代表一次测试/试卷，`test_attempt_questions` 表固化本次抽到的题目，避免后续题库变动影响历史记录。
- 得分设计：使用 `student_question_progress` 记录每个学生每题是否已经拿过分，保证跨测试重复答对不重复加分。
- 班级设计：使用 `classes` 和绑定表连接教师、学生，所有教师端学生查询都必须带班级权限过滤。

## 3. 页面/路由结构（按角色分开）

### 公共

- `/login`：统一登录页，根据角色跳转。

### 学生

- `/student`：学生首页，展示当前总积分、未完成测试入口、最近测试记录入口。
- `/student/practice`：开始固定 10 题测试；如果已有未完成测试，则提示继续该测试。
- `/student/attempts/:attemptId`：答题页，逐题提交并即时反馈。
- `/student/wrong-questions`：自己的最新错题列表和解析回顾。

### 教师

- `/teacher`：教师首页，展示自己班级学生概览和题目管理入口。
- `/teacher/students`：自己班级内学生列表，只读查看学生分数和测试情况。
- `/teacher/students/:studentId`：自己班级内学生详情、测试记录、错题。
- `/teacher/questions`：题目管理列表。
- `/teacher/questions/new`、`/teacher/questions/:questionId/edit`：题目录入/编辑，支持 2-4 个选项，默认 4 个。

### 管理员

- `/admin`：管理员首页。
- `/admin/users`：账号管理，创建/停用教师和学生账号，重置用户密码。
- `/admin/users/new`：创建账号，并选择角色及班级绑定。
- `/admin/classes`：班级管理，创建/编辑班级，维护教师与学生绑定关系。
- `/admin/students`：学生分数、测试记录、错题汇总。
- `/admin/students/:studentId`：学生详情，可重置分数、清空/重置测试记录。
- `/admin/questions`：题目管理列表。
- `/admin/questions/new`、`/admin/questions/:questionId/edit`：题目录入/编辑，支持 2-4 个选项，默认 4 个。

## 4. 后端模块与 API 设计

### 模块

- `auth`：登录、JWT、密码哈希。
- `users`：管理员创建教师/学生、用户列表、停用账号、重置密码。
- `classes`：管理员维护班级、教师绑定、学生绑定；教师端查询学生时复用班级权限校验。
- `questions`：题目录入、编辑、列表、停用；校验选项数量为 2-4，默认 4。
- `attempts`：创建测试、抽题、读取当前测试、提交答案；统一使用常量 `TEST_QUESTION_COUNT = 10`，并限制每个学生最多 1 个未完成测试。
- `student`：学生个人积分、最新错题、测试记录。
- `progress`：维护 `student_question_progress`，保证同一学生同一题只首次答对加分。
- `teacher/admin report`：学生分数、测试记录、错题查看；教师端仅限自己班级。
- `admin reset`：管理员重置密码、分数、测试记录，并同步相关进度/错题数据。

### API

公共：

- `GET /api/health`
- `POST /api/auth/login`

管理员：

- `GET /api/admin/users?role=teacher|student`
- `POST /api/admin/users`
- `PATCH /api/admin/users/:id`
- `POST /api/admin/users/:id/disable`
- `POST /api/admin/users/:id/reset-password`
- `GET /api/admin/classes`
- `POST /api/admin/classes`
- `PATCH /api/admin/classes/:id`
- `POST /api/admin/classes/:id/teachers/:teacherId`
- `DELETE /api/admin/classes/:id/teachers/:teacherId`
- `POST /api/admin/classes/:id/students/:studentId`
- `DELETE /api/admin/classes/:id/students/:studentId`
- `GET /api/admin/students`
- `GET /api/admin/students/:studentId`
- `GET /api/admin/students/:studentId/wrong-answers`
- `POST /api/admin/students/:studentId/reset-score`
- `POST /api/admin/students/:studentId/reset-attempts`

教师和管理员共享题目管理：

- `GET /api/questions`
- `POST /api/questions`
- `GET /api/questions/:id`
- `PUT /api/questions/:id`
- `POST /api/questions/:id/disable`

学生：

- `GET /api/student/me`
- `GET /api/student/attempts`
- `POST /api/student/attempts`：创建一次固定 10 题测试；如已有未完成测试则返回冲突或当前测试信息。
- `GET /api/student/attempts/current`：获取当前未完成测试，没有则返回空。
- `GET /api/student/attempts/:attemptId`
- `POST /api/student/attempts/:attemptId/answers`
- `GET /api/student/wrong-questions`

教师查看学生：

- `GET /api/teacher/students`：仅返回教师所绑定班级内学生。
- `GET /api/teacher/students/:studentId`：校验学生属于教师班级。
- `GET /api/teacher/students/:studentId/wrong-answers`：校验学生属于教师班级。

约定：所有涉及测试题数的接口都不接收客户端传入的题数，后端固定使用 `TEST_QUESTION_COUNT = 10`。

## 5. SQLite 表设计

### `users`

- `id`
- `username` 唯一
- `password_hash`
- `display_name`
- `role`：`admin` / `teacher` / `student`
- `status`：`active` / `disabled`
- `created_by`：管理员用户 ID
- `created_at`
- `updated_at`

### `classes`

- `id`
- `name` 班级名称
- `description` 可选
- `status`：`active` / `disabled`
- `created_by`：管理员用户 ID
- `created_at`
- `updated_at`

### `class_teachers`

- `class_id`
- `teacher_id`
- `created_at`
- 主键或唯一约束：`UNIQUE(class_id, teacher_id)`。

### `class_students`

- `class_id`
- `student_id`
- `created_at`
- 主键或唯一约束：`UNIQUE(class_id, student_id)`。
- MVP 可先规定一个学生只属于一个班级：额外加 `UNIQUE(student_id)`；未来如需多班级再放开。

### `questions`

- `id`
- `title` 题干
- `option_count`：2-4，默认 4
- `option_a`
- `option_b`
- `option_c`
- `option_d`：当 `option_count < 4` 时可为空
- `correct_option`：`A` / `B` / `C` / `D`，且必须在 `option_count` 范围内
- `explanation` 解析
- `difficulty` 可选
- `source` 可选
- `status`：`active` / `disabled`
- `created_by`：管理员或教师 ID
- `updated_by`
- `created_at`
- `updated_at`

说明：MVP 继续使用固定列 `option_a` 到 `option_d`，用 `option_count` 控制有效选项数量，避免引入额外选项表导致实现复杂化。

### `test_attempts`

- `id`
- `student_id`
- `question_count`：固定写入 10，由后端常量 `TEST_QUESTION_COUNT` 控制
- `status`：`in_progress` / `completed`
- `score_delta`：本次测试新增积分，只统计首次答对此前未得分题目的积分
- `started_at`
- `completed_at`
- `created_at`
- `updated_at`
- 唯一约束建议：对每个学生最多 1 个 `in_progress` 记录。SQLite 可用部分唯一索引：`UNIQUE(student_id) WHERE status='in_progress'`。

### `test_attempt_questions`

- `id`
- `attempt_id`
- `question_id`
- `question_order`：可随机也可简单递增
- `selected_option`：未答为空
- `is_correct`：未答为空
- `earned_score`：本题实际获得积分，取值 0 或 1；重复答对已得分题时为 0
- `answered_at`
- 唯一约束：`UNIQUE(attempt_id, question_id)`，保证同一次测试不重复。
- 唯一约束：`UNIQUE(attempt_id, question_order)`，保证顺序位唯一。

### `student_scores`

- `student_id` 主键
- `total_score`
- `updated_at`

说明：`total_score` 只在首次答对新题时增加；管理员重置分数时需要同步重置 `student_question_progress` 和相关测试得分统计策略。

### `student_question_progress`

- `student_id`
- `question_id`
- `has_earned_score`：是否已经因该题拿过分
- `first_correct_attempt_id`
- `first_correct_at`
- `last_answered_at`
- `last_is_correct`
- `updated_at`
- 主键或唯一约束：`UNIQUE(student_id, question_id)`。

用途：提交答案时先查该表。若本题答对且 `has_earned_score=false` 或记录不存在，则本题 `earned_score=1`，学生总分加 1，并更新为已得分；否则 `earned_score=0`。

### `wrong_answers`

- `id`
- `student_id`
- `attempt_id`
- `question_id`
- `selected_option`
- `correct_option`
- `updated_at`
- 唯一约束：`UNIQUE(student_id, question_id)`，保证每个学生每题只保留最新错题记录。

说明：答错时使用 upsert 覆盖该学生该题的最新错题；答对后是否自动移除错题可作为产品选择，MVP 建议保留最新错题直到管理员清空或后续增加“已掌握”状态。

## 6. 权限与业务规则

- `admin`：
  - 可创建、查看、停用教师和学生账号。
  - 可重置用户密码。
  - 可创建/维护班级，绑定教师和学生。
  - 可管理题目。
  - 可查看所有学生分数、测试记录、错题。
  - 可重置学生分数、清空/重置学生测试记录。
- `teacher`：
  - 可管理题目。
  - 只能查看自己绑定班级内学生的分数、测试记录、错题。
  - MVP 不能创建账号；未来可扩展为教师创建自己班级内学生。
- `student`：
  - 只能查看自己的积分、测试、错题。
  - 只能提交自己测试中的题目答案。
  - 同一时间只能有 1 个未完成测试。
- 创建测试时：
  - 每次固定抽取 10 道题，不提供学生自选题数。
  - 如果学生已有 `in_progress` 测试，则不创建新测试，返回当前未完成测试或明确错误。
  - 从 `questions.status='active'` 中随机选择 10 道。
  - 后端校验可用题数必须 >= 10。
  - 抽题结果写入 `test_attempt_questions`；后续答题只从该表读取。
  - 数据库唯一约束 `UNIQUE(attempt_id, question_id)` 是最终防线。
- 提交答案时：
  - 校验题目属于该学生的当前测试。
  - 校验所选答案在该题有效选项范围内。
  - 已答题目不可重复提交，除非产品未来明确允许重答。
  - 正确时检查 `student_question_progress`：首次答对该题才 `earned_score=1`，并更新 `score_delta`、`student_scores.total_score`；重复答对该题 `earned_score=0`。
  - 错误时 `earned_score=0`，并 upsert `wrong_answers`，只保留该学生该题最新错误记录。
  - 每次提交都更新 `student_question_progress.last_answered_at` 和 `last_is_correct`。
  - 返回字段建议包括：`isCorrect`、`earnedScore`、`currentTotalScore`、`correctOption`、`explanation`。
- 完成测试时：
  - 当 10 道题全部作答后，将 `test_attempts.status` 更新为 `completed` 并写入 `completed_at`。
- 管理员重置时：
  - 重置密码：生成/设置新密码，更新 `users.password_hash`。
  - 重置分数：将 `student_scores.total_score` 归零，并清理或重置 `student_question_progress.has_earned_score`，避免旧进度阻止后续重新得分。
  - 重置测试记录：按产品操作清空该学生 `test_attempts/test_attempt_questions`，必要时同步清理 `wrong_answers` 和进度表；MVP 建议提供“清空测试记录并清空错题/进度”的明确操作。

## 7. 第一阶段骨架怎么搭

1. 初始化后端 Gin 项目和前端 Vite React TS 项目。
2. 后端接入 SQLite、GORM、基础迁移、`GET /api/health`。
3. 建立 `users/classes/class_teachers/class_students/questions/test_attempts/test_attempt_questions/student_scores/student_question_progress/wrong_answers` 模型。
4. 加 seed 数据：1 个 admin、1 个 teacher、1 个 class、2 个 student、教师和学生班级绑定、若干题目。
5. 实现登录、JWT 中间件、角色中间件。
6. 实现班级权限校验工具：教师访问学生数据前必须验证学生属于教师绑定班级。
7. 前端搭建登录页、三类角色 Layout、路由守卫。
8. 实现学生首页、创建/继续固定 10 题测试、答题提交和即时反馈闭环。
9. 实现答题计分逻辑：首次答对加分、重复答对不加分、错题 upsert 最新记录。
10. 实现管理员创建教师/学生账号、班级绑定、密码重置。
11. 实现管理员重置学生分数和测试记录。
12. 实现教师/管理员题目管理基础 CRUD，支持 2-4 个选项。
13. 实现教师/管理员查看学生分数和错题，其中教师端只展示自己班级学生。

## 8. 后续迭代顺序

1. 完成 MVP 闭环：登录、学生固定 10 题测试、即时反馈、首次答对计分、最新错题记录。
2. 完成管理员账号管理、班级管理、教师/学生绑定。
3. 完成管理员重置密码、重置分数、重置测试记录。
4. 完成教师/管理员题目管理，支持 2-4 个选项和题目停用。
5. 完成教师/管理员学生分数和错题查看，并确保教师端班级隔离。
6. 增加题目分类、难度、来源、批量导入。
7. 增加错题重练、掌握状态、统计图表、导出成绩。
8. 增加更细权限、审计日志、重置操作记录。

## 9. 仍需确认的问题

- 一个学生未来是否可能属于多个班级？MVP 暂定一个学生只属于一个班级。
- 学生答错后，后续同题答对时是否应从错题列表自动移除，还是保留最新错题并增加“已掌握”状态？MVP 暂定保留最新错题。
- 管理员“重置测试记录”是否需要拆成多个独立操作：仅删除未完成测试、清空全部测试、同时清空错题/进度？MVP 建议先提供一个明确的全量重置操作。
- 重置分数后，历史测试中的 `earned_score` 是否保留作为历史事实，还是同步归零？MVP 建议保留历史答题事实，但清空当前总分和得分进度，让学生后续可重新得分。

## 10. 里程碑拆分

### Milestone 1：项目骨架与基础设施
目标：搭建可运行的前后端工程、数据库与基础开发流程。
范围：
- 初始化 Go + Gin API、React + TypeScript + Ant Design 前端工程
- 配置 SQLite 连接、迁移机制与基础目录结构
- 建立本地启动脚本、环境配置与基础健康检查接口
验收：
- 前后端可分别启动并访问健康检查页面/接口
- 数据库文件可自动创建并执行首批迁移
- README 或现有计划中记录本地运行命令
依赖：无，后续所有里程碑依赖本里程碑。

### Milestone 2：账号、角色与权限体系
目标：实现 admin / teacher / student 三类角色的认证、授权与账号管理基础能力。
范围：
- 实现登录、登出、会话/JWT 校验与路由守卫
- 实现 admin 创建 teacher/student 账号
- 实现 teacher 仅可查看并管理自己班级学生的权限边界
- 实现 admin 重置用户密码能力
验收：
- 三类角色登录后进入各自可访问页面
- 非授权角色访问接口与页面会被拒绝
- admin 可创建账号并重置密码
- teacher 无法查看其他班级学生
依赖：依赖 Milestone 1。

### Milestone 3：题库与选项管理
目标：实现 teacher + admin 可用的文言文选择题题库管理。
范围：
- 设计题目、选项、正确答案等数据表与 API
- 支持题目增删改查，并限制选项数量为 2-4 个
- 新建题目默认生成 4 个选项编辑位
- 前端实现题库列表、编辑表单与权限控制
验收：
- admin 与 teacher 可管理题目
- student 无法进入题库管理功能
- 少于 2 个或多于 4 个选项的题目无法保存
- 默认 4 选项行为符合计划规则
依赖：依赖 Milestone 2；可与 Milestone 4 的前端页面细化部分并行。

### Milestone 4：测试流程与答题约束
目标：实现学生每次固定 10 题、且同一时间仅允许一个未完成测试的答题流程。
范围：
- 设计 test、test_question、answer 相关数据模型
- 实现创建测试时固定抽取/分配 10 道题
- 实现未完成测试续答，禁止重复创建第二个未完成测试
- 前端实现开始测试、继续测试、提交答案与完成测试流程
验收：
- 每次测试题目数固定为 10
- 存在未完成测试时再次开始会进入原测试
- 完成测试后才允许创建下一次测试
- 刷新页面后答题进度仍可恢复
依赖：依赖 Milestone 3 的题库数据；可与 Milestone 5 的统计接口设计并行。

### Milestone 5：评分与错题记录规则
目标：实现跨测试的首次答对计分与最新错题记录规则。
范围：
- 实现“同一学生同一题仅首次答对计分”的评分逻辑
- 维护 student/question 粒度的最新错题记录，仅保留最新一次错误答案
- 答对后按计划规则更新或清理相关错题状态
- 提供学生得分、错题列表与测试结果接口
验收：
- 同一题多次答对只增加一次分数
- 同一学生同一题多次答错只保留最新错误记录
- 测试结果、总分与错题列表数据一致
- 评分逻辑有可重复验证的后端测试或脚本验证
依赖：依赖 Milestone 4；统计展示页面可与 Milestone 6 并行。

### Milestone 6：管理端统计与重置能力
目标：完成 admin/teacher 面向班级、学生、题目与测试记录的管理闭环。
范围：
- teacher 查看自己班级学生成绩、测试记录与错题概览
- admin 查看全局账号、成绩、题目与测试记录
- admin 实现重置密码、重置分数、清空/重置测试记录
- 前端补齐管理端列表、筛选、确认弹窗与操作反馈
验收：
- teacher 只能看到自己班级范围内统计数据
- admin 可执行密码、分数、测试记录重置
- 重置后学生得分、测试状态、错题记录符合源规则
- 高风险操作具备确认提示并返回明确结果
依赖：依赖 Milestone 5；部分列表 UI 可在 Milestone 5 后期并行开发。

### Milestone 7：联调、验收与发布准备
目标：完成端到端联调、规则回归验证与可部署交付准备。
范围：
- 覆盖角色权限、题库、10 题测试、单未完成测试、评分、错题、重置流程
- 补齐必要的后端单元/集成测试与前端关键流程自测
- 统一错误提示、空状态、加载态与表单校验体验
- 整理部署配置、数据库初始化与管理员初始账号方案
验收：
- 核心规则均有手工验收清单或自动化测试覆盖
- 前后端构建通过，关键流程端到端可跑通
- 新环境可按文档初始化并登录 admin
- 无阻塞级权限、评分或数据一致性问题
依赖：依赖 Milestone 2-6 完成；只能在主要功能闭环后收尾。
