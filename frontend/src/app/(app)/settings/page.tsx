'use client'

import { useEffect, useState } from 'react'
import { Bell, MessageSquare, Monitor, Moon, Sun, Volume2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { useAppearanceStore } from '@/store/appearance'

export default function SettingsPage() {
  const { theme, setTheme, sidebarPinned, setSidebarPinned } = useAppearanceStore()
  const [soundEnabled, setSoundEnabled] = useState(true)
  const [alertMine, setAlertMine] = useState(true)
  const [alertUnassigned, setAlertUnassigned] = useState(true)
  const [alertOthers, setAlertOthers] = useState(false)

  useEffect(() => {
    const notification = JSON.parse(localStorage.getItem('notification_settings') || '{}')
    const events = JSON.parse(localStorage.getItem('notification_alert_events') || '{}')
    if (notification.enabled !== undefined) setSoundEnabled(notification.enabled)
    if (events.assigned_to_me !== undefined) setAlertMine(events.assigned_to_me)
    if (events.unassigned !== undefined) setAlertUnassigned(events.unassigned)
    if (events.assigned_to_others !== undefined) setAlertOthers(events.assigned_to_others)
  }, [])

  const saveSound = (enabled: boolean) => {
    setSoundEnabled(enabled)
    localStorage.setItem('notification_settings', JSON.stringify({ enabled, sound: 'notification-1', volume: 50 }))
    toast.success(enabled ? 'Som de notificação ativado' : 'Som de notificação desativado')
  }

  const saveEvents = (mine: boolean, unassigned: boolean, others: boolean) => {
    setAlertMine(mine)
    setAlertUnassigned(unassigned)
    setAlertOthers(others)
    localStorage.setItem('notification_alert_events', JSON.stringify({
      assigned_to_me: mine,
      unassigned,
      assigned_to_others: others,
    }))
  }

  return (
    <div className="mx-auto max-w-4xl p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Configurações do sistema</h1>
        <p className="mt-1 text-sm text-gray-500">Preferências gerais do CRM. Dados pessoais ficam em Meu perfil.</p>
      </div>

      <div className="space-y-6">
        <section className="card p-6">
          <div className="mb-5 flex items-center gap-3">
            <Monitor size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Aparência</h2>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <button
              onClick={() => setTheme('light')}
              className={`rounded-xl border p-4 text-left ${theme === 'light' ? 'border-primary-500 bg-primary-50' : 'border-gray-200 dark:border-gray-800'}`}
            >
              <Sun size={18} className="mb-2 text-amber-500" />
              <p className="font-medium text-gray-900 dark:text-white">Tema claro</p>
              <p className="mt-1 text-xs text-gray-500">Interface com fundo branco.</p>
            </button>
            <button
              onClick={() => setTheme('dark')}
              className={`rounded-xl border p-4 text-left ${theme === 'dark' ? 'border-primary-500 bg-primary-950/30' : 'border-gray-200 dark:border-gray-800'}`}
            >
              <Moon size={18} className="mb-2 text-primary-400" />
              <p className="font-medium text-gray-900 dark:text-white">Tema escuro</p>
              <p className="mt-1 text-xs text-gray-500">Interface com fundo escuro.</p>
            </button>
          </div>
          <label className="mt-5 flex items-center justify-between rounded-xl border border-gray-100 p-4 dark:border-gray-800">
            <div>
              <span className="text-sm font-medium text-gray-900 dark:text-white">Fixar menu lateral</span>
              <p className="text-xs text-gray-500">Quando desativado, o menu expande apenas ao passar o mouse.</p>
            </div>
            <input type="checkbox" checked={sidebarPinned} onChange={(e) => setSidebarPinned(e.target.checked)} />
          </label>
        </section>

        <section className="card p-6">
          <div className="mb-5 flex items-center gap-3">
            <Bell size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Alertas de conversas</h2>
          </div>
          <div className="space-y-3">
            <Toggle label="Conversas atribuídas a mim" checked={alertMine} onChange={(v) => saveEvents(v, alertUnassigned, alertOthers)} />
            <Toggle label="Conversas sem atendente" checked={alertUnassigned} onChange={(v) => saveEvents(alertMine, v, alertOthers)} />
            <Toggle label="Conversas de outros atendentes" checked={alertOthers} onChange={(v) => saveEvents(alertMine, alertUnassigned, v)} />
          </div>
        </section>

        <section className="card p-6">
          <div className="mb-5 flex items-center gap-3">
            <Volume2 size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Som de notificação</h2>
          </div>
          <Toggle label="Reproduzir som ao receber nova mensagem" checked={soundEnabled} onChange={saveSound} />
        </section>

        <section className="card p-6">
          <div className="mb-5 flex items-center gap-3">
            <MessageSquare size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Canais e atendimento</h2>
          </div>
          <p className="text-sm leading-6 text-gray-500">
            Configurações de canais, Gmail, Outlook e WhatsApp ficam no menu Canais. Perfil, foto e status do atendente ficam ao clicar no seu nome no menu lateral.
          </p>
        </section>
      </div>
    </div>
  )
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-center justify-between rounded-xl border border-gray-100 p-4 dark:border-gray-800">
      <span className="text-sm font-medium text-gray-900 dark:text-white">{label}</span>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
    </label>
  )
}
