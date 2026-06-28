import Link from 'next/link'
import { Mail, MessageSquare, ShieldCheck, Users } from 'lucide-react'

export default function Home() {
  return (
    <main className="min-h-screen bg-white text-gray-900">
      <section className="border-b border-gray-100 bg-gradient-to-b from-blue-50 to-white">
        <div className="mx-auto max-w-6xl px-6 py-16">
          <div className="max-w-3xl">
            <p className="text-sm font-semibold uppercase tracking-wide text-primary-600">crmvgon</p>
            <h1 className="mt-4 text-4xl font-bold tracking-tight text-gray-950 sm:text-5xl">
              crmvgon e um CRM de atendimento para centralizar conversas com clientes.
            </h1>
            <p className="mt-5 text-lg leading-8 text-gray-600">
              O aplicativo ajuda empresas a organizar contatos, equipes, conversas e canais de atendimento em um unico painel.
              Com a integracao de e-mail, usuarios autorizados podem conectar Gmail ou Outlook para ler mensagens recebidas,
              responder clientes pelo CRM e manter o historico do atendimento em conversas.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link href="/login" className="btn-primary">
                Acessar o CRM
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
