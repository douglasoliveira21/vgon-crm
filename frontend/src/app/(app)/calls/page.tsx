'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import { Phone, PhoneIncoming, PhoneOutgoing, PhoneMissed, Clock } from 'lucide-react'

export default function CallsPage() {
  const [calls, setCalls] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(false)
    // calls are registered manually or via future integrations
  }, [])

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Chamadas</h1>
          <p className="text-gray-500 mt-1">Histórico e registro de chamadas de áudio</p>
        </div>
        <button className="btn-primary">
          <Phone size={18} />
          Registrar chamada
        </button>
      </div>

      <div className="card p-12 text-center">
        <Phone size={40} className="text-gray-300 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">Módulo de Chamadas</h3>
        <p className="text-gray-500 text-sm max-w-md mx-auto">
          Registre chamadas realizadas, perdidas e recebidas. Acompanhe duração, 
          atendente responsável e métricas de chamadas.
        </p>
        <div className="flex items-center justify-center gap-6 mt-8">
          <div className="text-center">
            <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <PhoneOutgoing size={20} className="text-green-600" />
            </div>
            <p className="text-xs text-gray-500">Realizadas</p>
            <p className="font-bold text-gray-900">0</p>
          </div>
          <div className="text-center">
            <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <PhoneIncoming size={20} className="text-blue-600" />
            </div>
            <p className="text-xs text-gray-500">Recebidas</p>
            <p className="font-bold text-gray-900">0</p>
          </div>
          <div className="text-center">
            <div className="w-12 h-12 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <PhoneMissed size={20} className="text-red-600" />
            </div>
            <p className="text-xs text-gray-500">Perdidas</p>
            <p className="font-bold text-gray-900">0</p>
          </div>
          <div className="text-center">
            <div className="w-12 h-12 bg-purple-100 rounded-full flex items-center justify-center mx-auto mb-2">
              <Clock size={20} className="text-purple-600" />
            </div>
            <p className="text-xs text-gray-500">Duração média</p>
            <p className="font-bold text-gray-900">0min</p>
          </div>
        </div>
      </div>
    </div>
  )
}
