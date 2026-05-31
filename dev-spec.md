# 文言文选择题系统 Dev Spec

## 1. 项目目标

- 构建一个面向学生的文言文选择题网页应用。
- 学生可以在线完成测试、即时获得答题反馈，并查看自己的累计积分与错题。
- 教师可以管理题目，并查看自己班级学生的成绩、测试记录与错题情况。
- 管理员可以创建教师/学生账号、维护班级关系，并拥有全局管理与重置能力。

## 2. 技术栈

- 后端：Go + Gin + SQLite
- 前端：React + TypeScript + Ant Design
- 认证：JWT + bcrypt
- 开发工具：
  - 前端使用 Vite
  - 本地 Go 工具链位于 `./.tools/go`
  - 根目录 `Makefile` 统一封装常用命令

## 3. 当前实现基线

- 已完成：
  - M1 项目骨架与基础设施
  - M2 账号、角色与权限体系
  - M3 题库与选项管理
  - M4 测试流程与答题约束
  - M5 评分与错题记录规则
  - M6 管理端统计与重置能力
  - M7 联调、验收与发布准备
- 当前状态：
  - 所有既定 milestones 已完成并通过 supervisor 验收

## 4. 角色与权限

| 角色 | 核心权限 |
| --- | --- |
| admin | 创建教师/学生账号、管理班级、管理题目、查看全局学生数据、重置密码/分数/测试记录 |
| teacher | 管理题目、查看自己班级学生的成绩/测试/错题 |
| student | 开始或继续测试、提交答案、查看个人积分与错题 |

## 5. 核心业务规则

- 角色固定为 `admin`、`teacher`、`student`。
- 学生和教师账号统一由管理员创建。
- 管理员创建班级，并将教师、学生绑定到班级。
- 教师只能看到自己班级内的学生数据。
- 管理员和教师都可以录入、编辑、停用题目。
- 题目支持 2-4 个选项，默认 4 个选项。
- 学生每次测试固定为 10 道题。
- 同一次测试内题目不允许重复。
- 学生同一时间只能有 1 个未完成测试。
- 学生提交每道题后立即收到反馈：
  - 是否正确
  - 本题新增得分
  - 正确答案
  - 解析
- 同一学生同一题最多只加 1 分：
  - 首次答对该题 +1
  - 后续再次答对该题 +0
- 积分只增不减，除管理员重置外不会扣分。
- 每个学生每道题只保留最新一条错题记录。
- MVP 规则：学生后续答对同一题后，历史最新错题记录默认保留，直到管理员清空或未来引入“已掌握”状态。
- 管理员“重置分数”会清零累计积分与题目得分进度，但保留历史测试记录与错题记录。
- 管理员“重置测试记录”会清空测试、作答、错题与得分进度，并将累计积分归零。

## 6. 页面与路由

### 公共

- `/login`：统一登录页

### 学生

- `/student`：学生首页，显示累计积分、当前测试、错题回顾
- `/student/attempts/:attemptId`：答题页，逐题提交并即时反馈
- 规划中：`/student/wrong-questions` 独立错题页（当前可先在首页展示）

### 教师

- `/teacher`：教师学生页入口
- `/teacher/students`：自己班级学生列表，可查看学生成绩、测试记录与错题详情
- `/teacher/questions`：题目管理

### 管理员

- `/admin`：管理员首页
- `/admin/users`：账号管理
- `/admin/classes`：班级管理
- `/admin/questions`：题目管理
- `/admin/reports`：全局学生报告、错题查看、分数重置与测试记录重置

## 7. 后端模块

- `auth`：登录、JWT、当前用户解析
- `users/admin`：管理员账号管理、密码重置
- `classes`：班级创建与师生绑定
- `questions`：题目录入、编辑、停用、选项数量校验
- `attempts`：创建测试、抽题、续答、提交答案、完成测试
- `student`：学生个人积分、当前测试、错题列表
- `progress`：维护 `student_question_progress`，保证同题只首次答对加分
- `reports/reset`：教师/管理员学生报告、错题查看，以及管理员分数/测试记录重置

## 8. 主要接口

### 已实现接口

- 公共：
  - `GET /api/health`
  - `POST /api/auth/login`
- 鉴权：
  - `GET /api/auth/me`
  - `POST /api/auth/logout`
- 题库：
  - `GET /api/questions`
  - `POST /api/questions`
  - `GET /api/questions/:id`
  - `PATCH /api/questions/:id`
  - `POST /api/questions/:id/disable`
- 管理员：
  - `GET /api/admin/users`
  - `POST /api/admin/users`
  - `POST /api/admin/users/:id/disable`
  - `POST /api/admin/users/:id/reset-password`
  - `GET /api/admin/students`
  - `GET /api/admin/students/:studentId`
  - `GET /api/admin/students/:studentId/wrong-answers`
  - `POST /api/admin/students/:studentId/reset-score`
  - `POST /api/admin/students/:studentId/reset-attempts`
  - `GET /api/admin/classes`
  - `POST /api/admin/classes`
  - `PATCH /api/admin/classes/:id`
  - `POST /api/admin/classes/:id/teachers/:teacherId`
  - `DELETE /api/admin/classes/:id/teachers/:teacherId`
  - `POST /api/admin/classes/:id/students/:studentId`
  - `DELETE /api/admin/classes/:id/students/:studentId`
- 教师：
  - `GET /api/teacher/students`
  - `GET /api/teacher/students/:studentId`
  - `GET /api/teacher/students/:studentId/wrong-answers`
- 学生：
  - `GET /api/student/score`
  - `GET /api/student/wrong-answers`
  - `POST /api/student/attempts`
  - `GET /api/student/attempts/current`
  - `GET /api/student/attempts/:attemptId`
  - `POST /api/student/attempts/:attemptId/answers`

### 预留扩展

- M7 以联调、回归与发布准备为主，当前没有新的必需业务接口。

## 8.1 M7 完成内容

- 后端新增回归测试，覆盖：
  - 固定 10 题且同次测试不重复
  - 单个未完成测试复用
  - 同题仅首次答对加分
  - 错题记录只保留最新错误答案
  - 管理员重置测试记录后清空学生状态
- 前端关键页面补齐加载失败、空状态、提交失败和重试反馈：
  - 管理员账号页
  - 管理员班级页
  - 题库管理页
  - 学生首页与答题页
  - 管理员学生报告页
  - 教师学生页

## 9. 数据模型

### 当前主要表

- `users`：用户账号
- `classes`：班级
- `class_teachers`：班级-教师绑定
- `class_students`：班级-学生绑定
- `questions`：题库
- `attempts`：一次测试
- `attempt_questions`：一次测试中被抽到的题目快照
- `attempt_answers`：学生提交的答案记录
- `student_scores`：学生累计积分
- `student_question_progress`：学生在题目维度的得分进度
- `wrong_answers`：每个学生每题最新一条错题记录

### 关键约束

- `attempts`：每个学生最多 1 个 `in_progress` 测试
- `attempt_questions`：
  - 同一次测试中题目不可重复
  - 固化题干、选项、正确答案、解析，避免题库修改影响历史记录
- `attempt_answers`：
  - 每个 `attempt_question` 只能答一次
  - `score_delta` 取值为 `0` 或 `1`
- `student_question_progress`：
  - `UNIQUE(student_id, question_id)`
  - 记录是否已因该题拿过分
- `wrong_answers`：
  - `UNIQUE(student_id, question_id)`
  - 仅保留该学生该题最新一次错误答案

## 10. 本地开发

### 常用命令

- `make backend`
- `make frontend`
- `make test-backend`
- `make build-backend`
- `make build-frontend`

### 本地默认账号

- `admin / admin123`
- `teacher1 / teacher123`
- `student1 / student123`

## 11. 交付结论

- 当前 milestone 计划已全部完成。
- 交付基线验证命令：
  - `make test-backend`
  - `make build-backend`
  - `make build-frontend`
- 新环境按 `README.md` 初始化后，可使用内置管理员账号 `admin / admin123` 登录。
