import Link from 'next/link'

export const metadata = {
  title: 'Politica de Privacidade | VGON CRM',
  description: 'Politica de privacidade do aplicativo VGON CRM.',
}

export default function PrivacyPolicyPage() {
  return (
    <main className="min-h-screen bg-gray-50 text-gray-900">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <div className="mb-8">
          <Link href="/login" className="text-sm font-medium text-primary-600 hover:text-primary-700">
            VGON CRM
          </Link>
          <h1 className="mt-4 text-3xl font-bold">Politica de Privacidade</h1>
          <p className="mt-2 text-sm text-gray-500">Ultima atualizacao: 28 de junho de 2026</p>
        </div>

        <section className="space-y-6 rounded-xl bg-white p-8 shadow-sm border border-gray-100">
          <p>
            Esta Politica de Privacidade explica como o VGON CRM coleta, usa, armazena e protege
            informacoes dos usuarios e dos contatos atendidos pela plataforma.
          </p>

          <div>
            <h2 className="text-xl font-semibold">1. Dados que coletamos</h2>
            <p className="mt-2 text-gray-700">
              Podemos coletar dados de cadastro, como nome, e-mail, telefone, empresa, credenciais
              de acesso, registros de atendimento, mensagens, contatos, historico de conversas e
              configuracoes de canais conectados ao CRM.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">2. Integracao com e-mail, Gmail e Outlook</h2>
            <p className="mt-2 text-gray-700">
              Quando o usuario conecta uma conta de e-mail ao VGON CRM, o sistema pode solicitar
              permissoes para ler mensagens recebidas e enviar respostas em nome da conta conectada.
              Essas permissoes sao usadas somente para exibir e responder e-mails dentro das
              conversas do CRM.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">3. Uso de dados do Google</h2>
            <p className="mt-2 text-gray-700">
              O uso e a transferencia de informacoes recebidas das APIs do Google pelo VGON CRM
              obedecem a Politica de Dados de Usuario dos Servicos de API do Google, incluindo os
              requisitos de Uso Limitado. O VGON CRM nao vende dados do Gmail, nao usa dados do
              Gmail para publicidade e nao usa dados do Gmail para treinar modelos de inteligencia
              artificial.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">4. Finalidade do tratamento</h2>
            <p className="mt-2 text-gray-700">
              Os dados sao usados para autenticar usuarios, organizar atendimentos, importar
              mensagens recebidas, permitir respostas aos clientes, gerar historico de conversas,
              melhorar a operacao interna e cumprir obrigacoes legais e de seguranca.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">5. Compartilhamento de dados</h2>
            <p className="mt-2 text-gray-700">
              Nao vendemos dados pessoais. Podemos compartilhar dados apenas com provedores
              necessarios para funcionamento da plataforma, como hospedagem, banco de dados,
              mensageria, integracoes autorizadas e servicos de autentificacao, sempre conforme
              necessidade operacional e medidas de seguranca.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">6. Armazenamento e seguranca</h2>
            <p className="mt-2 text-gray-700">
              Aplicamos medidas tecnicas e organizacionais para proteger os dados contra acesso
              nao autorizado, perda, uso indevido ou alteracao. Credenciais e tokens de integracao
              devem ser tratados como informacoes confidenciais.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">7. Direitos do usuario</h2>
            <p className="mt-2 text-gray-700">
              O usuario pode solicitar acesso, correcao, exclusao ou limitacao de uso dos seus
              dados, quando aplicavel. Tambem pode revogar permissoes de Gmail ou Outlook nas
              configuracoes da propria conta Google ou Microsoft.
            </p>
          </div>

          <div>
            <h2 className="text-xl font-semibold">8. Contato</h2>
            <p className="mt-2 text-gray-700">
              Para duvidas sobre privacidade ou tratamento de dados, entre em contato pelo e-mail:
              contato@vgon.com.br.
            </p>
          </div>
        </section>
      </div>
    </main>
  )
}
