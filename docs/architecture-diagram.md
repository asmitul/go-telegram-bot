# 架构流程图

## 系统整体架构

```mermaid
graph TB
    Start([Telegram 消息]) --> Bot[Telegram Bot API]
    Bot --> DefaultHandler[Default Handler]
    DefaultHandler --> Convert[转换为 Handler Context]
    Convert --> Router[Router.Route]

    Router --> MatchLoop{遍历处理器<br/>按优先级}

    MatchLoop --> |Priority 100| Commands[命令处理器]
    MatchLoop --> |Priority 200| Keywords[关键词处理器]
    MatchLoop --> |Priority 300| Patterns[正则处理器]
    MatchLoop --> |Priority 900+| Listeners[监听器]

    Commands --> CmdMatch{Match?}
    Keywords --> KwMatch{Match?}
    Patterns --> PatMatch{Match?}
    Listeners --> ListMatch{Match?}

    CmdMatch --> |Yes| CmdChain[中间件链]
    KwMatch --> |Yes| KwChain[中间件链]
    PatMatch --> |Yes| PatChain[中间件链]
    ListMatch --> |Yes| ListChain[中间件链]

    CmdChain --> CmdHandle[Handle 处理]
    KwChain --> KwHandle[Handle 处理]
    PatChain --> PatHandle[Handle 处理]
    ListChain --> ListHandle[Handle 处理]

    CmdHandle --> CmdCont{Continue<br/>Chain?}
    KwHandle --> KwCont{Continue<br/>Chain?}
    PatHandle --> PatCont{Continue<br/>Chain?}
    ListHandle --> ListCont{Continue<br/>Chain?}

    CmdMatch --> |No| MatchLoop
    KwMatch --> |No| MatchLoop
    PatMatch --> |No| MatchLoop

    CmdCont --> |No| End([处理完成])
    KwCont --> |Yes| MatchLoop
    PatCont --> |Yes| MatchLoop
    ListCont --> |Yes| MatchLoop

    ListMatch --> |No| End

    style Commands fill:#90EE90
    style Keywords fill:#87CEEB
    style Patterns fill:#FFB6C1
    style Listeners fill:#FFD700
```

## 已实现的命令处理器（Priority: 100）

```mermaid
graph LR
    subgraph Commands[命令处理器]
        direction TB
        Ping["/ping<br/>🏓 测试响应速度"]
        Help["/help<br/>📖 显示帮助信息"]
        Stats["/stats<br/>📊 显示统计信息"]

        Ping --> |权限: User| PingResp["返回: Pong + 延迟"]
        Help --> |权限: User| HelpResp["返回: 命令列表"]
        Stats --> |权限: User| StatsResp["返回: 用户/群组统计"]
    end

    style Ping fill:#90EE90
    style Help fill:#90EE90
    style Stats fill:#90EE90
```

## 已实现的关键词处理器（Priority: 200）

```mermaid
graph LR
    subgraph Keywords[关键词处理器]
        direction TB
        Greeting["Greeting Handler<br/>关键词: 你好/hello/hi/嗨"]

        Greeting --> |匹配| GreetResp["返回: 问候语<br/>私聊/群组"]
        Greeting --> |支持| ChatTypes["✅ private<br/>✅ group<br/>✅ supergroup"]
    end

    style Greeting fill:#87CEEB
```

## 已实现的正则处理器（Priority: 300）

```mermaid
graph LR
    subgraph Patterns[正则匹配处理器]
        direction TB
        Weather["Weather Handler<br/>正则: (?i)天气\\s+(.+)"]

        Weather --> |提取| City["城市名称"]
        City --> |模拟| WeatherResp["返回: 天气信息<br/>（模拟数据）"]
    end

    style Weather fill:#FFB6C1
```

## 已实现的监听器（Priority: 900+）

```mermaid
graph LR
    subgraph Listeners[监听器]
        direction TB
        Logger["MessageLogger<br/>Priority: 900<br/>记录所有消息"]
        Analytics["Analytics<br/>Priority: 950<br/>分析统计"]

        Logger --> |记录| LogFields["user_id<br/>chat_id<br/>chat_type<br/>text<br/>username"]

        Analytics --> |统计| AnalFields["消息总数<br/>用户活跃度<br/>群组活跃度"]

        Logger --> Continue1["ContinueChain: true"]
        Analytics --> Continue2["ContinueChain: true"]
    end

    style Logger fill:#FFD700
    style Analytics fill:#FFD700
```

## 中间件执行流程（洋葱模型）

```mermaid
graph TB
    Start([请求开始]) --> Recovery[Recovery Middleware<br/>捕获 Panic]
    Recovery --> RecoveryIn[捕获异常]

    RecoveryIn --> Logging[Logging Middleware<br/>记录日志]
    Logging --> LogIn[记录请求信息]

    LogIn --> Permission[Permission Middleware<br/>加载用户权限]
    Permission --> PermIn[从数据库加载 User]

    PermIn --> Handler[Handler.Handle<br/>业务逻辑]

    Handler --> PermOut[权限检查完成]
    PermOut --> LogOut[记录响应]
    LogOut --> RecoveryOut[错误处理]
    RecoveryOut --> End([请求完成])

    style Recovery fill:#FF6B6B
    style Logging fill:#4ECDC4
    style Permission fill:#95E1D3
    style Handler fill:#F38181
```

## 定时任务系统

```mermaid
graph TB
    Scheduler[Scheduler 调度器] --> |每天 1d| Cleanup[CleanupExpiredData<br/>清理过期数据]
    Scheduler --> |每小时 1h| Stats[StatisticsReport<br/>统计报告]

    Cleanup --> CleanWarn[清理 90 天前的警告记录]
    Cleanup --> CleanUser[清理 180 天未活跃用户]

    Stats --> StatLog[记录统计信息<br/>用户数/群组数]

    subgraph "未启用的任务"
        Scheduler -.-> |每 5 分钟| AutoUnban[AutoUnban<br/>自动解封]
        Scheduler -.-> |每 30 分钟| CacheWarmup[CacheWarmup<br/>缓存预热]
    end

    style Cleanup fill:#90EE90
    style Stats fill:#87CEEB
    style AutoUnban fill:#D3D3D3
    style CacheWarmup fill:#D3D3D3
```

## 数据持久化架构

```mermaid
graph LR
    subgraph Domain[Domain Layer]
        User[User 用户实体]
        Group[Group 群组实体]
    end

    subgraph Repository[Repository Layer]
        UserRepo[UserRepository<br/>用户仓储]
        GroupRepo[GroupRepository<br/>群组仓储]
    end

    subgraph Database[MongoDB]
        UsersCol[(users 集合)]
        GroupsCol[(groups 集合)]
        WarningsCol[(warnings 集合)]
        BansCol[(bans 集合)]
    end

    User -.实现.-> UserRepo
    Group -.实现.-> GroupRepo

    UserRepo --> UsersCol
    GroupRepo --> GroupsCol

    UsersCol --> |索引| UserIdx["user_id: 1<br/>username: 1"]
    GroupsCol --> |索引| GroupIdx["group_id: 1"]

    style User fill:#FFB6C1
    style Group fill:#FFB6C1
    style UserRepo fill:#87CEEB
    style GroupRepo fill:#87CEEB
```

## 启动与关闭流程

```mermaid
graph TB
    Start([启动]) --> LoadEnv[加载 .env 配置]
    LoadEnv --> InitLogger[初始化 Logger]
    InitLogger --> ConnMongo[连接 MongoDB]
    ConnMongo --> CreateIndex[创建数据库索引]
    CreateIndex --> InitRepo[初始化 Repository]

    InitRepo --> CreateRouter[创建 Router]
    CreateRouter --> RegMiddleware[注册中间件<br/>Recovery, Logging, Permission]
    RegMiddleware --> RegHandlers[注册处理器<br/>3 命令 + 1 关键词 + 1 正则 + 2 监听器]

    RegHandlers --> InitBot[初始化 Telegram Bot]
    InitBot --> InitScheduler[初始化 Scheduler<br/>2 个定时任务]

    InitScheduler --> StartBot[启动 Bot]
    StartBot --> StartScheduler[启动 Scheduler]
    StartScheduler --> WaitSignal[等待退出信号<br/>SIGINT/SIGTERM]

    WaitSignal --> Shutdown[优雅关闭]

    Shutdown --> StopBot[停止接收消息]
    StopBot --> StopScheduler[停止 Scheduler]
    StopScheduler --> WaitMsg[等待处理中消息<br/>最多 30 秒]
    WaitMsg --> CloseMongo[关闭 MongoDB]
    CloseMongo --> LogStats[输出运行统计]
    LogStats --> Exit([退出])

    style Start fill:#90EE90
    style Exit fill:#FF6B6B
    style Shutdown fill:#FFD700
```

## 权限系统

```mermaid
graph TB
    subgraph Permission[权限等级]
        Owner[Owner 所有者<br/>Level: 4]
        SuperAdmin[SuperAdmin 超级管理员<br/>Level: 3]
        Admin[Admin 管理员<br/>Level: 2]
        User[User 普通用户<br/>Level: 1]
        None[None 无权限<br/>Level: 0]
    end

    Owner --> |可管理| SuperAdmin
    SuperAdmin --> |可管理| Admin
    Admin --> |可管理| User
    User --> |可管理| None

    subgraph PermCheck[权限检查]
        HasPerm[HasPermission<br/>检查权限]
        RequirePerm[RequirePermission<br/>要求权限]

        HasPerm --> |返回| Bool[true/false]
        RequirePerm --> |不足时| Error[返回错误信息]
    end

    subgraph PerGroup[按群组权限]
        UserPerms[User.Permissions<br/>map[groupID]Permission]

        UserPerms --> |私聊| UserID[使用 userID 作为 key]
        UserPerms --> |群组| GroupID[使用 chatID 作为 key]
    end

    style Owner fill:#FF6B6B
    style SuperAdmin fill:#FF8C42
    style Admin fill:#FFD166
    style User fill:#06FFA5
    style None fill:#D3D3D3
```

## 功能统计总览

```mermaid
pie title 已实现的处理器分布
    "命令处理器" : 3
    "关键词处理器" : 1
    "正则处理器" : 1
    "监听器" : 2
```

## 支持的聊天类型

```mermaid
graph LR
    subgraph ChatTypes[支持的聊天类型]
        Private[Private<br/>私聊]
        Group[Group<br/>普通群组]
        SuperGroup[SuperGroup<br/>超级群组]
        Channel[Channel<br/>频道]
    end

    Private --> |✅| AllHandlers[所有处理器都支持]
    Group --> |✅| AllHandlers
    SuperGroup --> |✅| AllHandlers
    Channel --> |⚠️| LimitedSupport[部分支持<br/>取决于处理器配置]

    style Private fill:#90EE90
    style Group fill:#90EE90
    style SuperGroup fill:#90EE90
    style Channel fill:#FFD700
```

---

## 图例说明

| 颜色 | 说明 |
|-----|------|
| 🟢 绿色 | 命令处理器 |
| 🔵 蓝色 | 关键词/数据层 |
| 🟣 粉色 | 正则处理器/领域层 |
| 🟡 黄色 | 监听器/警告 |
| ⚪ 灰色 | 未启用功能 |
| 🔴 红色 | 关键节点/错误处理 |

---

## 快速功能索引

### ✅ 已实现功能

**命令（3 个）**:
- `/ping` - 测试 Bot 响应
- `/help` - 显示帮助信息
- `/stats` - 显示统计数据

**关键词（1 个）**:
- 问候语检测（你好/hello/hi/嗨）

**正则匹配（1 个）**:
- 天气查询（天气 + 城市名）

**监听器（2 个）**:
- MessageLogger - 消息日志记录
- Analytics - 数据分析统计

**中间件（3 个）**:
- Recovery - Panic 恢复
- Logging - 日志记录
- Permission - 权限加载

**定时任务（2 个启用）**:
- CleanupExpiredData - 清理过期数据（每天）
- StatisticsReport - 统计报告（每小时）

**数据库集合（4 个）**:
- users - 用户信息
- groups - 群组信息
- warnings - 警告记录
- bans - 封禁记录

### 🔧 配置的但未启用

**定时任务（2 个）**:
- AutoUnban - 自动解封（每 5 分钟）
- CacheWarmup - 缓存预热（每 30 分钟）

**中间件（1 个）**:
- RateLimit - 限流中间件（已实现但未注册）

---

**更新日期**: 2025-10-02
**架构版本**: v2.0.0
**维护者**: Telegram Bot Development Team
