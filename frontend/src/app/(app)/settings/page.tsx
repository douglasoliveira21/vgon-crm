'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Bell, Building, MessageSquare, Monitor, Moon, Play, Sun, Tag, Volume2 } from 'lucide-react'
import { useAppearanceStore } from '@/store/appearance'

export default function SettingsPage() {
  const { theme, setTheme, sidebarPinned, setSidebarPinned } = useAppearanceStore()

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-8">Configurações do sistema</h1>

      <div className="space-y-6">
        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Monitor size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Aparência</h2>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <button
              onClick={() => setTheme('light')}
              className={`rounded-xl border p-4 text-left transition-colors ${
                theme === 'light' ? 'border-primary-500 bg-primary-50 text-primary-700' : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <Sun size={18} className="mb-2 text-amber-500" />
              <p className="font-medium">Tema claro</p>
              <p className="mt-1 text-xs text-gray-500">Interface com fundo branco.</p>
            </button>
            <button
              onClick={() => setTheme('dark')}
              className={`rounded-xl border p-4 text-left transition-colors ${
                theme === 'dark' ? 'border-primary-500 bg-primary-950/40 text-primary-200' : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <Moon size={18} className="mb-2 text-primary-400" />
              <p className="font-medium">Tema escuro</p>
              <p className="mt-1 text-xs text-gray-500">Interface com fundo escuro.</p>
            </button>
          </div>

          <label className="mt-4 flex items-center justify-between">
            <div>
              <span className="text-sm text-gray-700">Fixar menu lateral</span>
              <p className="text-xs text-gray-400">Se desativado, o menu expande ao passar o mouse.</p>
            </div>
            <input
              type="checkbox"
              checked={sidebarPinned}
              onChange={(e) => setSidebarPinned(e.target.checked)}
              className="rounded border-gray-300"
            />
          </label>
        </div>

        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Building size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Empresa</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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

        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <Bell size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Notificações</h2>
          </div>

          <div className="space-y-3">
            <Toggle label="Nova conversa atribuída" defaultChecked />
            <Toggle label="Nova mensagem recebida" defaultChecked />
            <Toggle label="Menção com @nome" defaultChecked />
            <Toggle label="Recados internos" defaultChecked />
          </div>
        </div>

        <NotificationSoundSettings />
        <NotificationAlertEvents />

        <div className="card p-6">
          <div className="flex items-center gap-3 mb-4">
            <MessageSquare size={20} className="text-gray-400" />
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">WhatsApp / Canais</h2>
          </div>

          <div className="space-y-3">
            <label className="flex items-center justify-between gap-4">
              <div>
                <span className="text-sm text-gray-700">Receber mensagens de grupos</span>
                <p className="text-xs text-gray-400">Se desativado, mensagens de grupos do WhatsApp serão ignoradas</p>
              </div>
              <input
                type="checkbox"
                defaultChecked={false}
                onChange={async (e) => {
                  try {
                    await api.put('/settings', { receive_group_messages: e.target.checked })
                    toast.success(e.target.checked ? 'Mensagens de grupos ativadas' : 'Mensagens de grupos desativadas')
                  } catch {
                    localStorage.setItem('receive_group_messages', String(e.target.checked))
                    toast.success(e.target.checked ? 'Mensagens de grupos ativadas' : 'Mensagens de grupos desativadas')
                  }
                }}
                className="rounded border-gray-300"
              />
            </label>
            <Toggle label="Ignorar mensagens de status/stories" description="Não receber atualizações de status do WhatsApp" defaultChecked />
            <Toggle label="Enviar confirmação de leitura" description="Marcar mensagens como lidas ao visualizar no CRM" defaultChecked />
          </div>
        </div>

        <TagsManager />
      </div>
    </div>
  )
}

function Toggle({ label, description, defaultChecked = false }: { label: string; description?: string; defaultChecked?: boolean }) {
  return (
    <label className="flex items-center justify-between gap-4">
      <div>
        <span className="text-sm text-gray-700">{label}</span>
        {description && <p className="text-xs text-gray-400">{description}</p>}
      </div>
      <input type="checkbox" defaultChecked={defaultChecked} className="rounded border-gray-300" />
    </label>
  )
}

function NotificationAlertEvents() {
  const [assignedToMe, setAssignedToMe] = useState(true)
  const [unassigned, setUnassigned] = useState(true)
  const [assignedToOthers, setAssignedToOthers] = useState(false)

  useEffect(() => {
    try {
      const settings = JSON.parse(localStorage.getItem('notification_alert_events') || '{}')
      if (settings.assigned_to_me !== undefined) setAssignedToMe(settings.assigned_to_me)
      if (settings.unassigned !== undefined) setUnassigned(settings.unassigned)
      if (settings.assigned_to_others !== undefined) setAssignedToOthers(settings.assigned_to_others)
    } catch {}
  }, [])

  const save = (mine: boolean, unass: boolean, others: boolean) => {
    localStorage.setItem('notification_alert_events', JSON.stringify({
      assigned_to_me: mine,
      unassigned: unass,
      assigned_to_others: others,
    }))
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-3 mb-2">
        <Bell size={20} className="text-gray-400" />
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Eventos de alerta para conversas</h2>
      </div>
      <p className="text-xs text-gray-400 mb-4">
        Os alertas são emitidos para novas mensagens conforme as regras abaixo, inclusive com a aba em segundo plano quando permitido pelo navegador.
      </p>

      <div className="space-y-3">
        <label className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-gray-700">Conversas atribuídas a mim</span>
            <p className="text-xs text-gray-400">Alerta quando receber mensagem em conversas suas</p>
          </div>
          <input
            type="checkbox"
            checked={assignedToMe}
            onChange={(e) => { setAssignedToMe(e.target.checked); save(e.target.checked, unassigned, assignedToOthers) }}
            className="rounded border-gray-300 w-5 h-5 text-primary-600"
          />
        </label>

        <label className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-gray-700">Conversas não atribuídas</span>
            <p className="text-xs text-gray-400">Alerta quando mensagem chega em conversa sem atendente</p>
          </div>
          <input
            type="checkbox"
            checked={unassigned}
            onChange={(e) => { setUnassigned(e.target.checked); save(assignedToMe, e.target.checked, assignedToOthers) }}
            className="rounded border-gray-300 w-5 h-5 text-primary-600"
          />
        </label>

        <label className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-gray-700">Conversas atribuídas a outras pessoas</span>
            <p className="text-xs text-gray-400">Alerta quando mensagem chega em conversa de outro atendente</p>
          </div>
          <input
            type="checkbox"
            checked={assignedToOthers}
            onChange={(e) => { setAssignedToOthers(e.target.checked); save(assignedToMe, unassigned, e.target.checked) }}
            className="rounded border-gray-300 w-5 h-5 text-primary-600"
          />
        </label>
      </div>
    </div>
  )
}

function NotificationSoundSettings() {
  const [enabled, setEnabled] = useState(true)
  const [browserEnabled, setBrowserEnabled] = useState(true)
  const [browserPermission, setBrowserPermission] = useState<'default' | 'granted' | 'denied' | 'unsupported'>('unsupported')
  const [sound, setSound] = useState('notification-1')
  const [volume, setVolume] = useState(50)

  const soundOptions = [
    { id: 'notification-1', label: 'Padrão' },
    { id: 'notification-2', label: 'Suave' },
    { id: 'notification-3', label: 'Alerta' },
    { id: 'notification-4', label: 'Mensagem' },
    { id: 'notification-5', label: 'Pop' },
    { id: 'notification-6', label: 'Ding' },
  ]

  useEffect(() => {
    try {
      const settings = JSON.parse(localStorage.getItem('notification_settings') || '{}')
      if (settings.enabled !== undefined) setEnabled(settings.enabled)
      if (settings.browserEnabled !== undefined) setBrowserEnabled(settings.browserEnabled)
      if (settings.sound) setSound(settings.sound)
      if (settings.volume !== undefined) setVolume(settings.volume)
      setBrowserPermission('Notification' in window ? Notification.permission : 'unsupported')
    } catch {}
  }, [])

  const saveSettings = (newEnabled: boolean, newSound: string, newVolume: number, newBrowserEnabled = browserEnabled) => {
    localStorage.setItem('notification_settings', JSON.stringify({
      enabled: newEnabled,
      browserEnabled: newBrowserEnabled,
      sound: newSound,
      volume: newVolume,
    }))
  }

  const toggleEnabled = (val: boolean) => {
    setEnabled(val)
    saveSettings(val, sound, volume)
    toast.success(val ? 'Som de notificação ativado' : 'Som de notificação desativado')
  }

  const toggleBrowserEnabled = async (val: boolean) => {
    if (!('Notification' in window)) {
      setBrowserPermission('unsupported')
      toast.error('Este navegador não suporta notificações.')
      return
    }

    let permission = Notification.permission
    if (val && permission === 'default') {
      permission = await Notification.requestPermission()
    }

    setBrowserPermission(permission)
    if (val && permission !== 'granted') {
      setBrowserEnabled(false)
      saveSettings(enabled, sound, volume, false)
      toast.error('Permita as notificações do navegador para receber alertas fora da aba.')
      return
    }

    setBrowserEnabled(val)
    saveSettings(enabled, sound, volume, val)
    toast.success(val ? 'Notificações do navegador ativadas' : 'Notificações do navegador desativadas')
  }

  const changeSound = (newSound: string) => {
    setSound(newSound)
    saveSettings(enabled, newSound, volume)
    try {
      const audio = new Audio(`/sounds/${newSound}.wav`)
      audio.volume = volume / 100
      audio.play().catch(() => {})
    } catch {}
  }

  const changeVolume = (newVolume: number) => {
    setVolume(newVolume)
    saveSettings(enabled, sound, newVolume)
  }

  const testSound = () => {
    try {
      const audio = new Audio(`/sounds/${sound}.wav`)
      audio.volume = volume / 100
      audio.play().catch(() => toast.error('Não foi possível reproduzir o som.'))
    } catch {
      toast.error('Erro ao reproduzir som')
    }
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-3 mb-4">
        <Volume2 size={20} className="text-gray-400" />
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Som de notificação</h2>
      </div>

      <div className="space-y-4">
        <label className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-gray-700">Ativar som de notificação</span>
            <p className="text-xs text-gray-400">Reproduzir um som ao receber nova mensagem</p>
          </div>
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => toggleEnabled(e.target.checked)}
            className="rounded border-gray-300 w-5 h-5 text-primary-600"
          />
        </label>

        <label className="flex items-center justify-between">
          <div>
            <span className="text-sm font-medium text-gray-700">Notificação do navegador</span>
            <p className="text-xs text-gray-400">
              Exibir alerta do sistema quando o navegador estiver minimizado ou em outra aba.
              {browserPermission === 'denied' && ' Permissão bloqueada no navegador.'}
            </p>
          </div>
          <input
            type="checkbox"
            checked={browserEnabled && browserPermission !== 'denied' && browserPermission !== 'unsupported'}
            onChange={(e) => toggleBrowserEnabled(e.target.checked)}
            disabled={browserPermission === 'denied' || browserPermission === 'unsupported'}
            className="rounded border-gray-300 w-5 h-5 text-primary-600 disabled:opacity-50"
          />
        </label>
        {enabled && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Escolher som</label>
              <div className="grid grid-cols-2 gap-2">
                {soundOptions.map((option) => (
                  <button
                    key={option.id}
                    onClick={() => changeSound(option.id)}
                    className={`px-3 py-2.5 text-sm rounded-lg border transition-colors text-left flex items-center gap-2 ${
                      sound === option.id
                        ? 'border-primary-500 bg-primary-50 text-primary-700 font-medium'
                        : 'border-gray-200 text-gray-600 hover:border-gray-300 hover:bg-gray-50'
                    }`}
                  >
                    <span className="text-lg">{sound === option.id ? '🔊' : '🔈'}</span>
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Volume: {volume}%
              </label>
              <input
                type="range"
                min={0}
                max={100}
                value={volume}
                onChange={(e) => changeVolume(Number(e.target.value))}
                className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer accent-primary-600"
              />
            </div>

            <button onClick={testSound} className="btn-secondary text-sm flex items-center gap-2">
              <Play size={14} /> Testar som
            </button>
          </>
        )}
      </div>
    </div>
  )
}

function TagsManager() {
  const [tags, setTags] = useState<Array<{id: string; name: string; color: string}>>([])
  const [newTag, setNewTag] = useState('')
  const [newColor, setNewColor] = useState('#3B82F6')

  useEffect(() => {
    fetchTags()
  }, [])

  const fetchTags = async () => {
    try {
      const res = await api.get('/tags')
      setTags(res.data.tags || [])
    } catch {}
  }

  const createTag = async () => {
    if (!newTag.trim()) return
    try {
      await api.post('/tags', { name: newTag, color: newColor })
      toast.success('Tag criada')
      setNewTag('')
      fetchTags()
    } catch {
      toast.error('Erro ao criar tag')
    }
  }

  const deleteTag = async (id: string) => {
    if (!confirm('Remover esta tag?')) return
    try {
      await api.delete(`/tags/${id}`)
      setTags(tags.filter(t => t.id !== id))
      toast.success('Tag removida')
    } catch {
      toast.error('Erro ao remover')
    }
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-3 mb-4">
        <Tag size={20} className="text-gray-400" />
        <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Tags / Etiquetas</h2>
      </div>

      <div className="flex items-center gap-2 mb-4">
        <input
          type="color"
          value={newColor}
          onChange={(e) => setNewColor(e.target.value)}
          className="w-9 h-9 rounded-lg border border-gray-200 cursor-pointer"
        />
        <input
          type="text"
          value={newTag}
          onChange={(e) => setNewTag(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') createTag() }}
          className="input flex-1"
          placeholder="Nome da nova tag..."
        />
        <button onClick={createTag} disabled={!newTag.trim()} className="btn-primary text-sm py-2.5">
          Criar
        </button>
      </div>

      <div className="flex flex-wrap gap-2">
        {tags.map((tag) => (
          <div
            key={tag.id}
            className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium border"
            style={{ backgroundColor: tag.color + '20', borderColor: tag.color, color: tag.color }}
          >
            <span className="w-2 h-2 rounded-full" style={{ backgroundColor: tag.color }} />
            {tag.name}
            <button onClick={() => deleteTag(tag.id)} className="ml-1 opacity-50 hover:opacity-100">
              ×
            </button>
          </div>
        ))}
        {tags.length === 0 && (
          <p className="text-sm text-gray-400">Nenhuma tag criada. Crie tags para organizar seus contatos.</p>
        )}
      </div>
    </div>
  )
}
