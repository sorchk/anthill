<template>
  <div class="users-page max-w-6xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('users.title') }}</h2>
      <Button v-if="authStore.isAdmin" @click="showAddModal = true">
        <Plus class="h-4 w-4 mr-2" />
        {{ t('users.addUser') }}
      </Button>
    </div>

    <Card>
      <CardHeader>
        <div class="flex items-center justify-between">
          <span>{{ t('users.total', { count: data.length }) }}</span>
          <Button variant="outline" size="sm" @click="loadData">
            <Refresh class="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('users.id') }}</TableHead>
              <TableHead>{{ t('users.username') }}</TableHead>
              <TableHead>{{ t('users.role') }}</TableHead>
              <TableHead>{{ t('users.created') }}</TableHead>
              <TableHead>{{ t('users.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.id }}</TableCell>
              <TableCell>{{ row.username }}</TableCell>
              <TableCell>
                <Badge :variant="row.role === 'admin' ? 'destructive' : row.role === 'operator' ? 'default' : 'secondary'">
                  {{ row.role }}
                </Badge>
              </TableCell>
              <TableCell>{{ new Date(row.created_at).toLocaleString() }}</TableCell>
              <TableCell>
                <div class="flex gap-2">
                  <Button variant="ghost" size="sm" @click="openEditModal(row)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="sm" @click="openPasswordModal(row, false)">
                    <Key class="h-4 w-4" />
                  </Button>
                  <AlertDialog v-if="authStore.isAdmin && row.id !== authStore.user?.id">
                    <AlertDialogTrigger as-child>
                      <Button variant="ghost" size="sm">
                        <Trash class="h-4 w-4 text-destructive" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>{{ t('users.deleteConfirm') }}</AlertDialogTitle>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>{{ t('users.cancel') }}</AlertDialogCancel>
                        <AlertDialogAction @click="handleDelete(row.id)">{{ t('users.delete') }}</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="showAddModal">
      <DialogContent class="sm:max-w-[450px]">
        <DialogHeader>
          <DialogTitle>{{ t('users.addUser') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleAdd" class="space-y-4">
          <div class="space-y-2">
            <Label for="username">{{ t('users.username') }}</Label>
            <Input id="username" v-model="formValue.username" placeholder="john_doe" />
          </div>
          <div class="space-y-2">
            <Label for="password">{{ t('users.password') }}</Label>
            <Input id="password" v-model="formValue.password" type="password" :placeholder="t('users.minLength')" />
          </div>
          <div class="space-y-2">
            <Label for="role">{{ t('users.role') }}</Label>
            <Select v-model="formValue.role">
              <SelectTrigger>
                <SelectValue :placeholder="t('users.roleRequired')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">{{ t('users.roles.admin') }}</SelectItem>
                <SelectItem value="operator">{{ t('users.roles.operator') }}</SelectItem>
                <SelectItem value="viewer">{{ t('users.roles.viewer') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showAddModal = false">{{ t('users.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('users.createUser') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showEditModal">
      <DialogContent class="sm:max-w-[450px]">
        <DialogHeader>
          <DialogTitle>{{ t('users.edit') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleUpdate" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-username">{{ t('users.username') }}</Label>
            <Input id="edit-username" v-model="editFormValue.username" />
          </div>
          <div class="space-y-2">
            <Label for="edit-role">{{ t('users.role') }}</Label>
            <Select v-model="editFormValue.role">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">{{ t('users.roles.admin') }}</SelectItem>
                <SelectItem value="operator">{{ t('users.roles.operator') }}</SelectItem>
                <SelectItem value="viewer">{{ t('users.roles.viewer') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showEditModal = false">{{ t('users.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('users.updateUser') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showPasswordModal">
      <DialogContent class="sm:max-w-[400px]">
        <DialogHeader>
          <DialogTitle>{{ t('users.changePassword') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleChangePassword" class="space-y-4">
          <div v-if="isSelfPassword" class="space-y-2">
            <Label for="old-password">{{ t('users.currentPassword') }}</Label>
            <Input id="old-password" v-model="passwordForm.old_password" type="password" />
          </div>
          <div class="space-y-2">
            <Label for="new-password">{{ t('users.newPassword') }}</Label>
            <Input id="new-password" v-model="passwordForm.new_password" type="password" :placeholder="t('users.minLength')" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showPasswordModal = false">{{ t('users.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('users.change') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Refresh, Pencil, Key, Trash } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const { toast } = useToast()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const showPasswordModal = ref(false)
const editingUser = ref<any>(null)
const isSelfPassword = ref(false)

const formValue = ref({ username: '', password: '', role: 'viewer' })
const editFormValue = ref({ username: '', role: '' })
const passwordForm = ref({ old_password: '', new_password: '' })

const isAdmin = computed(() => authStore.isAdmin)

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/users')
    data.value = res.data || []
  } catch {
    toast({ title: 'Error', description: t('users.failedToLoad'), variant: 'destructive' })
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
  if (!formValue.value.username || !formValue.value.password) return

  saving.value = true
  try {
    await api.post('/users', formValue.value)
    toast({ title: t('users.userCreated') })
    showAddModal.value = false
    formValue.value = { username: '', password: '', role: 'viewer' }
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('users.failedToCreate'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleUpdate() {
  if (!editingUser.value) return

  saving.value = true
  try {
    await api.put(`/users/${editingUser.value.id}`, editFormValue.value)
    toast({ title: t('users.userUpdated') })
    showEditModal.value = false
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('users.failedToUpdate'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleChangePassword() {
  if (!editingUser.value) return

  saving.value = true
  try {
    await api.post(`/users/${editingUser.value.id}/password`, {
      old_password: isSelfPassword.value ? passwordForm.value.old_password : undefined,
      new_password: passwordForm.value.new_password
    })
    toast({ title: t('users.passwordChanged') })
    showPasswordModal.value = false
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('users.failedToChangePassword'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/users/${id}`)
    toast({ title: t('users.userDeleted') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('users.failedToDelete'), variant: 'destructive' })
  }
}

onMounted(loadData)
</script>