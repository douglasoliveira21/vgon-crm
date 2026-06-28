'use client'

import Link from 'next/link'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import toast from 'react-hot-toast'
import { Eye, EyeOff, Mail, MessageSquare, ShieldCheck, Users } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()
  const { login, isLoading } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      await login(email, password)
      router.push('/dashboard')
      toast.success('Login realizado!')
    } catch (error: any) {
      toast.error(error.message)
    }
  }

  return (
    <main className="min-h-screen bg-dark-900 text-white">
      <div className="grid min-h-screen lg:grid-cols-[1.15fr_0.85fr]">
        <section className="flex items-center px-6 py-10 sm:px-10 lg:px-16">
          <div className="w-full max-w-3xl">
            <img src="/logo-white.png" alt="Logo do crmvgon" className="mb-10 h-20 w-auto max-w-full object-contain" />

            <p className="text-sm font-semibold uppercase tracking-wide text-primary-300">Aplicativo crmvgon</p>
            <h1 className="mt-4 text-4xl font-bold tracking-tight sm:text-5xl">
              CRM de atendimento para conversas, contatos e e-mails.
            </h1>
            <p className="mt-5 max-w-2xl text-base leading-7 text-gray-300">
              O crmvgon centraliza atendimentos em um painel seguro para equipes acompanharem
              clientes, organizarem contatos e responderem mensagens. Usuários autorizados podem
              conectar Gmail ou Outlook para ler e-mails recebidos e enviar respostas dentro do CRM.
            </p>

            <div className="mt-8 grid gap-3 sm:grid-cols-2">
              <InfoItem icon={<MessageSquare size={18} />} text="Conversas e histórico de atendimento" />
              <InfoItem icon={<Mail size={18} />} text="Integração com Gmail e Outlook via OAuth" />
              <InfoItem icon={<Users size={18} />} text="Gestão de contatos, equipes e responsáveis" />
              <InfoItem icon={<ShieldCheck size={18} />} text="Uso autorizado, com política de privacidade pública" />
            </div>

            <div className="mt-8 flex flex-wrap gap-4 text-sm">
              <Link href="/politica-de-privacidade" className="font-medium text-primary-300 hover:text-primary-200">
                Política de Privacidade
              </Link>
              <Link href="/termos-de-servico" className="font-medium text-primary-300 hover:text-primary-200">
                Termos de Serviço
              </Link>
            </div>
          </div>
        </section>

        <section className="flex items-center justify-center bg-white px-6 py-10 text-gray-900 sm:px-10">
          <div className="w-full max-w-md">
            <div className="mb-8 rounded-2xl bg-white p-4 shadow-sm ring-1 ring-gray-100">
              <img src="/logo-black.png" alt="Logo do crmvgon" className="h-16 w-auto max-w-full object-contain" />
            </div>

            <div className="rounded-2xl border border-gray-100 bg-white p-8 shadow-xl shadow-gray-200/70">
              <h2 className="text-2xl font-bold text-gray-950">Entrar no crmvgon</h2>
              <p className="mt-2 text-sm leading-6 text-gray-500">
                Acesse sua área segura para gerenciar conversas, contatos e canais de atendimento.
              </p>

              <form onSubmit={handleSubmit} className="mt-7 space-y-5">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">
                    E-mail
                  </label>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full rounded-lg border border-gray-200 bg-white px-4 py-3 text-gray-950 outline-none transition-all placeholder:text-gray-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                    placeholder="seu@email.com"
                    required
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">
                    Senha
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full rounded-lg border border-gray-200 bg-white px-4 py-3 pr-12 text-gray-950 outline-none transition-all placeholder:text-gray-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
                      placeholder="Digite sua senha"
                      required
                      minLength={8}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700"
                      aria-label={showPassword ? 'Ocultar senha' : 'Mostrar senha'}
                    >
                      {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
                    </button>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={isLoading}
                  className="w-full rounded-lg bg-primary-600 py-3 font-medium text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {isLoading ? 'Entrando...' : 'Entrar'}
                </button>
              </form>

              <p className="mt-6 text-center text-xs leading-5 text-gray-500">
                Ao acessar, você concorda com os{' '}
                <Link href="/termos-de-servico" className="font-medium text-primary-600 hover:text-primary-700">
                  Termos de Serviço
                </Link>{' '}
                e com a{' '}
                <Link href="/politica-de-privacidade" className="font-medium text-primary-600 hover:text-primary-700">
                  Política de Privacidade
                </Link>
                .
              </p>
            </div>
          </div>
        </section>
      </div>
    </main>
  )
}

function InfoItem({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3 text-sm text-gray-200">
      <span className="text-primary-300">{icon}</span>
      <span>{text}</span>
    </div>
  )
}
