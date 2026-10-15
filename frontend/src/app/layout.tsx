import type { Metadata } from 'next'
import { GeistSans } from 'geist/font/sans'
import { GeistMono } from 'geist/font/mono'
import './globals.css'
import { Providers } from '@/components/providers'
import { AppLoader } from '@/components/app-loader'

export const metadata: Metadata = {
  title: 'crmvgon - CRM de atendimento',
  description: 'crmvgon centraliza conversas, contatos, Gmail, Outlook e canais de atendimento em um CRM.',
  icons: {
    icon: '/favicon.png',
    shortcut: '/favicon.png',
    apple: '/favicon.png',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="pt-BR" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className={GeistSans.className}>
        <Providers>
          <AppLoader />
          {children}
        </Providers>
      </body>
    </html>
  )
}
