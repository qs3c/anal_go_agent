# Go 项目结构体依赖分析报告

**项目路径**: /Users/albert/Desktop/fromGithub/anal_go_agent/testdata/sample_project
**分析起点**: UserService
**分析深度**: 2
**生成时间**: 2026-01-20 08:45:19

---

## 分析概览

- **总结构体数**: 6
- **分析深度分布**:
  - 深度 0: 1 个
  - 深度 1: 3 个
  - 深度 2: 2 个
- **总依赖关系数**: 10
- **循环依赖**: 0 个

---

## 深度 0

### UserService

**功能**: 待分析

**所属包**: `service`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| repo | \*repository.UserRepository | ✗ | 待分析 |
| cache | \*cache.Cache | ✗ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| CreateUser | (name string, age int) error | ✓ | 待分析 |
| GetUserByID | (id int64) (\*model.User, error) | ✓ | 待分析 |
| UpdateUser | (user \*model.User) error | ✓ | 待分析 |
| DeleteUser | (id int64) error | ✓ | 待分析 |

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

**功能**: 待分析

**所属包**: `repository`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| db | \*Database | ✗ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Save | (user model.User) error | ✓ | 待分析 |
| FindByID | (id int64) (\*model.User, error) | ✓ | 待分析 |
| Update | (user \*model.User) error | ✓ | 待分析 |
| Delete | (id int64) error | ✓ | 待分析 |
| FindAll | () ([]\*model.User, error) | ✓ | 待分析 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| Database | 字段依赖 | db 字段 | 2 |
| UserRepositoryInterface | 接口实现 | 实现接口 | 2 |

---

### Cache

**功能**: 待分析

**所属包**: `cache`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| client | \*RedisClient | ✗ | 待分析 |
| ttl | time.Duration | ✗ | 待分析 |
| mu | sync.RWMutex | ✗ | 待分析 |
| local | map[int64]interface{} | ✗ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Get | (key int64) interface{} | ✓ | 待分析 |
| Set | (key int64, value interface{}) | ✓ | 待分析 |
| Delete | (key int64) | ✓ | 待分析 |
| Clear | () | ✓ | 待分析 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| RedisClient | 字段依赖 | client 字段 | 2 |

---

### User

**功能**: 待分析

**所属包**: `model`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| ID | int64 | ✓ | 待分析 |
| Name | string | ✓ | 待分析 |
| Age | int | ✓ | 待分析 |
| Email | string | ✓ | 待分析 |
| CreatedAt | time.Time | ✓ | 待分析 |
| UpdatedAt | time.Time | ✓ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Validate | () error | ✓ | 待分析 |
| IsAdult | () bool | ✓ | 待分析 |
| UpdateTimestamp | () | ✓ | 待分析 |

---

## 深度 2

### Database

**功能**: 待分析

**所属包**: `repository`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| connectionString | string | ✗ | 待分析 |
| pool | \*ConnectionPool | ✗ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Insert | (table string, data interface{}) error | ✓ | 待分析 |
| QueryByID | (table string, id int64) (interface{}, error) | ✓ | 待分析 |
| QueryAll | (table string) ([]interface{}, error) | ✓ | 待分析 |
| Update | (table string, id int64, data interface{}) error | ✓ | 待分析 |
| DeleteByID | (table string, id int64) error | ✓ | 待分析 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| ConnectionPool | 字段依赖 | pool 字段 | 3 |
| User | 方法内初始化 | QueryByID 方法 | 3 |

---

### RedisClient

**功能**: 待分析

**所属包**: `cache`

#### 字段列表

| 字段名 | 类型 | 导出 | 描述 |
|--------|------|------|------|
| address | string | ✗ | 待分析 |
| password | string | ✗ | 待分析 |
| db | int | ✗ | 待分析 |
| pool | \*RedisPool | ✗ | 待分析 |

#### 方法列表

| 方法名 | 签名 | 导出 | 描述 |
|--------|------|------|------|
| Get | (key int64) interface{} | ✓ | 待分析 |
| Set | (key int64, value interface{}, ttl time.Duration) | ✓ | 待分析 |
| Delete | (key int64) | ✓ | 待分析 |
| FlushAll | () | ✓ | 待分析 |

#### 依赖关系

| 目标结构体 | 依赖类型 | 上下文 | 深度 |
|-----------|---------|--------|------|
| RedisPool | 字段依赖 | pool 字段 | 3 |

---

## 依赖关系图

```mermaid
graph TD
    UserService["UserService<br/>待分析"]
    UserRepository["UserRepository<br/>待分析"]
    Cache["Cache<br/>待分析"]
    User["User<br/>待分析"]
    Database["Database<br/>待分析"]
    RedisClient["RedisClient<br/>待分析"]

    UserService -->|字段| UserRepository
    UserService -->|字段| Cache
    UserService -->|初始化| User
    UserRepository -->|字段| Database
    UserRepository -->|实现| UserRepositoryInterface
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
3. UserRepositoryInterface - 被依赖 1 次
4. RedisClient - 被依赖 1 次
5. ConnectionPool - 被依赖 1 次
6. RedisPool - 被依赖 1 次
7. UserRepository - 被依赖 1 次
8. Cache - 被依赖 1 次

---

生成于: 2026-01-20 08:45:19
