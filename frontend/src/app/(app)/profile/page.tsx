'use client'

import { useRef, useState } from 'react'
import { useAuthStore } from '@/store/auth'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Camera, Circle, KeyRound, Mail, Phone, Save, User } from 'lucide-react'

const resolveImage = (url?: string) => {
  if (!url) return ''
  return url.startsWith('/') ? `${process.env.NEXT_PUBLIC_API_URL}${url}` : url
}

const statusOptions = [
  { value: 'online', label: 'Online', description: 'Disponível para receber atendimentos', color: 'text-green-500' },
  { value: 'busy', label: 'Ocupado', description: 'Em atendimento ou indisponível no momento', color: 'text-amber-500' },
  { value: 'offline', label: 'Offline', description: 'Fora da fila de atendimento', color: 'text-gray-400' },
] as const

export default function ProfilePage() {
  const { user, updateProfile, updateStatus, uploadAvatar } = useAuthStore()
  const [name, setName] = useState(user?.name || '')
  const [phone, setPhone] = useState(user?.phone || '')
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [savingPassword, setSavingPassword] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const saveProfile = async () => {
    if (!name.trim()) {
      toast.error('Informe seu nome')
      return
    }
    setSaving(true)
    try {
      await updateProfile({ name: name.trim(), phone: phone.trim() })
      toast.success('Perfil atualizado')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao salvar perfil')
    } finally {
      setSaving(false)
    }
  }

  const changeAvatar = async (file?: File) => {
    if (!file) return
    setUploading(true)
    try {
      await uploadAvatar(file)
      toast.success('Foto atualizada')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao enviar foto')
    } finally {
      setUploading(false)
    }
  }

  const currentStatus = user?.is_online ? (user.availability_status || 'online') : 'offline'

  const changePassword = async () => {
    if (!currentPassword || !newPassword) {
      toast.error('Informe a senha atual e a nova senha')
      return
    }
    if (newPassword.length < 8) {
      toast.error('A nova senha precisa ter pelo menos 8 caracteres')
      return
    }
    if (newPassword !== confirmPassword) {
      toast.error('A confirmação não confere com a nova senha')
      return
    }

    setSavingPassword(true)
    try {
      await api.put('/me/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.success('Senha alterada')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao alterar senha')
    } finally {
      setSavingPassword(false)
    }
  }

  return (
    <div className="mx-auto max-w-5xl p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Meu perfil</h1>
        <p className="mt-1 text-sm text-gray-500">Gerencie sua foto, dados pessoais e disponibilidade como atendente.</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
        <div className="card p-6">
          <div className="flex flex-col items-center text-center">
            <div className="relative">
              {user?.avatar_url ? (
                <img src={resolveImage(user.avatar_url)} alt={user.name} className="h-32 w-32 rounded-full object-cover" />
              ) : (
                <div className="flex h-32 w-32 items-center justify-center rounded-full bg-primary-600 text-4xl font-semibold text-white">
                  {user?.name?.charAt(0)?.toUpperCase() || 'U'}
                </div>
              )}
              <button
                type="button"
                onClick={() => fileRef.current?.click()}
                disabled={uploading}
                className="absolute bottom-1 right-1 flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-white shadow-lg hover:bg-primary-700 disabled:opacity-60"
                title="Alterar foto"
              >
                <Camera size={18} />
              </button>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              className="hidden"
              onChange={(e) => changeAvatar(e.target.files?.[0])}
            />
            <h2 className="mt-4 text-lg font-semibold text-gray-900 dark:text-white">{user?.name}</h2>
            <p className="text-sm text-gray-500">{user?.role_name || 'Atendente'}</p>
            <p className="mt-3 text-xs text-gray-400">JPG, PNG ou WEBP até 3MB.</p>
          </div>
        </div>

        <div className="space-y-6">
          <div className="card p-6">
            <div className="mb-5 flex items-center gap-3">
              <User size={20} className="text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Dados do usuário</h2>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Nome</label>
                <input value={name} onChange={(e) => setName(e.target.value)} className="input" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">E-mail</label>
                <div className="relative">
                  <Mail size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input value={user?.email || ''} disabled className="input pl-9 opacity-70" />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Telefone</label>
                <div className="relative">
                  <Phone size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input value={phone} onChange={(e) => setPhone(e.target.value)} className="input pl-9" placeholder="(00) 00000-0000" />
                </div>
              </div>
            </div>
            <button onClick={saveProfile} disabled={saving} className="btn-primary mt-5">
              <Save size={16} />
              {saving ? 'Salvando...' : 'Salvar perfil'}
            </button>
          </div>

          <div className="card p-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Disponibilidade do atendente</h2>
            <p className="mt-1 text-sm text-gray-500">Esse status aparece para a equipe e ajuda na distribuição manual de atendimentos.</p>
            <div className="mt-5 grid gap-3 sm:grid-cols-3">
              {statusOptions.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => updateStatus(option.value).then(() => toast.success(`Status alterado para ${option.label}`))}
                  className={`rounded-xl border p-4 text-left transition-all ${
                    currentStatus === option.value
                      ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/40'
                      : 'border-gray-200 bg-white hover:border-gray-300 dark:border-gray-800 dark:bg-gray-900'
                  }`}
                >
                  <div className="flex items-center gap-2 font-semibold text-gray-900 dark:text-white">
                    <Circle size={10} fill="currentColor" className={option.color} />
                    {option.label}
                  </div>
                  <p className="mt-2 text-xs leading-5 text-gray-500">{option.description}</p>
                </button>
              ))}
            </div>
          </div>

          <div className="card p-6">
            <div className="mb-5 flex items-center gap-3">
              <KeyRound size={20} className="text-gray-400" />
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Trocar senha</h2>
            </div>
            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Senha atual</label>
                <input
                  type="password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  className="input"
                  placeholder="Digite a senha atual"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Nova senha</label>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="input"
                  placeholder="Mínimo 8 caracteres"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">Confirmar senha</label>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="input"
                  placeholder="Repita a nova senha"
                />
              </div>
            </div>
            <button onClick={changePassword} disabled={savingPassword} className="btn-primary mt-5">
              <Save size={16} />
              {savingPassword ? 'Alterando...' : 'Alterar senha'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
