'use client'

import { useEffect, useState, useRef } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import {
  Play,
  Pause,
  Trash2,
  Download,
  Search,
  PhoneIncoming,
  PhoneOutgoing,
  Calendar,
  Clock,
  ArrowLeft,
  Mic,
} from 'lucide-react'
import { useRouter } from 'next/navigation'

interface Recording {
  id: string
  from_number: string
  to_number: string
  direction: string
  duration: number
  recording_url: string
  created_at: string
  contact_name: string
}

export default function RecordingsPage() {
  const router = useRouter()
  const [recordings, setRecordings] = useState<Recording[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [playingId, setPlayingId] = useState<string | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    fetchRecordings()
  }, [])

  const fetchRecordings = async () => {
    setLoading(true)
    try {
      const params: any = { limit: 100 }
      if (search) params.search = search
      if (dateFrom) params.date_from = dateFrom
      if (dateTo) params.date_to = dateTo

      const res = await api.get('/telephony/recordings', { params })
      setRecordings(res.data.recordings || [])
    } catch {
      toast.error('Erro ao carregar gravações')
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    fetchRecordings()
  }

  const handlePlay = (recording: Recording) => {
    if (playingId === recording.id) {
      // Stop playing
      if (audioRef.current) {
        audioRef.current.pause()
        audioRef.current.currentTime = 0
      }
      setPlayingId(null)
      return
    }

    // Play new recording
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001'
    const url = `${apiUrl}/api/telephony/recordings/${recording.id}/play`
    const token = localStorage.getItem('access_token')

    if (audioRef.current) {
      audioRef.current.pause()
    }

    const audio = new Audio()
    audio.src = url
    // Add auth header via fetch and blob
    fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.blob())
      .then(blob => {
        const objectUrl = URL.createObjectURL(blob)
        audio.src = objectUrl
        audio.play()
        audioRef.current = audio
        setPlayingId(recording.id)

        audio.onended = () => {
          setPlayingId(null)
          URL.revokeObjectURL(objectUrl)
        }
      })
      .catch(() => {
        toast.error('Erro ao reproduzir gravação')
      })
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Deseja realmente excluir esta gravação?')) return
    try {
      await api.delete(`/telephony/recordings/${id}`)
      toast.success('Gravação excluída')
      setRecordings(prev => prev.filter(r => r.id !== id))
      if (playingId === id && audioRef.current) {
        audioRef.current.pause()
        setPlayingId(null)
      }
    } catch {
      toast.error('Erro ao excluir gravação')
    }
  }

  const handleDownload = (recording: Recording) => {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001'
    const url = `${apiUrl}/api/telephony/recordings/${recording.id}/play`
    const token = localStorage.getItem('access_token')

    fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      .then(res => res.blob())
      .then(blob => {
        const objectUrl = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = objectUrl
        a.download = `recording-${recording.id}.wav`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(objectUrl)
      })
      .catch(() => {
        toast.error('Erro ao baixar gravação')
      })
  }

  const formatDuration = (seconds: number) => {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('pt-BR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push('/calls')} className="p-2 hover:bg-gray-100 rounded-lg">
            <ArrowLeft size={20} className="text-gray-600" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Gravações</h1>
            <p className="text-sm text-gray-500">{recordings.length} gravação(ões) encontrada(s)</p>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="card p-4 mb-6">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-xs font-medium text-gray-700 mb-1">Buscar</label>
            <div className="relative">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleSearch() }}
                placeholder="Número ou nome..."
                className="input pl-9"
              />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1">Data início</label>
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              className="input"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-1">Data fim</label>
            <input
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              className="input"
            />
          </div>
          <button onClick={handleSearch} className="btn-primary text-sm">
            <Search size={16} /> Filtrar
          </button>
        </div>
      </div>

      {/* Recordings list */}
      {loading ? (
        <div className="text-center py-12">
          <div className="animate-spin w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full mx-auto mb-4" />
          <p className="text-gray-500">Carregando gravações...</p>
        </div>
      ) : recordings.length === 0 ? (
        <div className="text-center py-16">
          <Mic size={48} className="text-gray-300 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-700 mb-1">Nenhuma gravação encontrada</h3>
          <p className="text-sm text-gray-500">As gravações de chamadas aparecerão aqui quando disponíveis.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {recordings.map((recording) => (
            <div key={recording.id} className="card p-4 flex items-center gap-4 hover:shadow-md transition-shadow">
              {/* Direction icon */}
              <div className={`p-2 rounded-full ${recording.direction === 'inbound' ? 'bg-blue-100' : 'bg-green-100'}`}>
                {recording.direction === 'inbound' ? (
                  <PhoneIncoming size={18} className="text-blue-600" />
                ) : (
                  <PhoneOutgoing size={18} className="text-green-600" />
                )}
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium text-gray-900 truncate">
                    {recording.contact_name || (recording.direction === 'inbound' ? recording.from_number : recording.to_number)}
                  </p>
                  {recording.contact_name && (
                    <span className="text-xs text-gray-400">
                      {recording.direction === 'inbound' ? recording.from_number : recording.to_number}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-500 mt-0.5">
                  <span className="flex items-center gap-1">
                    <Calendar size={12} />
                    {formatDate(recording.created_at)}
                  </span>
                  <span className="flex items-center gap-1">
                    <Clock size={12} />
                    {formatDuration(recording.duration)}
                  </span>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${recording.direction === 'inbound' ? 'bg-blue-50 text-blue-700' : 'bg-green-50 text-green-700'}`}>
                    {recording.direction === 'inbound' ? 'Recebida' : 'Realizada'}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-1">
                <button
                  onClick={() => handlePlay(recording)}
                  className={`p-2 rounded-lg transition-colors ${playingId === recording.id ? 'bg-primary-100 text-primary-700' : 'hover:bg-gray-100 text-gray-600'}`}
                  title={playingId === recording.id ? 'Parar' : 'Reproduzir'}
                >
                  {playingId === recording.id ? <Pause size={18} /> : <Play size={18} />}
                </button>
                <button
                  onClick={() => handleDownload(recording)}
                  className="p-2 rounded-lg hover:bg-gray-100 text-gray-600 transition-colors"
                  title="Download"
                >
                  <Download size={18} />
                </button>
                <button
                  onClick={() => handleDelete(recording.id)}
                  className="p-2 rounded-lg hover:bg-red-50 text-gray-400 hover:text-red-600 transition-colors"
                  title="Excluir"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
