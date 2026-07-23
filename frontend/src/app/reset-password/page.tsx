'use client'

import Link from 'next/link'
import { Suspense, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import toast from 'react-hot-toast'
import { SafeImage } from '@/components/safe-image'
import { Eye, EyeOff } from 'lucide-react'
import api from '@/lib/api'

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<ResetPasswordShell loading />}>
      <ResetPasswordForm />
    </Suspense>
  )
}

function ResetPasswordForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const token = searchParams.get('token') || ''
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!token) {
      toast.error('Link de recuperacao invalido')
      return
    }
    if (password.length < 8) {
      toast.error('A senha precisa ter pelo menos 8 caracteres')
      return
    }
    if (password !== confirmPassword) {
      toast.error('As senhas nao conferem')
      return
    }

    setIsSubmitting(true)
    try {
      await api.post('/auth/reset-password', {
        token,
        new_password: password,
      })
      toast.success('Senha atualizada. Entre novamente.')
      router.push('/login')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Nao foi possivel redefinir a senha')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <ResetPasswordShell>
      <form onSubmit={handleSubmit} className="mt-7 space-y-5">
        <PasswordField
          label="Nova senha"
          value={password}
          onChange={setPassword}
          showPassword={showPassword}
          toggleShowPassword={() => setShowPassword((value) => !value)}
        />

        <PasswordField
          label="Confirmar senha"
          value={confirmPassword}
          onChange={setConfirmPassword}
          showPassword={showPassword}
          toggleShowPassword={() => setShowPassword((value) => !value)}
        />

        <button
          type="submit"
          disabled={isSubmitting || !token}
          className="w-full rounded-lg bg-primary-600 py-3 font-medium text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isSubmitting ? 'Salvando...' : 'Redefinir senha'}
        </button>

        <Link
          href="/login"
          className="block w-full rounded-lg border border-gray-200 bg-white py-3 text-center font-medium text-gray-700 transition-colors hover:bg-gray-50"
        >
          Voltar para o login
        </Link>
      </form>
    </ResetPasswordShell>
  )
}

function ResetPasswordShell({
  children,
  loading = false,
}: {
  children?: React.ReactNode
  loading?: boolean
}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-dark-900 px-6 py-10 text-white">
      <div className="w-full max-w-md">
        <div className="mb-8 flex justify-center">
          <SafeImage src="/logo-white.png" fallbackSrc="/assets/images/logo-vgon-negativo.png" alt="Logo do crmvgon" className="h-20 w-auto max-w-full object-contain" />
        </div>

        <div className="rounded-2xl border border-gray-100 bg-white p-8 text-gray-900 shadow-xl shadow-black/20">
          <h1 className="text-2xl font-bold text-gray-950">Criar nova senha</h1>
          <p className="mt-2 text-sm leading-6 text-gray-500">
            Digite uma nova senha para voltar a acessar o crmvgon com seguranca.
          </p>

          {loading ? (
            <div className="mt-7 h-40 animate-pulse rounded-xl bg-gray-100" />
          ) : (
            children
          )}
        </div>
      </div>
    </main>
  )
}

function PasswordField({
  label,
  value,
  onChange,
  showPassword,
  toggleShowPassword,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  showPassword: boolean
  toggleShowPassword: () => void
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700">{label}</label>
      <div className="relative">
        <input
          type={showPassword ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full rounded-lg border border-gray-200 bg-white px-4 py-3 pr-12 text-gray-950 outline-none transition-all placeholder:text-gray-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-500/20"
          placeholder="Minimo de 8 caracteres"
          required
          minLength={8}
        />
        <button
          type="button"
          onClick={toggleShowPassword}
          className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-700"
          aria-label={showPassword ? 'Ocultar senha' : 'Mostrar senha'}
        >
          {showPassword ? <EyeOff size={20} /> : <Eye size={20} />}
        </button>
      </div>
    </div>
  )
}
