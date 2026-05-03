# 初次使用流程实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现初次使用流程：首次访问判断数据库是否有账号，无账号引导创建管理员账号，有账号直接跳转登录页

**Architecture:** 前端路由拦截 + 独立向导页方案。路由守卫统一处理初始化检查，未初始化时跳转到 `/init` 向导页。

**Tech Stack:** Go (Gin), Vue 3, TypeScript, Pinia, Naive UI

---

## 文件结构

```
# 后端
admin/internal/handler/auth.go    # 新增 Check, Init handlers
admin/internal/router/router.go   # 新增 /api/auth/check, /api/auth/init 路由

# 前端
web/src/views/Init.vue            # 新增：初始化向导页
web/src/router/index.ts           # 新增 /init 路由，修改路由守卫
web/src/stores/auth.ts            # 新增 checkInitialized, initAdmin 方法
web/src/locales/zh-CN.json        # 新增 init 页面翻译
web/src/locales/en-US.json        # 新增 init 页面翻译
```

---

## Task 1: 后端 - 新增 Check 和 Init Handlers

**Files:**
- Modify: `admin/internal/handler/auth.go:1-103`

- [ ] **Step 1: 添加 Check handler**

在 `auth.go` 文件末尾添加以下方法：

```go
func (h *AuthHandler) Check(c *gin.Context) {
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"initialized": count > 0})
}
```

- [ ] **Step 2: 添加 Init handler**

```go
func (h *AuthHandler) Init(c *gin.Context) {
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "System already initialized"})
		return
	}

	var req struct {
		Username        string `json:"username" binding:"required"`
		Password        string `json:"password" binding:"required,min=6"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request"})
		return
	}

	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Passwords do not match"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to hash password"})
		return
	}

	_, err = h.DB.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')",
		req.Username, string(passwordHash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
```

- [ ] **Step 3: 提交代码**

```bash
cd /datadisk/anthill && git add admin/internal/handler/auth.go && git commit -m "feat(auth): add Check and Init handlers for first-time setup"
```

---

## Task 2: 后端 - 添加路由

**Files:**
- Modify: `admin/internal/router/router.go:41-45`

- [ ] **Step 1: 在 auth group 中添加新路由**

找到 `router.go` 中的 auth group 部分：

```go
auth := api.Group("/auth")
{
    auth.POST("/login", authHandler.Login)
    auth.POST("/logout", authHandler.Logout)
}
```

修改为：

```go
auth := api.Group("/auth")
{
    auth.POST("/login", authHandler.Login)
    auth.POST("/logout", authHandler.Logout)
    auth.GET("/check", authHandler.Check)
    auth.POST("/init", authHandler.Init)
}
```

- [ ] **Step 2: 提交代码**

```bash
cd /datadisk/anthill && git add admin/internal/router/router.go && git commit -m "feat(router): add /api/auth/check and /api/auth/init routes"
```

---

## Task 3: 前端 - 添加 auth store 方法

**Files:**
- Modify: `web/src/stores/auth.ts:1-50`

- [ ] **Step 1: 添加 checkInitialized 和 initAdmin 方法**

在 `auth.ts` 的 `return` 语句之前添加：

```typescript
async function checkInitialized(): Promise<boolean> {
  try {
    const response = await axios.get('/api/auth/check')
    return response.data.initialized
  } catch {
    return true
  }
}

async function initAdmin(username: string, password: string, confirmPassword: string) {
  const response = await axios.post('/api/auth/init', {
    username,
    password,
    confirmPassword
  })
  return response.data
}
```

在 `return` 语句中添加新方法：

```typescript
return { token, user, isAuthenticated, isAdmin, login, logout, fetchUser, checkInitialized, initAdmin }
```

- [ ] **Step 2: 提交代码**

```bash
cd /datadisk/anthill && git add web/src/stores/auth.ts && git commit -m "feat(auth): add checkInitialized and initAdmin methods"
```

---

## Task 4: 前端 - 添加国际化翻译

**Files:**
- Modify: `web/src/locales/zh-CN.json:1-331`
- Modify: `web/src/locales/en-US.json:1-331`

- [ ] **Step 1: 添加中文翻译**

在 `zh-CN.json` 的 `login` 部分之后添加：

```json
"init": {
  "welcomeTitle": "欢迎使用 Anthill",
  "welcomeSubtitle": "请创建管理员账号",
  "username": "用户名",
  "password": "密码",
  "confirmPassword": "确认密码",
  "next": "下一步",
  "creating": "创建中...",
  "usernameRequired": "用户名为必填项",
  "passwordRequired": "密码为必填项",
  "passwordMinLength": "密码至少6个字符",
  "passwordMismatch": "两次输入的密码不一致",
  "usernameExists": "用户名已存在",
  "createSuccess": "账号创建成功",
  "redirecting": "正在跳转...",
  "networkError": "网络错误，请检查网络连接",
  "createFailed": "创建账号失败"
},
```

- [ ] **Step 2: 添加英文翻译**

在 `en-US.json` 的 `login` 部分之后添加：

```json
"init": {
  "welcomeTitle": "Welcome to Anthill",
  "welcomeSubtitle": "Please create an administrator account",
  "username": "Username",
  "password": "Password",
  "confirmPassword": "Confirm Password",
  "next": "Next",
  "creating": "Creating...",
  "usernameRequired": "Username is required",
  "passwordRequired": "Password is required",
  "passwordMinLength": "Password must be at least 6 characters",
  "passwordMismatch": "Passwords do not match",
  "usernameExists": "Username already exists",
  "createSuccess": "Account created successfully",
  "redirecting": "Redirecting...",
  "networkError": "Network error, please check your connection",
  "createFailed": "Failed to create account"
},
```

- [ ] **Step 3: 提交代码**

```bash
cd /datadisk/anthill && git add web/src/locales/zh-CN.json web/src/locales/en-US.json && git commit -m "feat(i18n): add init page translations"
```

---

## Task 5: 前端 - 创建 Init.vue 向导页

**Files:**
- Create: `web/src/views/Init.vue`

- [ ] **Step 1: 创建 Init.vue 文件**

```vue
<template>
  <div class="init-page">
    <div class="init-card">
      <div class="init-header">
        <n-icon size="48" depth="3">
          <server-outline />
        </n-icon>
        <h1>{{ t('init.welcomeTitle') }}</h1>
        <p>{{ t('init.welcomeSubtitle') }}</p>
      </div>

      <n-form
        v-if="step === 1"
        ref="formRef"
        :model="formValue"
        :rules="rules"
        size="large"
        @submit.prevent="handleNext"
      >
        <n-form-item path="username" :label="t('init.username')">
          <n-input
            v-model:value="formValue.username"
            :placeholder="t('init.username')"
            :maxlength="32"
            clearable
          >
            <template #prefix>
              <n-icon><person-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-form-item path="password" :label="t('init.password')">
          <n-input
            v-model:value="formValue.password"
            type="password"
            :placeholder="t('init.password')"
            show-password-on="mousedown"
            :maxlength="64"
          >
            <template #prefix>
              <n-icon><lock-closed-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-form-item path="confirmPassword" :label="t('init.confirmPassword')">
          <n-input
            v-model:value="formValue.confirmPassword"
            type="password"
            :placeholder="t('init.confirmPassword')"
            show-password-on="mousedown"
            :maxlength="64"
            @keydown.enter="handleNext"
          >
            <template #prefix>
              <n-icon><lock-closed-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-button
          type="primary"
          attr-type="submit"
          block
          :loading="loading"
          :disabled="loading"
          @click="handleNext"
        >
          {{ loading ? t('init.creating') : t('init.next') }}
        </n-button>
      </n-form>

      <div v-else-if="step === 2" class="init-success">
        <n-icon size="64" color="#18a058">
          <checkmark-circle-outline />
        </n-icon>
        <p>{{ t('init.createSuccess') }}</p>
        <p class="redirect-text">{{ t('init.redirecting') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NForm, NFormItem, NInput, NButton, NIcon,
  useMessage
} from 'naive-ui'
import { ServerOutline, PersonOutline, LockClosedOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const message = useMessage()
const authStore = useAuthStore()

const step = ref(1)
const loading = ref(false)
const formRef = ref()

const formValue = ref({
  username: '',
  password: '',
  confirmPassword: ''
})

const rules = {
  username: { required: true, message: t('init.usernameRequired'), trigger: 'blur' },
  password: [
    { required: true, message: t('init.passwordRequired'), trigger: 'blur' },
    { min: 6, message: t('init.passwordMinLength'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('init.passwordRequired'), trigger: 'blur' },
    {
      validator: (_rule: any, value: string) => value === formValue.value.password,
      message: t('init.passwordMismatch'),
      trigger: 'blur'
    }
  ]
}

async function handleNext() {
  if (loading.value) return

  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  loading.value = true

  try {
    await authStore.initAdmin(
      formValue.value.username,
      formValue.value.password,
      formValue.value.confirmPassword
    )
    step.value = 2
    setTimeout(() => {
      router.push('/login')
    }, 3000)
  } catch (error: any) {
    const errorMsg = error?.response?.data?.error || t('init.createFailed')
    if (errorMsg.includes('already exists')) {
      message.error(t('init.usernameExists'))
    } else if (errorMsg.includes('network') || error?.code === 'ECONNREFUSED') {
      message.error(t('init.networkError'))
    } else {
      message.error(errorMsg)
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const initialized = await authStore.checkInitialized()
  if (initialized) {
    router.push('/login')
  }
})
</script>

<style scoped>
.init-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.init-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.init-header {
  text-align: center;
  margin-bottom: 32px;
}

.init-header .n-icon {
  color: #18a058;
  margin-bottom: 16px;
}

.init-header h1 {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px;
}

.init-header p {
  font-size: 14px;
  color: #999;
  margin: 0;
}

.init-success {
  text-align: center;
  padding: 32px 0;
}

.init-success p {
  margin: 16px 0 0;
  font-size: 16px;
  color: #333;
}

.redirect-text {
  color: #999;
  font-size: 14px !important;
}
</style>
```

- [ ] **Step 2: 提交代码**

```bash
cd /datadisk/anthill && git add web/src/views/Init.vue && git commit -m "feat: add Init.vue for first-time setup wizard"
```

---

## Task 6: 前端 - 修改路由配置

**Files:**
- Modify: `web/src/router/index.ts:1-46`

- [ ] **Step 1: 添加 /init 路由**

找到 routes 数组，添加 Init 路由：

```typescript
const routes: RouteRecordRaw[] = [
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/views/Init.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  // ... existing routes
]
```

- [ ] **Step 2: 修改路由守卫逻辑**

找到 `router.beforeEach`，修改为：

```typescript
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  if (to.name === 'Init') {
    const initialized = await authStore.checkInitialized()
    if (initialized) {
      next({ name: 'Login' })
    } else {
      next()
    }
    return
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    const initialized = await authStore.checkInitialized()
    if (!initialized) {
      next({ name: 'Init' })
    } else {
      next({ name: 'Login' })
    }
  } else if (to.name === 'Login' && authStore.isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})
```

- [ ] **Step 3: 提交代码**

```bash
cd /datadisk/anthill && git add web/src/router/index.ts && git commit -m "feat(router): add /init route and update navigation guard"
```

---

## Task 7: 验证实现

- [ ] **Step 1: 启动后端服务**

```bash
cd /datadisk/anthill/admin && go build -o anthill ./cmd/server && ./anthill &
```

- [ ] **Step 2: 测试 API**

```bash
# 检查系统是否已初始化（应返回 initialized: false）
curl http://localhost:8080/api/auth/check

# 初始化管理员账号
curl -X POST http://localhost:8080/api/auth/init \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","confirmPassword":"admin123"}'

# 再次检查（应返回 initialized: true）
curl http://localhost:8080/api/auth/check
```

- [ ] **Step 3: 启动前端服务**

```bash
cd /datadisk/anthill/web && npm run dev
```

- [ ] **Step 4: 手动测试流程**

1. 清空浏览器 localStorage（如果有 token）
2. 访问 http://localhost:3000
3. 验证：应该跳转到 /init 页面
4. 填写表单创建账号
5. 验证：Step 2 显示成功后 3 秒跳转到 /login
6. 使用创建的账号登录
7. 验证：登录成功后跳转到 Dashboard

---

## 实现顺序

1. Task 1: 后端 - 新增 Check 和 Init Handlers
2. Task 2: 后端 - 添加路由
3. Task 3: 前端 - 添加 auth store 方法
4. Task 4: 前端 - 添加国际化翻译
5. Task 5: 前端 - 创建 Init.vue 向导页
6. Task 6: 前端 - 修改路由配置
7. Task 7: 验证实现
