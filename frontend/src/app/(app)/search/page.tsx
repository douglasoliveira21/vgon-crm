'use client'

import { useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import Link from 'next/link'
import api from '@/lib/api'
import { Building2, Loader2, MessageSquare, Search, SearchX, Users } from 'lucide-react'

interface GlobalSearchResponse {
  contacts: Array<{ id: string; name: string; phone?: string; email?: string; company_name?: string; origin?: string }>
  conversations: Array<{ id: string; contact_name: string; phone?: string; email?: string; subject?: string; preview?: string; channel_name?: string; status?: string; last_message_at?: string }>
  companies: Array<{ id: string; name: string; cnpj?: string; trade_name?: string; legal_name?: string; email?: string; phone?: string; city?: string; state?: string }>
}

const emptyResults: GlobalSearchResponse = { contacts: [], conversations: [], companies: [] }

export default function GlobalSearchPage() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<GlobalSearchResponse>(emptyResults)
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)

  const totalResults = useMemo(
    () => results.contacts.length + results.conversations.length + results.companies.length,
    [results]
  )

  useEffect(() => {
    const trimmed = query.trim()
    if (trimmed.length < 2) {
      setResults(emptyResults)
      setSearched(false)
      return
    }

    const timer = setTimeout(() => {
      runSearch(trimmed)
    }, 350)

    return () => clearTimeout(timer)
  }, [query])

  const runSearch = async (value = query.trim()) => {
    if (value.length < 2) return
    setLoading(true)
    setSearched(true)
    try {
      const response = await api.get('/search', { params: { q: value } })
      setResults({
        contacts: response.data.contacts || [],
        conversations: response.data.conversations || [],
        companies: response.data.companies || [],
      })
    } catch {
      setResults(emptyResults)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mx-auto max-w-6xl p-4 sm:p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Busca global</h1>
        <p className="text-gray-500 mt-1">Encontre contatos, conversas, empresas, e-mails e telefones em um só lugar.</p>
      </div>

      <div className="card p-4 mb-6">
        <div className="relative">
          <Search size={20} className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') runSearch()
            }}
            className="input pl-12 pr-12 text-base"
            autoFocus
            placeholder="Buscar por nome, telefone, e-mail, empresa, CNPJ ou trecho da conversa..."
          />
          {loading && <Loader2 size={20} className="absolute right-4 top-1/2 -translate-y-1/2 animate-spin text-primary-600" />}
        </div>
      </div>

      {searched && !loading && (
        <p className="text-sm text-gray-500 mb-4">{totalResults} resultado(s) encontrado(s)</p>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <ResultSection title="Contatos" icon={<Users size={18} />} count={results.contacts.length}>
          {results.contacts.map((contact) => (
            <Link key={contact.id} href={`/contacts?search=${encodeURIComponent(contact.phone || contact.email || contact.name)}`} className="block rounded-lg border border-gray-100 p-3 hover:bg-gray-50">
              <p className="text-sm font-medium text-gray-900">{contact.name}</p>
              <p className="text-xs text-gray-500">{contact.phone || 'Sem telefone'} {contact.email ? `| ${contact.email}` : ''}</p>
              {contact.company_name && <p className="text-xs text-gray-400 mt-1">{contact.company_name}</p>}
            </Link>
          ))}
        </ResultSection>

        <ResultSection title="Conversas" icon={<MessageSquare size={18} />} count={results.conversations.length}>
          {results.conversations.map((conversation) => (
            <Link key={conversation.id} href={`/conversations?conversation=${conversation.id}`} className="block rounded-lg border border-gray-100 p-3 hover:bg-gray-50">
              <p className="text-sm font-medium text-gray-900">{conversation.contact_name}</p>
              <p className="text-xs text-gray-500">{conversation.channel_name || 'Sem canal'} | {conversation.status || 'open'}</p>
              <p className="text-xs text-gray-400 mt-1 line-clamp-2">{conversation.preview || conversation.subject || 'Sem prévia'}</p>
            </Link>
          ))}
        </ResultSection>

        <ResultSection title="Empresas" icon={<Building2 size={18} />} count={results.companies.length}>
          {results.companies.map((company) => (
            <Link key={company.id} href={`/companies?search=${encodeURIComponent(company.cnpj || company.name)}`} className="block rounded-lg border border-gray-100 p-3 hover:bg-gray-50">
              <p className="text-sm font-medium text-gray-900">{company.name}</p>
              <p className="text-xs text-gray-500">{company.cnpj || 'Sem CNPJ'} {company.phone ? `| ${company.phone}` : ''}</p>
              {(company.city || company.state) && <p className="text-xs text-gray-400 mt-1">{company.city} {company.state}</p>}
            </Link>
          ))}
        </ResultSection>
      </div>

      {!loading && searched && totalResults === 0 && (
        <div className="card p-12 text-center mt-6">
          <SearchX size={36} className="mx-auto mb-3 text-gray-300" />
          <p className="text-gray-500">Nenhum resultado encontrado para essa busca.</p>
        </div>
      )}
    </div>
  )
}

function ResultSection({ title, icon, count, children }: { title: string; icon: ReactNode; count: number; children: ReactNode }) {
  return (
    <div className="card p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2 text-gray-900">
          {icon}
          <h2 className="font-semibold">{title}</h2>
        </div>
        <span className="text-xs text-gray-400">{count}</span>
      </div>
      <div className="space-y-2">
        {count > 0 ? children : <p className="text-sm text-gray-400 py-4">Sem resultados.</p>}
      </div>
    </div>
  )
}
