import Link from 'next/link'

export const metadata = {
  title: 'Política de Privacidade | VGON',
  description: 'Política de privacidade da plataforma VGON CRM.',
}

export default function PrivacyPolicyPage() {
  return (
    <main className="min-h-screen bg-[#050505] text-white">
      {/* Header */}
      <header className="border-b border-white/[0.06]">
        <div className="mx-auto max-w-5xl px-6 py-6 flex items-center justify-between">
          <Link href="/login">
            <img src="/assets/images/logo-vgon-negativo.png" alt="VGON" className="h-8 w-auto object-contain opacity-90" />
          </Link>
          <Link href="/login" className="text-sm text-white/40 hover:text-white/80 transition-colors duration-300">
            ← Voltar ao login
          </Link>
        </div>
      </header>

      {/* Content */}
      <div className="mx-auto max-w-4xl px-6 py-16">
        <div className="mb-12">
          <h1 className="text-3xl font-bold tracking-[-0.02em]">Política de Privacidade</h1>
          <p className="mt-3 text-sm text-white/40">Última atualização: 28 de junho de 2026</p>
        </div>

        <div className="space-y-10 text-[15px] leading-7 text-white/70">
          <p>
            Esta Política de Privacidade explica como o VGON CRM coleta, usa, armazena e protege
            informações dos usuários e dos contatos atendidos pela plataforma.
          </p>

          <Section title="1. Dados que coletamos">
            Podemos coletar dados de cadastro, como nome, e-mail, telefone, empresa, credenciais
            de acesso, registros de atendimento, mensagens, contatos, histórico de conversas e
            configurações de canais conectados ao CRM.
          </Section>

          <Section title="2. Integração com e-mail, Gmail e Outlook">
            Quando o usuário conecta uma conta de e-mail ao VGON CRM, o sistema pode solicitar
            permissões para ler mensagens recebidas e enviar respostas em nome da conta conectada.
            Essas permissões são usadas somente para exibir e responder e-mails dentro das
            conversas do CRM.
          </Section>

          <Section title="3. Uso de dados do Google">
            O uso e a transferência de informações recebidas das APIs do Google pelo VGON CRM
            obedecem à Política de Dados de Usuário dos Serviços de API do Google, incluindo os
            requisitos de Uso Limitado. O VGON CRM não vende dados do Gmail, não usa dados do
            Gmail para publicidade e não usa dados do Gmail para treinar modelos de inteligência
            artificial.
          </Section>

          <Section title="4. Finalidade do tratamento">
            Os dados são usados para autenticar usuários, organizar atendimentos, importar
            mensagens recebidas, permitir respostas aos clientes, gerar histórico de conversas,
            melhorar a operação interna e cumprir obrigações legais e de segurança.
          </Section>

          <Section title="5. Compartilhamento de dados">
            Não vendemos dados pessoais. Podemos compartilhar dados apenas com provedores
            necessários para funcionamento da plataforma, como hospedagem, banco de dados,
            mensageria, integrações autorizadas e serviços de autenticação, sempre conforme
            necessidade operacional e medidas de segurança.
          </Section>

          <Section title="6. Armazenamento e segurança">
            Aplicamos medidas técnicas e organizacionais para proteger os dados contra acesso
            não autorizado, perda, uso indevido ou alteração. Credenciais e tokens de integração
            devem ser tratados como informações confidenciais.
          </Section>

          <Section title="7. Direitos do usuário">
            O usuário pode solicitar acesso, correção, exclusão ou limitação de uso dos seus
            dados, quando aplicável. Também pode revogar permissões de Gmail ou Outlook nas
            configurações da própria conta Google ou Microsoft.
          </Section>

          <Section title="8. Contato">
            Para dúvidas sobre privacidade ou tratamento de dados, entre em contato pelo e-mail:
            contato@vgon.com.br.
          </Section>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] mt-16">
        <div className="mx-auto max-w-5xl px-6 py-6 flex items-center justify-between text-xs text-white/30">
          <span>© 2026 VGON. Todos os direitos reservados.</span>
          <div className="flex gap-4">
            <Link href="/termos-de-servico" className="hover:text-white/70 transition-colors">Termos de Uso</Link>
            <Link href="/politica-de-privacidade" className="text-white/60">Privacidade</Link>
          </div>
        </div>
      </footer>
    </main>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h2 className="text-lg font-semibold text-white mb-3">{title}</h2>
      <p>{children}</p>
    </div>
  )
}
