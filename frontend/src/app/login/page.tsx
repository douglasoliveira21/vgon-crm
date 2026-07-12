'use client'

import Link from 'next/link'
import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/store/auth'
import toast from 'react-hot-toast'
import { Eye, EyeOff } from 'lucide-react'
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
    <main className="login-page">
      <style jsx global>{`
        @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&display=swap');

        .login-page {
          font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
          min-height: 100vh;
          width: 100vw;
          display: grid;
          grid-template-columns: 55% 45%;
          background: #050505;
          color: #fff;
          overflow: hidden;
        }

        @media (max-width: 1024px) {
          .login-page { grid-template-columns: 1fr; }
          .login-left { display: none; }
        }

        .login-left {
          position: relative;
          display: flex;
          flex-direction: column;
          justify-content: center;
          padding: 4rem 5rem;
          overflow: hidden;
        }

        .login-left::before {
          content: '';
          position: absolute;
          inset: 0;
          background: radial-gradient(ellipse at 50% 50%, rgba(255,255,255,0.03) 0%, transparent 70%);
          pointer-events: none;
        }

        .login-left::after {
          content: '';
          position: absolute;
          inset: 0;
          background-image: url('/assets/images/bg-world-map.svg');
          background-size: 90%;
          background-position: center;
          background-repeat: no-repeat;
          opacity: 0.06;
          pointer-events: none;
        }

        .login-glow {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          width: 600px;
          height: 600px;
          background: radial-gradient(circle, rgba(255,255,255,0.04) 0%, transparent 70%);
          border-radius: 50%;
          pointer-events: none;
        }

        .login-particles {
          position: absolute;
          inset: 0;
          background-image: url("data:image/svg+xml,%3Csvg width='400' height='400' xmlns='http://www.w3.org/2000/svg'%3E%3Cdefs%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3C/defs%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='0.015'/%3E%3C/svg%3E");
          pointer-events: none;
        }

        .login-left-content {
          position: relative;
          z-index: 1;
        }

        .login-logo {
          height: 44px;
          width: auto;
          margin-bottom: 3rem;
          opacity: 0.95;
        }

        .login-headline {
          font-size: 2.8rem;
          font-weight: 300;
          line-height: 1.2;
          letter-spacing: -0.03em;
          margin-bottom: 1.5rem;
        }

        .login-headline strong {
          font-weight: 700;
        }

        .login-subtitle {
          font-size: 1rem;
          line-height: 1.7;
          color: rgba(255,255,255,0.5);
          max-width: 480px;
          margin-bottom: 3rem;
        }

        .login-cards {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 12px;
        }

        @media (max-width: 1280px) {
          .login-cards { grid-template-columns: repeat(2, 1fr); }
        }

        .login-card {
          background: rgba(255,255,255,0.03);
          border: 1px solid rgba(255,255,255,0.06);
          border-radius: 12px;
          padding: 20px 16px;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 10px;
          backdrop-filter: blur(8px);
          transition: all 300ms ease;
        }

        .login-card:hover {
          background: rgba(255,255,255,0.06);
          border-color: rgba(255,255,255,0.12);
          transform: translateY(-2px);
        }

        .login-card img {
          width: 22px;
          height: 22px;
          opacity: 0.7;
          filter: invert(1);
        }

        .login-card span {
          font-size: 12px;
          font-weight: 500;
          color: rgba(255,255,255,0.6);
          text-align: center;
        }

        .login-right {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 3rem 2.5rem;
          background: #0a0a0a;
          border-left: 1px solid rgba(255,255,255,0.04);
          position: relative;
        }

        .login-right::before {
          content: '';
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          height: 1px;
          background: linear-gradient(90deg, transparent, rgba(255,255,255,0.06), transparent);
        }

        .login-form-container {
          width: 100%;
          max-width: 380px;
        }

        .login-form-title {
          font-size: 1.5rem;
          font-weight: 600;
          letter-spacing: -0.02em;
          margin-bottom: 0.5rem;
          color: #fff;
        }

        .login-form-subtitle {
          font-size: 0.875rem;
          color: rgba(255,255,255,0.4);
          margin-bottom: 2rem;
          line-height: 1.5;
        }

        .login-field {
          margin-bottom: 1.25rem;
        }

        .login-label {
          display: block;
          font-size: 0.8125rem;
          font-weight: 500;
          color: rgba(255,255,255,0.6);
          margin-bottom: 0.5rem;
        }

        .login-input {
          width: 100%;
          padding: 12px 16px;
          background: rgba(255,255,255,0.04);
          border: 1px solid rgba(255,255,255,0.08);
          border-radius: 10px;
          color: #fff;
          font-size: 0.9375rem;
          outline: none;
          transition: all 300ms ease;
          box-sizing: border-box;
        }

        .login-input::placeholder {
          color: rgba(255,255,255,0.25);
        }

        .login-input:focus {
          border-color: rgba(255,255,255,0.2);
          background: rgba(255,255,255,0.06);
          box-shadow: 0 0 0 3px rgba(255,255,255,0.03);
        }

        .login-password-wrapper {
          position: relative;
        }

        .login-password-wrapper input {
          padding-right: 48px;
        }

        .login-password-toggle {
          position: absolute;
          right: 14px;
          top: 50%;
          transform: translateY(-50%);
          background: none;
          border: none;
          color: rgba(255,255,255,0.35);
          cursor: pointer;
          padding: 4px;
          transition: color 300ms;
        }

        .login-password-toggle:hover {
          color: rgba(255,255,255,0.7);
        }

        .login-forgot {
          display: block;
          text-align: right;
          font-size: 0.8125rem;
          color: rgba(255,255,255,0.4);
          margin-top: -0.5rem;
          margin-bottom: 1.5rem;
          cursor: pointer;
          transition: color 300ms;
          background: none;
          border: none;
          margin-left: auto;
        }

        .login-forgot:hover {
          color: rgba(255,255,255,0.8);
        }

        .login-submit {
          width: 100%;
          padding: 13px;
          background: #fff;
          color: #000;
          border: none;
          border-radius: 10px;
          font-size: 0.9375rem;
          font-weight: 600;
          cursor: pointer;
          transition: all 300ms ease;
          letter-spacing: -0.01em;
        }

        .login-submit:hover:not(:disabled) {
          background: #e5e5e5;
          transform: translateY(-1px);
          box-shadow: 0 4px 20px rgba(255,255,255,0.1);
        }

        .login-submit:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .login-back-btn {
          width: 100%;
          padding: 13px;
          background: transparent;
          color: rgba(255,255,255,0.6);
          border: 1px solid rgba(255,255,255,0.1);
          border-radius: 10px;
          font-size: 0.9375rem;
          font-weight: 500;
          cursor: pointer;
          transition: all 300ms ease;
          margin-top: 12px;
        }

        .login-back-btn:hover {
          background: rgba(255,255,255,0.04);
          border-color: rgba(255,255,255,0.2);
        }

        .login-links {
          margin-top: 1.5rem;
          text-align: center;
          font-size: 0.75rem;
          color: rgba(255,255,255,0.3);
          line-height: 1.8;
        }

        .login-links a {
          color: rgba(255,255,255,0.5);
          text-decoration: none;
          transition: color 300ms;
        }

        .login-links a:hover {
          color: #fff;
        }

        .login-footer {
          margin-top: 3rem;
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 1.5rem;
          flex-wrap: wrap;
        }

        .login-footer-item {
          display: flex;
          align-items: center;
          gap: 6px;
          font-size: 0.6875rem;
          color: rgba(255,255,255,0.3);
          letter-spacing: 0.02em;
        }

        .login-footer-item img {
          width: 14px;
          height: 14px;
          opacity: 0.4;
          filter: invert(1);
        }
      `}</style>

      {/* Left Panel */}
      <section className="login-left">
        <div className="login-glow" />
        <div className="login-particles" />
        <div className="login-left-content">
          <img
            src="/assets/images/logo-vgon-negativo.png"
            alt="VGON"
            className="login-logo"
          />

          <h1 className="login-headline">
            Conecte tudo.<br />
            <strong>Atenda melhor.</strong><br />
            <strong>Transforme experiências.</strong>
          </h1>

          <p className="login-subtitle">
            Plataforma Omnichannel completa para integrar WhatsApp, E-mail, Website e chamadas em um único lugar.
          </p>

          <div className="login-cards">
            <div className="login-card">
              <img src="/assets/icons/email.svg" alt="E-mail" />
              <span>E-mail</span>
            </div>
            <div className="login-card">
              <img src="/assets/icons/whatsapp.svg" alt="WhatsApp" />
              <span>WhatsApp</span>
            </div>
            <div className="login-card">
              <img src="/assets/icons/website.svg" alt="Website" />
              <span>Website</span>
            </div>
            <div className="login-card">
              <img src="/assets/icons/phone.svg" alt="Ligações" />
              <span>Ligações</span>
            </div>
          </div>
        </div>
      </section>

      {/* Right Panel - Login Form */}
      <section className="login-right">
        <div className="login-form-container">
          <h2 className="login-form-title">
            {showForgotPassword ? 'Recuperar senha' : 'Bem-vindo de volta'}
          </h2>
          <p className="login-form-subtitle">
            {showForgotPassword
              ? 'Informe seu e-mail para receber um link seguro de redefinição.'
              : 'Faça login para acessar sua plataforma.'}
          </p>

          <form onSubmit={showForgotPassword ? handleForgotPassword : handleSubmit}>
            <div className="login-field">
              <label className="login-label">E-mail</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="login-input"
                placeholder="seu@email.com"
                required
              />
            </div>

            {!showForgotPassword && (
              <>
                <div className="login-field">
                  <label className="login-label">Senha</label>
                  <div className="login-password-wrapper">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      className="login-input"
                      placeholder="Digite sua senha"
                      required
                      minLength={8}
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="login-password-toggle"
                      aria-label={showPassword ? 'Ocultar senha' : 'Mostrar senha'}
                    >
                      {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                    </button>
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => setShowForgotPassword(true)}
                  className="login-forgot"
                >
                  Esqueceu sua senha?
                </button>
              </>
            )}

            <button
              type="submit"
              disabled={isLoading || isSendingReset}
              className="login-submit"
            >
              {showForgotPassword
                ? (isSendingReset ? 'Enviando...' : 'Enviar link de recuperação')
                : (isLoading ? 'Entrando...' : 'Entrar')}
            </button>

            {showForgotPassword && (
              <button
                type="button"
                onClick={() => setShowForgotPassword(false)}
                className="login-back-btn"
              >
                Voltar para o login
              </button>
            )}
          </form>

          <div className="login-links">
            <Link href="/termos-de-servico">Termos de Uso</Link>
            {' · '}
            <Link href="/politica-de-privacidade">Política de Privacidade</Link>
          </div>

          <div className="login-footer">
            <div className="login-footer-item">
              <img src="/assets/icons/security.svg" alt="" />
              <span>Ambiente Seguro</span>
            </div>
            <div className="login-footer-item">
              <img src="/assets/icons/shield.svg" alt="" />
              <span>Infraestrutura Estável</span>
            </div>
            <div className="login-footer-item">
              <img src="/assets/icons/support.svg" alt="" />
              <span>Suporte Humanizado</span>
            </div>
          </div>
        </div>
      </section>
    </main>
  )
}
