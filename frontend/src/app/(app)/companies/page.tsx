'use client'

import { useEffect, useState } from 'react'
import api from '@/lib/api'
import toast from 'react-hot-toast'
import { Building2, Edit2, FileDown, Plus, Search, Trash2 } from 'lucide-react'

interface CustomerCompany {
  id: string
  name: string
  cnpj?: string
  trade_name?: string
  legal_name?: string
  email?: string
  phone?: string
  city?: string
  state?: string
  address?: string
  initial_response_sla_minutes: number
  resolution_sla_minutes: number
  is_active: boolean
}

const emptyForm = {
  name: '',
  cnpj: '',
  trade_name: '',
  legal_name: '',
  email: '',
  phone: '',
  city: '',
  state: '',
  address: '',
  initial_response_sla_minutes: 30,
  resolution_sla_minutes: 240,
  is_active: true,
}

function formatCNPJ(value: string) {
  const digits = value.replace(/\D/g, '').slice(0, 14)
  return digits
    .replace(/^(\d{2})(\d)/, '$1.$2')
    .replace(/^(\d{2})\.(\d{3})(\d)/, '$1.$2.$3')
    .replace(/\.(\d{3})(\d)/, '.$1/$2')
    .replace(/(\d{4})(\d)/, '$1-$2')
}

export default function CompaniesPage() {
  const [companies, setCompanies] = useState<CustomerCompany[]>([])
  const [search, setSearch] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<CustomerCompany | null>(null)

  useEffect(() => {
    fetchCompanies()
  }, [search])

  const fetchCompanies = async () => {
    const res = await api.get('/customer-companies', { params: { search } })
    setCompanies(res.data.companies || [])
  }

  const removeCompany = async (id: string) => {
    if (!confirm('Remover esta empresa? Os contatos vinculados ficarão sem empresa.')) return
    await api.delete(`/customer-companies/${id}`)
    toast.success('Empresa removida')
    fetchCompanies()
  }

  const exportReport = async () => {
    try {
      const response = await api.get('/customer-companies/sla-report.csv', { responseType: 'blob' })
      const url = window.URL.createObjectURL(new Blob([response.data], { type: 'text/csv;charset=utf-8;' }))
      const link = document.createElement('a')
      link.href = url
      link.download = `relatorio-sla-empresas-${new Date().toISOString().slice(0, 10)}.csv`
      link.click()
      window.URL.revokeObjectURL(url)
    } catch {
      toast.error('Erro ao exportar relatório')
    }
  }

  return (
    <div className="mx-auto max-w-7xl p-6">
      <div className="mb-6 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Empresas</h1>
          <p className="mt-1 text-sm text-gray-500">Cadastre empresas clientes, vincule contatos e defina SLA de atendimento.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={exportReport} className="btn-secondary">
            <FileDown size={18} />
            Exportar SLA
          </button>
          <button onClick={() => { setEditing(null); setShowForm(true) }} className="btn-primary">
            <Plus size={18} />
            Nova empresa
          </button>
        </div>
      </div>

      <div className="relative mb-6">
        <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input value={search} onChange={(e) => setSearch(e.target.value)} className="input pl-10" placeholder="Buscar por nome, CNPJ ou razão social..." />
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {companies.map((company) => (
          <div key={company.id} className="card p-5">
            <div className="mb-4 flex items-start justify-between gap-3">
              <div className="flex gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary-100 text-primary-700">
                  <Building2 size={20} />
                </div>
                <div>
                  <h2 className="font-semibold text-gray-900 dark:text-white">{company.name}</h2>
                  <p className="text-xs text-gray-500">{company.cnpj || 'Sem CNPJ'}</p>
                </div>
              </div>
              <div className="flex gap-1">
                <button onClick={() => { setEditing(company); setShowForm(true) }} className="p-1.5 text-gray-400 hover:text-primary-600 hover:bg-primary-50 rounded">
                  <Edit2 size={14} />
                </button>
                <button onClick={() => removeCompany(company.id)} className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded">
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
            <div className="space-y-2 text-sm text-gray-600">
              <p><strong>SLA inicial:</strong> {company.initial_response_sla_minutes} min</p>
              <p><strong>SLA resolução:</strong> {company.resolution_sla_minutes} min</p>
              {(company.city || company.state) && <p>{company.city} {company.state && `/${company.state}`}</p>}
            </div>
          </div>
        ))}
      </div>

      {companies.length === 0 && (
        <div className="card p-10 text-center text-gray-500">Nenhuma empresa cadastrada.</div>
      )}

      {showForm && (
        <CompanyModal
          company={editing}
          onClose={() => setShowForm(false)}
          onSaved={() => { setShowForm(false); fetchCompanies() }}
        />
      )}
    </div>
  )
}

function CompanyModal({ company, onClose, onSaved }: { company: CustomerCompany | null; onClose: () => void; onSaved: () => void }) {
  const [form, setForm] = useState<any>(company || emptyForm)
  const [saving, setSaving] = useState(false)
  const [lookupLoading, setLookupLoading] = useState(false)

  const lookupCNPJ = async () => {
    const digits = String(form.cnpj || '').replace(/\D/g, '')
    if (digits.length !== 14) {
      toast.error('Informe um CNPJ válido')
      return
    }
    setLookupLoading(true)
    try {
      const res = await api.get(`/customer-companies/lookup/${digits}`)
      setForm((prev: any) => ({ ...prev, ...res.data }))
      toast.success('CNPJ localizado')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao buscar CNPJ')
    } finally {
      setLookupLoading(false)
    }
  }

  const save = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      if (company) {
        await api.put(`/customer-companies/${company.id}`, form)
        toast.success('Empresa atualizada')
      } else {
        await api.post('/customer-companies', form)
        toast.success('Empresa criada')
      }
      onSaved()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Erro ao salvar empresa')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-2xl bg-white p-6 dark:bg-gray-900">
        <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">{company ? 'Editar empresa' : 'Nova empresa'}</h3>
        <form onSubmit={save} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">CNPJ</label>
              <input value={form.cnpj} onChange={(e) => setForm({ ...form, cnpj: formatCNPJ(e.target.value) })} className="input" placeholder="00.000.000/0000-00" />
            </div>
            <button type="button" onClick={lookupCNPJ} disabled={lookupLoading} className="btn-secondary self-end">
              {lookupLoading ? 'Buscando...' : 'Buscar CNPJ'}
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Nome da empresa" value={form.name} onChange={(v) => setForm({ ...form, name: v })} required />
            <Field label="Razão social" value={form.legal_name} onChange={(v) => setForm({ ...form, legal_name: v })} />
            <Field label="Nome fantasia" value={form.trade_name} onChange={(v) => setForm({ ...form, trade_name: v })} />
            <Field label="E-mail" value={form.email} onChange={(v) => setForm({ ...form, email: v })} />
            <Field label="Telefone" value={form.phone} onChange={(v) => setForm({ ...form, phone: v })} />
            <Field label="Cidade" value={form.city} onChange={(v) => setForm({ ...form, city: v })} />
            <Field label="UF" value={form.state} onChange={(v) => setForm({ ...form, state: v.toUpperCase().slice(0, 2) })} />
            <Field label="Endereço" value={form.address} onChange={(v) => setForm({ ...form, address: v })} />
            <Field label="SLA atendimento inicial (min)" type="number" value={form.initial_response_sla_minutes} onChange={(v) => setForm({ ...form, initial_response_sla_minutes: Number(v) })} />
            <Field label="SLA resolução do chat (min)" type="number" value={form.resolution_sla_minutes} onChange={(v) => setForm({ ...form, resolution_sla_minutes: Number(v) })} />
          </div>
          <div className="flex gap-3 pt-4">
            <button type="button" onClick={onClose} className="btn-secondary flex-1">Cancelar</button>
            <button type="submit" disabled={saving} className="btn-primary flex-1">{saving ? 'Salvando...' : 'Salvar'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, value, onChange, type = 'text', required = false }: { label: string; value: any; onChange: (v: string) => void; type?: string; required?: boolean }) {
  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-gray-700">{label}</label>
      <input type={type} value={value || ''} onChange={(e) => onChange(e.target.value)} className="input" required={required} />
    </div>
  )
}
