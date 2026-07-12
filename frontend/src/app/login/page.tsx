'use client'

import Link from 'next/link'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import toast from 'react-hot-toast'
import { Eye, EyeOff, Mail, MessageCircle, Globe, Lock, Shield, HeadphonesIcon } from 'lucide-react'
import api from '@/lib/api'

export default function LoginPage() {
  const router = useRouter()
  const { login, isLoading } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [showForgotPassword, setShowForgotPassword] = useState(false)
  const [isSendingReset, setIsSendingReset] = useState(false)

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

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSendingReset(true)
    try {
      await api.post('/auth/forgot-password', { email })
      toast.success('Se o e-mail existir, enviamos o link de recuperação.')
      setShowForgotPassword(false)
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Não foi possível enviar o e-mail de recuperação')
    } finally {
      setIsSendingReset(false)
    }
  }

  return (
    <div className="min-h-screen w-full bg-[#050505] text-white flex flex-col overflow-x-hidden">
      {/* Main Content */}
      <div className="flex-1 grid grid-cols-1 lg:grid-cols-[55%_45%]">

        {/* Left Panel - Brand */}
        <section className="relative hidden lg:flex flex-col justify-center px-12 xl:px-20 py-16 overflow-hidden">
          {/* Video Background */}
          <video
            autoPlay
            loop
            muted
            playsInline
            className="absolute inset-0 w-full h-full object-cover opacity-50 pointer-events-none"
          >
            <source src="https://vgon.com.br/wp-content/uploads/2026/07/5925-187109675_medium.mp4" type="video/mp4" />
          </video>
          {/* Dark overlay on top of video */}
          <div className="absolute inset-0 bg-[#050505]/40 pointer-events-none" />

          {/* Content */}
          <div className="relative z-10">
            <img
              src="/assets/images/logo-vgon-negativo.png"
              alt="VGON"
              className="h-auto w-[clamp(180px,18vw,310px)] object-contain mb-12 opacity-95"
            />

            <h1 className="text-[clamp(2rem,4vw,3.2rem)] font-light leading-[1.15] tracking-[-0.03em] mb-6">
              Conecte tudo.<br />
              <span className="font-bold">Atenda melhor.</span><br />
              <span className="font-bold">Transforme experiências.</span>
            </h1>

            <p className="text-base leading-7 text-white/50 max-w-[480px] mb-12">
              Plataforma Omnichannel completa para integrar WhatsApp, E-mail e Website em um único lugar.
            </p>

            {/* Channel Cards */}
            <div className="grid grid-cols-3 gap-3">
              <ChannelCard icon={<Mail size={20} />} label="E-mail" />
              <ChannelCard icon={<MessageCircle size={20} />} label="WhatsApp" />
              <ChannelCard icon={<Globe size={20} />} label="Website" />
            </div>
          </div>
        </section>

        {/* Right Panel - Form */}
        <section className="flex items-center justify-center px-6 sm:px-12 lg:px-16 py-12 bg-[#0a0a0a] lg:border-l lg:border-white/[0.04]">
          <div className="w-full max-w-[400px]">
            {/* Mobile Logo */}
            <div className="lg:hidden mb-10">
              <img
                src="/assets/images/logo-vgon-negativo.png"
                alt="VGON"
                className="h-auto w-[180px] object-contain opacity-95"
              />
            </div>

            <h2 className="text-2xl font-semibold tracking-[-0.02em] text-white mb-2">
              {showForgotPassword ? 'Recuperar senha' : 'Bem-vindo de volta'}
            </h2>
            <p className="text-sm text-white/40 mb-8 leading-relaxed">
              {showForgotPassword
                ? 'Informe seu e-mail para receber um link seguro de redefinição.'
                : 'Faça login para acessar sua plataforma.'}
            </p>

            <form onSubmit={showForgotPassword ? handleForgotPassword : handleSubmit} className="space-y-5">
              {/* Email Field */}
              <div>
                <label className="block text-[13px] font-medium text-white/60 mb-2">E-mail</label>
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full h-[52px] px-4 bg-white/[0.04] border border-white/[0.08] rounded-[10px] text-white text-[15px] placeholder:text-white/25 outline-none transition-all duration-300 focus:border-white/20 focus:bg-white/[0.06] focus:shadow-[0_0_0_3px_rgba(255,255,255,0.03)]"
                  placeholder="seu@email.com"
                  required
                />
              </div>

              {/* Password Field */}
              {!showForgotPassword && (
                <div>
                  <label className="block text-[13px] font-medium text-white/60 mb-2">Senha</label>
                  <div className="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="w-full h-[52px] px-4 pr-12 bg-white/[0.04] border border-white/[0.08] rounded-[10px] text-white text-[15px] placeholder:text-white/25 outline-none transition-all duration-300 focus:border-white/20 focus:bg-white/[0.06] focus:shadow-[0_0_0_3px_rgba(255,255,255,0.03)]"
                      placeholder="Digite sua senha"
                      required
                      minLength={8}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-4 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/70 transition-colors duration-300"
                      aria-label={showPassword ? 'Ocultar senha' : 'Mostrar senha'}
                    >
                      {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                    </button>
                  </div>
                </div>
              )}

              {/* Forgot Password Link */}
              {!showForgotPassword && (
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={() => setShowForgotPassword(true)}
                    className="text-[13px] text-white/40 hover:text-white/80 transition-colors duration-300"
                  >
                    Esqueceu sua senha?
                  </button>
                </div>
              )}

              {/* Submit Button */}
              <button
                type="submit"
                disabled={isLoading || isSendingReset}
                className="w-full h-[52px] bg-white text-[#050505] rounded-[10px] text-[15px] font-semibold tracking-[-0.01em] transition-all duration-300 hover:bg-[#e5e5e5] hover:-translate-y-[1px] hover:shadow-[0_4px_20px_rgba(255,255,255,0.1)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-none"
              >
                {showForgotPassword
                  ? (isSendingReset ? 'Enviando...' : 'Enviar link de recuperação')
                  : (isLoading ? 'Entrando...' : 'Entrar')}
              </button>

              {/* Back Button (Forgot Password mode) */}
              {showForgotPassword && (
                <button
                  type="button"
                  onClick={() => setShowForgotPassword(false)}
                  className="w-full h-[52px] bg-transparent border border-white/10 text-white/60 rounded-[10px] text-[15px] font-medium transition-all duration-300 hover:bg-white/[0.04] hover:border-white/20"
                >
                  Voltar para o login
                </button>
              )}
            </form>

            {/* Legal Links */}
            <div className="mt-6 text-center text-xs text-white/30 space-x-3">
              <Link href="/termos-de-servico" className="hover:text-white/70 transition-colors duration-300">
                Termos de Uso
              </Link>
              <span className="text-white/15">·</span>
              <Link href="/politica-de-privacidade" className="hover:text-white/70 transition-colors duration-300">
                Política de Privacidade
              </Link>
            </div>

            {/* Footer Badges */}
            <div className="mt-10 flex items-center justify-center gap-5 flex-wrap">
              <FooterBadge icon={<Lock size={13} />} text="Ambiente Seguro" />
              <FooterBadge icon={<Shield size={13} />} text="Infraestrutura Estável" />
              <FooterBadge icon={<HeadphonesIcon size={13} />} text="Suporte Humanizado" />
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}

function ChannelCard({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div className="flex flex-col items-center gap-2.5 py-5 px-3 bg-white/[0.03] border border-white/[0.06] rounded-xl backdrop-blur-sm transition-all duration-300 hover:bg-white/[0.06] hover:border-white/[0.12] hover:-translate-y-0.5">
      <span className="text-white/60">{icon}</span>
      <span className="text-[11px] font-medium text-white/50">{label}</span>
    </div>
  )
}

function FooterBadge({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="flex items-center gap-1.5 text-[11px] text-white/30">
      <span className="text-white/40">{icon}</span>
      <span>{text}</span>
    </div>
  )
}
