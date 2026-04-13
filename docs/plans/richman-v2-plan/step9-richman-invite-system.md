# Step 9: richman Invite System

> Phase 3 | 并行组 R5 (可与 Step 8, 10 同时执行) | 前置: Step 7

## 任务目标

实现完整邀请裂变系统：InviteService（邀请码生成/消费/查询 + 连续登录解锁 + 双向奖励），AuthService 集成（注册消费专属码 + 登录更新 streak），分享链接附带邀请码。同时处理邀请系统全部已知问题。

## 涉及文件

### 创建

- `backend/internal/service/invite/service.go` -- InviteService 全部方法

### 修改

- `backend/internal/service/auth/` -- Register 增加 disclaimerAccepted 校验 + InviteService 集成, Login 增加 UpdateLoginStreak

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| InviteService 全部方法 | SS14.3 邀请裂变 | invite SS8.1 |
| 邀请码生成 (crypto/rand, RM+8位) | SS14.3 | invite SS4.1 |
| 注册消费专属码 (原子操作) | SS14.3 | invite SS4.2 |
| 双向奖励 | SS14.3 | invite SS4.3 |
| 连续登录解锁 (原子 SQL) | SS14.3 | invite SS4.4 |
| AuthService.Register 集成 | SS14.3 | invite SS8.2 |
| AuthService.Login 集成 | SS14.3 | invite SS8.2 |
| disclaimerAccepted 校验 | SS13 免责声明 | richman SS15.2.1 |
| 分享链接 GetFirstAvailableCode | SS14.3 | invite SS6.1 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G2.8 暴力破解防护 | 邀请码验证失败计数 + 锁定 | richman SS22.8 |
| G2.10 悬空引用 | hard delete 流程先清理 invite 关联 | richman SS22.10 |
| G4.1 streak 无上限 | 邀请码总数上限 20 + 未使用码 <3 才生成 | invite SS11.1 |
| G4.2 时区边界 | PostgreSQL timezone = Asia/Shanghai | invite SS11.2 |
| G4.3 悬空引用 | used_by_user_id NULL 处理 | invite SS11.3 |

- 邀请码使用 crypto/rand 生成，UNIQUE 冲突重试最多 3 次
- UseInviteCode 使用 `UPDATE ... WHERE is_used = FALSE RETURNING *` 原子操作防并发
- 注册事务中：创建用户 -> 消费码 -> 双向奖励 -> 生成 3 码，任一失败回滚
- login_streak 更新使用原子 SQL（CASE WHEN），不在应用层读-改-写
- `login_streak % 7 == 0` 且当前未使用码 <3 且总码数 <20 时才生成新码

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] InviteService 6 个方法全部可调用
- [ ] GenerateCodesForUser 生成指定数量码，格式 RM+8 位
- [ ] UseInviteCode 并发测试：两个 goroutine 同时消费同一码，只有一个成功
- [ ] 注册流程集成：注册后用户有 3 个邀请码
- [ ] login_streak 原子 SQL 在连续/断连/同日重复三种场景下输出正确
- [ ] 邀请码总数上限 20 约束生效

## 变更点清单覆盖

D3.12-D3.17 (6), D4.4-D4.6 (3), G2.8 (1), G2.10 (1), G4.1-G4.3 (3) = **14 项**
