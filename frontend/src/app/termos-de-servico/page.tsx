import Link from 'next/link'

export const metadata = {
  title: 'Termos de Servico | crmvgon',
  description: 'Termos de servico do aplicativo crmvgon.',
}

export default function TermsOfServicePage() {
  return (
    <main className="min-h-screen bg-gray-50 text-gray-900">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <div className="mb-8">
          <Link href="/login" className="text-sm font-medium text-primary-600 hover:text-primary-700">
            <span className="inline-flex items-center gap-2">
              <img src="/logo-black.png" alt="Logo do crmvgon" className="h-9 w-auto object-contain" />
              crmvgon
            </span>
          </Link>
          <h1 className="mt-4 text-3xl font-bold">Termos de Servico</h1>
          <p className="mt-2 text-sm text-gray-500">Ultima atualizacao: 28 de junho de 2026</p>
        </div>

        <section className="space-y-6 rounded-xl bg-white p-8 shadow-sm border border-gray-100">
          <p>
            Estes Termos de Servico regulam o uso do crmvgon, plataforma de atendimento e gestao
            de conversas, contatos, canais e integracoes.
          </p>

          <div>
            <h2 className="text-xl font-semibold">1. Aceite dos termos</h2>
            <p className="mt-2 text-gray-700">
              Ao acessar ou utilizar o crmvgon, o usuario declara que leu, compreendeu e concorda
              com estes termos e com a Politica de Privacidade.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">2. Uso permitido</h2>
            <p className="mt-2 text-gray-700">
              O crmvgon deve ser usado para fins legitimos de atendimento, relacionamento com
              clientes, organizacao comercial e comunicacao autorizada. O usuario e responsavel
              pelo conteudo enviado pela plataforma e pelas integracoes conectadas.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">3. Integracoes de terceiros</h2>
            <p className="mt-2 text-gray-700">
              O sistema pode integrar servicos como Gmail, Outlook, Microsoft Graph, WhatsApp e
              outros provedores. O uso dessas integracoes tambem esta sujeito aos termos e politicas
              dos respectivos provedores.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">4. Permissoes de e-mail</h2>
            <p className="mt-2 text-gray-700">
              Ao conectar uma conta Gmail ou Outlook, o usuario autoriza o crmvgon a ler e-mails
              recebidos e enviar respostas conforme necessario para o atendimento dentro da
              plataforma. O usuario pode revogar o acesso a qualquer momento nas configuracoes da
              conta Google ou Microsoft.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">5. Responsabilidades do usuario</h2>
            <p className="mt-2 text-gray-700">
              O usuario deve manter suas credenciais seguras, respeitar leis aplicaveis, nao enviar
              spam, nao praticar abuso, fraude ou violacao de direitos de terceiros e garantir que
              possui autorizacao para tratar os dados inseridos no CRM.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">6. Disponibilidade</h2>
            <p className="mt-2 text-gray-700">
              Buscamos manter o servico disponivel e seguro, mas interrupcoes podem ocorrer por
              manutencao, falhas tecnicas, indisponibilidade de terceiros ou eventos fora do nosso
              controle.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">7. Limitacao de responsabilidade</h2>
            <p className="mt-2 text-gray-700">
              Na maxima extensao permitida pela lei, o crmvgon nao se responsabiliza por perdas
              indiretas, lucros cessantes, indisponibilidade de provedores externos ou uso indevido
              da plataforma pelo usuario.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">8. Alteracoes dos termos</h2>
            <p className="mt-2 text-gray-700">
              Estes termos podem ser atualizados periodicamente. A continuidade do uso da plataforma
              apos alteracoes representa aceite da versao atualizada.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">9. Contato</h2>
            <p className="mt-2 text-gray-700">
              Para duvidas sobre estes termos, entre em contato pelo e-mail: contato@vgon.com.br.
            </p>
          </div>
        </section>
      </div>
    </main>
  )
}
