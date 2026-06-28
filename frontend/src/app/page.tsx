import Link from 'next/link'
import { Mail, MessageSquare, ShieldCheck, Users } from 'lucide-react'

export default function Home() {
  return (
    <main className="min-h-screen bg-white text-gray-900">
      <section className="border-b border-gray-100 bg-gradient-to-b from-blue-50 to-white">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <div className="max-w-3xl">
            <div className="mb-8 rounded-2xl bg-white p-5 shadow-sm ring-1 ring-gray-100">
              <img src="/logo-black.png" alt="Logo do crmvgon" className="h-16 w-auto max-w-full object-contain" />
            </div>
            <h1 className="mt-4 text-4xl font-bold tracking-tight text-gray-950 sm:text-5xl">
              crmvgon
            </h1>
            <p className="mt-5 text-lg leading-8 text-gray-600">
              crmvgon e um aplicativo de CRM de atendimento que ajuda empresas a organizar contatos, equipes,
              conversas e canais de atendimento em um unico painel.
              Com a integracao de e-mail, usuarios autorizados podem conectar Gmail ou Outlook para ler mensagens recebidas,
              responder clientes pelo CRM e manter o historico do atendimento em conversas.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link href="/login" className="btn-primary">
                Acessar o crmvgon
              </Link>
              <Link href="/politica-de-privacidade" className="btn-secondary">
                Politica de Privacidade
              </Link>
              <Link href="/termos-de-servico" className="btn-secondary">
                Termos de Servico
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-14">
        <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          <Feature
            icon={<MessageSquare size={22} />}
            title="Conversas centralizadas"
            description="Reune mensagens e atendimentos em uma caixa de conversas para acompanhamento pela equipe."
          />
          <Feature
            icon={<Mail size={22} />}
            title="Gmail e Outlook"
            description="Permite conectar contas com autorizacao OAuth para ler e responder e-mails no CRM."
          />
          <Feature
            icon={<Users size={22} />}
            title="Equipes e contatos"
            description="Organiza contatos, responsaveis, times, historico e distribuicao de atendimentos."
          />
          <Feature
            icon={<ShieldCheck size={22} />}
            title="Uso autorizado"
            description="As permissoes de e-mail sao usadas apenas para exibir mensagens e enviar respostas solicitadas pelo usuario."
          />
        </div>
      </section>

      <section className="bg-gray-50">
        <div className="mx-auto max-w-6xl px-6 py-12">
          <h2 className="text-2xl font-bold text-gray-950">Finalidade do aplicativo</h2>
          <p className="mt-4 max-w-4xl leading-7 text-gray-600">
            O crmvgon existe para facilitar o atendimento ao cliente. Ele importa mensagens recebidas de canais autorizados,
            cria conversas vinculadas aos contatos e permite que os atendentes respondam diretamente pelo painel. No caso
            de Gmail e Outlook, o app solicita permissoes de leitura e envio para que o usuario consiga visualizar e
            responder e-mails da propria conta conectada.
          </p>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-6 py-12">
        <div className="grid gap-8 lg:grid-cols-2">
          <div>
            <h2 className="text-2xl font-bold text-gray-950">Como o crmvgon usa dados do Google</h2>
            <p className="mt-4 leading-7 text-gray-600">
              Quando um usuario conecta uma conta Gmail, o crmvgon solicita acesso somente para as
              funcoes de atendimento por e-mail dentro do CRM: ler mensagens recebidas, organizar
              essas mensagens em conversas e enviar respostas quando o proprio usuario decide
              responder um cliente pelo sistema.
            </p>
            <p className="mt-4 leading-7 text-gray-600">
              O crmvgon nao vende dados do Gmail, nao usa dados do Gmail para publicidade e nao usa
              dados do Gmail para treinar modelos de inteligencia artificial. O acesso pode ser
              revogado pelo usuario a qualquer momento na conta Google.
            </p>
          </div>

          <div>
            <h2 className="text-2xl font-bold text-gray-950">Acesso publico e documentos</h2>
            <p className="mt-4 leading-7 text-gray-600">
              Esta pagina inicial e publica e pode ser acessada sem login para explicar a finalidade
              do aplicativo, suas funcionalidades principais e o motivo das permissoes solicitadas.
            </p>
            <div className="mt-6 flex flex-wrap gap-3">
              <Link href="/politica-de-privacidade" className="btn-secondary">
                Ver Politica de Privacidade
              </Link>
              <Link href="/termos-de-servico" className="btn-secondary">
                Ver Termos de Servico
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className="bg-gray-950">
        <div className="mx-auto max-w-6xl px-6 py-10">
          <img src="/logo-white.png" alt="Logo branco do crmvgon" className="h-14 w-auto max-w-full object-contain" />
        </div>
      </section>
    </main>
  )
}

function Feature({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode
  title: string
  description: string
}) {
  return (
    <div className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm">
      <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-blue-50 text-primary-600">
        {icon}
      </div>
      <h3 className="font-semibold text-gray-950">{title}</h3>
      <p className="mt-2 text-sm leading-6 text-gray-600">{description}</p>
    </div>
  )
}
