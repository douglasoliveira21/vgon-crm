'use client'

import { useAuthStore } from '@/store/auth'
import { User, Building, Shield, Bell, Palette } from 'lucide-react'

export default function SettingsPage() {
  const { user } = useAuthStore()

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 mb-8">Configurações</h1>

      <div className="space-y-6">
        {/* Profile */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <User size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Meu perfil</h2>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nome</label>
              <input type="text" defaultValue={user?.name} className="input" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">E-mail</label>
              <input type="email" defaultValue={user?.email} className="input" disabled />
            </div>
          </div>

          <button className="btn-primary mt-4">Salvar alterações</button>
        </div>

        {/* Company */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Building size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Empresa</h2>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nome da empresa</label>
              <input type="text" className="input" placeholder="Minha Empresa LTDA" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">CNPJ</label>
              <input type="text" className="input" placeholder="00.000.000/0001-00" />
            </div>
          </div>
        </div>

        {/* Security */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Shield size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Segurança</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Senha atual</label>
              <input type="password" className="input" placeholder="••••••••" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Nova senha</label>
                <input type="password" className="input" placeholder="••••••••" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Confirmar nova senha</label>
                <input type="password" className="input" placeholder="••••••••" />
              </div>
            </div>
          </div>

          <button className="btn-primary mt-4">Alterar senha</button>
        </div>

        {/* Notifications */}
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Bell size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900">Notificações</h2>
          </div>

          <div className="space-y-3">
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Nova conversa atribuída</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Nova mensagem recebida</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Menção com @nome</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-700">Recados internos</span>
              <input type="checkbox" defaultChecked className="rounded border-gray-300" />
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}
