# Go 项目结构体依赖分析报告

**项目路径**: /Users/albert/Desktop/fromGithub/anal_go_agent/testdata/sample_project
**分析起点**: UserService
**分析深度**: 2
**生成时间**: 2026-01-19 22:02:11

---

## 分析概览

- **总结构体数**: 6
- **分析深度分布**:
  - 深度 0: 1 个
  - 深度 1: 3 个
  - 深度 2: 2 个
- **总依赖关系数**: 9
- **循环依赖**: 0 个

---

## 深度 0

### UserService

**功能**: 用户服务，处理用户数据及缓存管理

**所属包**: `service`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| repo | \*repository.UserRepository | ✗ | 负责用户数据持久化 |
| cache | \*cache.Cache | ✗ | 提供用户数据缓存 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| CreateUser | (name string, age int) error | ✓ | 创建新用户 |
| GetUserByID | (id int64) (\*model.User, error) | ✓ | 根据ID查询用户 |
| UpdateUser | (user \*model.User) error | ✓ | 更新用户信息 |
| DeleteUser | (id int64) error | ✓ | 删除用户 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| UserRepository | 字段依赖 | repo 字段 | 1 |
| Cache | 字段依赖 | cache 字段 | 1 |
| User | 方法内初始化 | CreateUser 方法 | 1 |
| User | 方法调用 | CreateUser -> Validate | 1 |

---

## 深度 1

### UserRepository

**功能**: 用户数据仓库，负责数据持久化

**所属包**: `repository`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| db | \*Database | ✗ | 数据库操作接口 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Save | (user model.User) error | ✓ | 保存用户到数据库 |
| FindByID | (id int64) (\*model.User, error) | ✓ | 根据ID查找用户 |
| Update | (user \*model.User) error | ✓ | 更新用户信息 |
| Delete | (id int64) error | ✓ | 删除用户 |
| FindAll | () ([]\*model.User, error) | ✓ | 查找所有用户 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| Database | 字段依赖 | db 字段 | 2 |

---

### Cache

**功能**: 实现本地与Redis的双级缓存服务

**所属包**: `cache`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| client | \*RedisClient | ✗ | Redis客户端连接 |
| ttl | time.Duration | ✗ | 缓存数据过期时间 |
| mu | sync.RWMutex | ✗ | 并发安全读写锁 |
| local | map[int64]interface{} | ✗ | 本地内存数据存储 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Get | (key int64) interface{} | ✓ | 获取缓存数据 |
| Set | (key int64, value interface{}) | ✓ | 设置缓存数据 |
| Delete | (key int64) | ✓ | 删除缓存数据 |
| Clear | () | ✓ | 清空所有缓存 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| RedisClient | 字段依赖 | client 字段 | 2 |

---

### User

**功能**: 用户实体，定义基本属性与元数据

**所属包**: `model`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| ID | int64 | ✓ | 用户唯一标识 |
| Name | string | ✓ | 用户姓名 |
| Age | int | ✓ | 用户年龄 |
| Email | string | ✓ | 电子邮箱地址 |
| CreatedAt | time.Time | ✓ | 记录创建时间 |
| UpdatedAt | time.Time | ✓ | 记录更新时间 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Validate | () error | ✓ | 校验数据合法性 |
| IsAdult | () bool | ✓ | 判断是否成年 |
| UpdateTimestamp | () | ✓ | 更新修改时间 |

---

## 深度 2

### Database

**功能**: 数据库操作封装，提供基础增删改查功能

**所属包**: `repository`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| connectionString | string | ✗ | 数据库连接地址 |
| pool | \*ConnectionPool | ✗ | 连接池管理实例 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Insert | (table string, data interface{}) error | ✓ | 向指定表插入数据 |
| QueryByID | (table string, id int64) (interface{}, error) | ✓ | 根据ID查询单条数据 |
| QueryAll | (table string) ([]interface{}, error) | ✓ | 查询表中所有数据 |
| Update | (table string, id int64, data interface{}) error | ✓ | 根据ID更新指定数据 |
| DeleteByID | (table string, id int64) error | ✓ | 根据ID删除指定数据 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| ConnectionPool | 字段依赖 | pool 字段 | 3 |
| User | 方法内初始化 | QueryByID 方法 | 3 |

---

### RedisClient

**功能**: Redis 客户端封装，提供缓存数据的存取与操作

**所属包**: `cache`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| address | string | ✗ | Redis 服务器地址 |
| password | string | ✗ | 连接认证密码 |
| db | int | ✗ | 数据库索引 |
| pool | \*RedisPool | ✗ | Redis 连接池 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Get | (key int64) interface{} | ✓ | 获取键对应的值 |
| Set | (key int64, value interface{}, ttl time.Duration) | ✓ | 设置键值及过期时间 |
| Delete | (key int64) | ✓ | 删除指定键数据 |
| FlushAll | () | ✓ | 清空所有数据 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| RedisPool | 字段依赖 | pool 字段 | 3 |

---

## 依赖关系图

```mermaid
graph TD
    UserService["UserService<br/>用户服务，处理用户数据及缓存管..."]
    UserRepository["UserRepository<br/>用户数据仓库，负责数据持久化"]
    Cache["Cache<br/>实现本地与Redis的双级缓存..."]
    User["User<br/>用户实体，定义基本属性与元数据"]
    Database["Database<br/>数据库操作封装，提供基础增删改..."]
    RedisClient["RedisClient<br/>Redis 客户端封装，提供缓..."]

    UserService -->|字段| UserRepository
    UserService -->|字段| Cache
    UserService -->|初始化| User
    UserRepository -->|字段| Database
    Cache -->|字段| RedisClient
    Database -->|字段| ConnectionPool
    Database -->|初始化| User
    RedisClient -->|字段| RedisPool

    style UserService fill:#ff9999
    style UserRepository fill:#99ccff
    style Cache fill:#99ccff
    style User fill:#99ccff
    style Database fill:#99ff99
    style RedisClient fill:#99ff99
```

---

## 统计信息

### 依赖深度分布
- 深度 0: 1 个结构体
- 深度 1: 3 个结构体
- 深度 2: 2 个结构体

### 被依赖次数排行
1. User - 被依赖 3 次
2. Database - 被依赖 1 次
3. RedisClient - 被依赖 1 次
4. ConnectionPool - 被依赖 1 次
5. RedisPool - 被依赖 1 次
6. UserRepository - 被依赖 1 次
7. Cache - 被依赖 1 次

---

生成于: 2026-01-19 22:02:11
