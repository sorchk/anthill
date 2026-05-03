# 初次使用流程设计

## 概述

系统首次访问时，判断数据库中是否有账号：
- 无账号 → 引导用户创建管理员账号，然后跳转登录页
- 有账号 → 直接跳转到登录页面要求登录

## 实现方案

采用**方案 A：前端路由拦截 + 独立向导页**

```
用户访问系统
    ↓
调用 /api/auth/check
    ↓
┌─────────────────────────┐
│ initialized = true?     │
└─────────────────────────┘
      ↓                ↓
     是                否
      ↓                ↓
  跳转 /login      跳转 /init
      ↓                ↓
  登录表单      Step1: 创建账号表单
      ↓                ↓
  登录成功      提交到 /api/auth/init
      ↓                ↓
  跳转 /       跳转 /init?step=2
                      ↓
                 Step2: 成功提示
                      ↓
                 3秒后跳转 /login
```

## API 设计

### 1. 检查系统是否已初始化

**接口：`GET /api/auth/check`**

```json
// 响应 - 已初始化（有账号）
{ "initialized": true }

// 响应 - 未初始化（无账号）
{ "initialized": false }
```

### 2. 初始化管理员账号

**接口：`POST /api/auth/init`**

```json
// 请求
{
  "username": "admin",
  "password": "admin123",
  "confirmPassword": "admin123"
}

// 响应 - 成功
{ "success": true }

// 响应 - 失败
{ "success": false, "error": "密码不匹配" }
```

## 路由设计

| 路由 | 页面 | 认证要求 |
|------|------|----------|
| `/init` | Init.vue - 初始化向导页 | 无需登录 |
| `/login` | Login.vue - 登录页 | 无需登录 |
| `/` | Dashboard - 仪表盘 | 需要登录 |

### 路由拦截逻辑

1. 访问 `/` 或需认证页面时：
   - 调用 `/api/auth/check`
   - `initialized = false` → redirect `/init`
   - `initialized = true` 但未登录 → redirect `/login`
   - 已登录 → 放行

2. 访问 `/login` 时：
   - 已登录 → redirect `/`

3. 访问 `/init` 时：
   - `initialized = true` → redirect `/login`

## 页面设计

### Init.vue 向导页

**视觉风格：** 与现有 Login.vue 保持一致（卡片样式、配色）

**Step 1 - 创建管理员账号**
- 标题：欢迎使用 Anthill
- 副标题：请创建管理员账号
- 表单字段：
  - 用户名（text）
  - 密码（password）
  - 确认密码（password）
- 按钮：下一步
- 校验规则：
  - 所有字段必填
  - 密码最少 6 字符
  - 两次密码必须一致

**Step 2 - 完成确认**
- 显示成功图标/动画
- 文字：账号创建成功，正在跳转...
- 3秒后自动跳转到 `/login`

## 文件结构

```
web/src/views/
  ├── Login.vue          # 现有登录页（不改动）
  └── Init.vue           # 新增：初始化向导页

web/src/router/index.ts   # 修改：添加 init 路由和路由拦截

admin/internal/handler/auth.go  # 新增：check, init handlers
admin/internal/router/router.go  # 修改：添加 auth/check, auth/init 路由
```

## 错误处理

| 场景 | 处理方式 |
|------|----------|
| API 调用失败 | 显示错误提示，允许重试 |
| 用户名已存在 | 提示"用户名已存在" |
| 密码不匹配 | 提示"两次输入的密码不一致" |
| 网络错误 | 提示"网络错误，请检查网络连接" |

## 后端实现要点

1. `/api/auth/check` - 查询 `users` 表是否有记录，返回 `{ initialized: count > 0 }`
2. `/api/auth/init` - 创建管理员账号，密码需 bcrypt 加密
3. 初始化接口只能调用一次（校验已初始化状态）

## 前端实现要点

1. 路由守卫统一处理初始化检查
2. Init.vue 使用 step 状态管理向导步骤
3. Step2 使用 setTimeout 延迟跳转
4. 保持与 Login.vue 一致的样式和交互体验
