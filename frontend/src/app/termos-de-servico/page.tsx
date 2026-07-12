import Link from 'next/link'

export const metadata = {
  title: 'Termos de Serviço | VGON',
  description: 'Termos de serviço da plataforma VGON CRM.',
}

export default function TermsOfServicePage() {
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
          <h1 className="text-3xl font-bold tracking-[-0.02em]">Termos de Serviço</h1>
          <p className="mt-3 text-sm text-white/40">Última atualização: 28 de junho de 2026</p>
        </div>

        <div className="space-y-10 text-[15px] leading-7 text-white/70">
          <p>
            Estes Termos de Serviço regulam o uso do VGON CRM, plataforma de atendimento e gestão
            de conversas, contatos, canais e integrações.
          </p>

          <Section title="1. Aceite dos termos">
            Ao acessar ou utilizar o VGON CRM, o usuário declara que leu, compreendeu e concorda
            com estes termos e com a Política de Privacidade.
          </Section>

          <Section title="2. Uso permitido">
            O VGON CRM deve ser usado para fins legítimos de atendimento, relacionamento com
            clientes, organização comercial e comunicação autorizada. O usuário é responsável
            pelo conteúdo enviado pela plataforma e pelas integrações conectadas.
          </Section>

          <Section title="3. Integrações de terceiros">
            O sistema pode integrar serviços como Gmail, Outlook, Microsoft Graph, WhatsApp e
            outros provedores. O uso dessas integrações também está sujeito aos termos e políticas
            dos respectivos provedores.
          </Section>

          <Section title="4. Permissões de e-mail">
            Ao conectar uma conta Gmail ou Outlook, o usuário autoriza o VGON CRM a ler e-mails
            recebidos e enviar respostas conforme necessário para o atendimento dentro da
            plataforma. O usuário pode revogar o acesso a qualquer momento nas configurações da
            conta Google ou Microsoft.
          </Section>

          <Section title="5. Responsabilidades do usuário">
            O usuário deve manter suas credenciais seguras, respeitar leis aplicáveis, não enviar
            spam, não praticar abuso, fraude ou violação de direitos de terceiros e garantir que
            possui autorização para tratar os dados inseridos no CRM.
          </Section>

          <Section title="6. Disponibilidade">
            Buscamos manter o serviço disponível e seguro, mas interrupções podem ocorrer por
            manutenção, falhas técnicas, indisponibilidade de terceiros ou eventos fora do nosso
            controle.
          </Section>

          <Section title="7. Limitação de responsabilidade">
            Na máxima extensão permitida pela lei, o VGON CRM não se responsabiliza por perdas
            indiretas, lucros cessantes, indisponibilidade de provedores externos ou uso indevido
            da plataforma pelo usuário.
          </Section>

          <Section title="8. Alterações dos termos">
            Estes termos podem ser atualizados periodicamente. A continuidade do uso da plataforma
            após alterações representa aceite da versão atualizada.
          </Section>

          <Section title="9. Contato">
            Para dúvidas sobre estes termos, entre em contato pelo e-mail: contato@vgon.com.br.
          </Section>
        </div>
      </div>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] mt-16">
        <div className="mx-auto max-w-5xl px-6 py-6 flex items-center justify-between text-xs text-white/30">
          <span>© 2026 VGON. Todos os direitos reservados.</span>
          <div className="flex gap-4">
            <Link href="/termos-de-servico" className="text-white/60">Termos de Uso</Link>
            <Link href="/politica-de-privacidade" className="hover:text-white/70 transition-colors">Privacidade</Link>
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
