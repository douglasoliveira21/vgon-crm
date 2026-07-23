'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Plus, Globe, Code, Eye } from 'lucide-react'
import { ChannelIcon } from '@/components/channel-icon'

export default function WidgetPage() {
  const [widgets, setWidgets] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [showPreview, setShowPreview] = useState(false)
  const [previewColor, setPreviewColor] = useState('#3B82F6')
  const [position, setPosition] = useState('bottom-right')
  const [saving, setSaving] = useState(false)
  const activeWidget = widgets[0]
  const [previewMessage, setPreviewMessage] = useState('Olá! Como podemos ajudar?')

  useEffect(() => {
    fetchWidgets()
  }, [])

  useEffect(() => {
    if (!activeWidget) return
    setPreviewColor(activeWidget.primary_color || '#3B82F6')
    setPreviewMessage(activeWidget.greeting_message || 'Olá! Como podemos ajudar?')
    setPosition(activeWidget.position || 'bottom-right')
  }, [activeWidget?.id])

  const fetchWidgets = async () => {
    try {
      const response = await api.get('/widgets')
      setWidgets(response.data.widgets || [])
    } catch (error) {
      console.error('Error:', error)
    } finally {
      setLoading(false)
    }
  }

  const getInstallScript = (widgetId: string) => {
    return `<script src="${process.env.NEXT_PUBLIC_API_URL}/widget/${widgetId}/embed.js"></script>`
  }

  const createWidget = async () => {
    setSaving(true)
    try {
      await api.post('/widgets', {
        name: 'Chat do site',
        primary_color: previewColor,
        greeting_message: previewMessage,
        position,
      })
      toast.success('Widget criado')
      fetchWidgets()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao criar widget')
    } finally {
      setSaving(false)
    }
  }

  const saveWidget = async () => {
    if (!activeWidget) {
      await createWidget()
      return
    }
    setSaving(true)
    try {
      await api.put(`/widgets/${activeWidget.id}`, {
        name: activeWidget.name || 'Chat do site',
        primary_color: previewColor,
        greeting_message: previewMessage,
        position,
        is_active: true,
      })
      toast.success('Widget salvo')
      fetchWidgets()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao salvar widget')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mx-auto max-w-5xl p-4 sm:p-6">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Widget para Site</h1>
          <p className="text-gray-500 mt-1">Configure o chat widget para seu website</p>
        </div>
        <button onClick={createWidget} disabled={saving} className="btn-primary">
          <Plus size={18} />
          Novo widget
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Configuration */}
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Personalização</h3>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Cor principal</label>
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={previewColor}
                  onChange={(e) => setPreviewColor(e.target.value)}
                  className="w-10 h-10 rounded-lg border border-gray-200 cursor-pointer"
                />
                <input
                  type="text"
                  value={previewColor}
                  onChange={(e) => setPreviewColor(e.target.value)}
                  className="input w-32"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Mensagem de saudação</label>
              <textarea
                value={previewMessage}
                onChange={(e) => setPreviewMessage(e.target.value)}
                className="input resize-none"
                rows={3}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Posição</label>
              <select value={position} onChange={(e) => setPosition(e.target.value)} className="input">
                <option value="bottom-right">Inferior direito</option>
                <option value="bottom-left">Inferior esquerdo</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Canais exibidos</label>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input type="checkbox" defaultChecked className="rounded border-gray-300" />
                  <span className="text-sm text-gray-600">WhatsApp</span>
                </label>
              </div>
            </div>
          </div>

          <button onClick={saveWidget} disabled={saving} className="btn-primary mt-6 w-full">
            {saving ? 'Salvando...' : 'Salvar configurações'}
          </button>
        </div>

        {/* Preview */}
        <div className="card p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Preview</h3>

          <div className="bg-gray-100 rounded-xl p-6 min-h-[400px] relative">
            {/* Mock website */}
            <div className="bg-white rounded-lg p-4 mb-4 shadow-sm">
              <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
              <div className="h-3 bg-gray-100 rounded w-full mb-1"></div>
              <div className="h-3 bg-gray-100 rounded w-5/6"></div>
            </div>

            {/* Widget bubble */}
            <div className="absolute bottom-6 right-6">
              {showPreview && (
                <div className="bg-white rounded-2xl shadow-lg border border-gray-200 w-72 mb-3 overflow-hidden">
                  <div className="p-4" style={{ backgroundColor: previewColor }}>
                    <p className="text-white text-sm font-medium">{previewMessage}</p>
                  </div>
                  <div className="p-4">
                    <div className="flex items-center gap-3 p-3 bg-green-50 rounded-lg cursor-pointer hover:bg-green-100 transition-colors">
                      <ChannelIcon type="whatsapp" size={20} />
                      <span className="text-sm font-medium text-green-700">Falar pelo WhatsApp</span>
                    </div>
                  </div>
                </div>
              )}

              <button
                onClick={() => setShowPreview(!showPreview)}
                className="w-14 h-14 rounded-full shadow-lg flex items-center justify-center transition-transform hover:scale-110"
                style={{ backgroundColor: previewColor }}
              >
                <ChannelIcon type="webchat" size={24} className="text-white" />
              </button>
            </div>
          </div>

          {/* Install code */}
          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
              <Code size={14} />
              Script de instalação
            </label>
            <pre className="bg-dark-900 text-green-400 p-3 rounded-lg text-xs overflow-x-auto">
              {activeWidget ? getInstallScript(activeWidget.id) : '<!-- Crie e salve um widget para gerar o script -->'}
            </pre>
            <p className="text-xs text-gray-400 mt-2">
              Cole este código antes do {'</body>'} no HTML do seu site.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
