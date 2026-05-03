<template>
  <div class="users-page">
    <div class="page-header">
      <h2>{{ t('users.title') }}</h2>
      <n-button type="primary" @click="showAddModal = true" v-if="authStore.isAdmin">
        <template #icon>
          <n-icon><add-outline /></n-icon>
        </template>
        {{ t('users.addUser') }}
      </n-button>
    </div>

    <n-card bordered>
      <template #header>
        <n-space>
          <span>{{ t('users.total', { count: data.length }) }}</span>
          <n-button size="small" @click="loadData">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
          </n-button>
        </n-space>
      </template>

      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :pagination="false"
      />
    </n-card>

    <!-- Add User Modal -->
    <n-modal v-model:show="showAddModal" preset="card" :title="t('users.addUser')" style="width: 450px">
      <n-form :model="formValue" :rules="formRules" ref="formRef" label-placement="top">
        <n-form-item :label="t('users.username')" path="username">
          <n-input v-model:value="formValue.username" placeholder="john_doe" />
        </n-form-item>
        <n-form-item :label="t('users.password')" path="password">
          <n-input v-model:value="formValue.password" type="password" show-password-on="mousedown" :placeholder="t('users.minLength')" />
        </n-form-item>
        <n-form-item :label="t('users.role')" path="role">
          <n-select
            v-model:value="formValue.role"
            :options="roleOptions"
            :placeholder="t('users.roleRequired')"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddModal = false">{{ t('users.cancel') }}</n-button>
          <n-button type="primary" @click="handleAdd" :loading="saving">{{ t('users.createUser') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Edit User Modal -->
    <n-modal v-model:show="showEditModal" preset="card" :title="t('users.edit')" style="width: 450px">
      <n-form :model="editFormValue" :rules="editFormRules" ref="editFormRef" label-placement="top">
        <n-form-item :label="t('users.username')" path="username">
          <n-input v-model:value="editFormValue.username" placeholder="john_doe" />
        </n-form-item>
        <n-form-item :label="t('users.role')" path="role">
          <n-select
            v-model:value="editFormValue.role"
            :options="roleOptions"
            :placeholder="t('users.roleRequired')"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t('users.cancel') }}</n-button>
          <n-button type="primary" @click="handleUpdate" :loading="saving">{{ t('users.updateUser') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Change Password Modal -->
    <n-modal v-model:show="showPasswordModal" preset="card" :title="t('users.changePassword')" style="width: 400px">
      <n-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-placement="top">
        <n-form-item :label="t('users.currentPassword')" path="old_password" v-if="isSelfPassword">
          <n-input v-model:value="passwordForm.old_password" type="password" show-password-on="mousedown" />
        </n-form-item>
        <n-form-item :label="t('users.newPassword')" path="new_password">
          <n-input v-model:value="passwordForm.new_password" type="password" show-password-on="mousedown" :placeholder="t('users.minLength')" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showPasswordModal = false">{{ t('users.cancel') }}</n-button>
          <n-button type="primary" @click="handleChangePassword" :loading="saving">{{ t('users.change') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NSpace, NIcon, NModal, NForm,
  NFormItem, NInput, NSelect, NTag, NPopconfirm, useMessage
} from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, CreateOutline, KeyOutline } from '@vicons/ionicons5'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const message = useMessage()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const showPasswordModal = ref(false)
const editingUser = ref<any>(null)
const isSelfPassword = ref(false)

const formRef = ref()
const editFormRef = ref()
const passwordFormRef = ref()

const formValue = ref({ username: '', password: '', role: 'viewer' })
const editFormValue = ref({ username: '', role: '' })
const passwordForm = ref({ old_password: '', new_password: '' })

const roleOptions = computed(() => [
  { label: () => t('users.roles.admin'), value: 'admin' },
  { label: () => t('users.roles.operator'), value: 'operator' },
  { label: () => t('users.roles.viewer'), value: 'viewer' }
])

const formRules = {
  username: { required: true, message: t('users.usernameRequired'), trigger: 'blur' },
  password: { required: true, message: t('users.passwordRequired'), trigger: 'blur', minLength: 6 },
  role: { required: true, message: t('users.roleRequired'), trigger: 'change' }
}

const editFormRules = {
  username: { required: true, message: t('users.usernameRequired'), trigger: 'blur' },
  role: { required: true, message: t('users.roleRequired'), trigger: 'change' }
}

const passwordRules = {
  new_password: { required: true, message: t('users.newPassword') + ' is required', trigger: 'blur', minLength: 6 }
}

const isAdmin = computed(() => authStore.isAdmin)

const columns = computed(() => [
  { title: () => t('users.id'), key: 'id', width: 80 },
  { title: () => t('users.username'), key: 'username', width: 150 },
  {
    title: () => t('users.role'),
    key: 'role',
    width: 120,
    render: (row: any) => {
      const type = row.role === 'admin' ? 'error' : row.role === 'operator' ? 'warning' : 'default'
      return h(NTag, { type, size: 'small' }, { default: () => row.role })
    }
  },
  {
    title: () => t('users.created'),
    key: 'created_at',
    width: 180,
    render: (row: any) => new Date(row.created_at).toLocaleString()
  },
  {
    title: () => t('users.actions'),
    key: 'actions',
    width: 180,
    render: (row: any) => {
      const actions: any[] = []

      actions.push(
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => openEditModal(row)
        }, { icon: () => h(NIcon, null, { default: () => h(CreateOutline) }) })
      )

      actions.push(
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => openPasswordModal(row, false)
        }, { icon: () => h(NIcon, null, { default: () => h(KeyOutline) }) })
      )

      if (isAdmin.value && row.id !== authStore.user?.id) {
        actions.push(
          h(NPopconfirm, {
            onPositiveClick: () => handleDelete(row.id)
          }, {
            trigger: () => h(NButton, { size: 'small', quaternary: true, circle: true },
              { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
            default: () => t('users.deleteConfirm')
          })
        )
      }

      return h(NSpace, { size: 'small' }, { default: () => actions })
    }
  }
])

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/users')
    data.value = res.data || []
  } catch (e: any) {
    message.error(t('users.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function openEditModal(user: any) {
  editingUser.value = user
  editFormValue.value = { username: user.username, role: user.role }
  showEditModal.value = true
}

function openPasswordModal(user: any, isSelf: boolean) {
  editingUser.value = user
  isSelfPassword.value = isSelf
  passwordForm.value = { old_password: '', new_password: '' }
  showPasswordModal.value = true
}

async function handleAdd() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    await api.post('/users', formValue.value)
    message.success(t('users.userCreated'))
    showAddModal.value = false
    formValue.value = { username: '', password: '', role: 'viewer' }
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('users.failedToCreate'))
  } finally {
    saving.value = false
  }
}

async function handleUpdate() {
  try {
    await editFormRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    await api.put(`/users/${editingUser.value.id}`, editFormValue.value)
    message.success(t('users.userUpdated'))
    showEditModal.value = false
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('users.failedToUpdate'))
  } finally {
    saving.value = false
  }
}

async function handleChangePassword() {
  try {
    await passwordFormRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    await api.post(`/users/${editingUser.value.id}/password`, {
      old_password: isSelfPassword.value ? passwordForm.value.old_password : undefined,
      new_password: passwordForm.value.new_password
    })
    message.success(t('users.passwordChanged'))
    showPasswordModal.value = false
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('users.failedToChangePassword'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/users/${id}`)
    message.success(t('users.userDeleted'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('users.failedToDelete'))
  }
}

onMounted(loadData)
</script>

<style scoped>
.users-page {
  max-width: 1200px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h2 {
  margin: 0;
}
</style>