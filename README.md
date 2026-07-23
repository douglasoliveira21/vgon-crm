# EvoCRM Pro

Sistema SaaS CRM Omnichannel com integração WhatsApp via Evolution API.

## Arquitetura

- **Backend:** Go (Fiber framework)
- **Frontend:** Next.js + React + TypeScript + Tailwind CSS
- **Banco de dados:** Supabase (PostgreSQL)
- **Cache/Fila:** Redis
- **Realtime:** WebSocket (Gorilla WebSocket)
- **Integração WhatsApp:** Evolution API
- **Deploy:** Docker Compose / EasyPanel

## Estrutura do Projeto

```
├── backend/              # API Go (Fiber)
│   ├── cmd/              # Entry points
│   ├── internal/         # Código interno
│   │   ├── config/       # Configurações
│   │   ├── middleware/   # Middlewares (auth, RBAC, rate limit)
│   │   ├── models/       # Modelos do banco
│   │   ├── handlers/     # Handlers HTTP
│   │   ├── services/     # Lógica de negócio
│   │   ├── repository/   # Acesso a dados
│   │   ├── websocket/    # WebSocket hub
│   │   └── evolution/    # Integração Evolution API
│   ├── migrations/       # Migrações SQL
│   └── docs/             # Documentação API
├── frontend/             # Next.js App
│   ├── src/
│   │   ├── app/          # App Router
│   │   ├── components/   # Componentes React
│   │   ├── lib/          # Utilitários
│   │   ├── hooks/        # React hooks
│   │   ├── store/        # State management
│   │   └── types/        # TypeScript types
├── docker-compose.yml    # Orquestração
└── README.md
```

## Instalação

### Pré-requisitos

- Go 1.22+
- Node.js 20+
- Docker & Docker Compose
- Redis
- Supabase (ou PostgreSQL local)
- Evolution API rodando

### Setup

1. Clone o repositório
2. Configure as variáveis de ambiente (`.env`)
3. Execute as migrações
4. Inicie com Docker Compose:

```bash
docker-compose up -d
```

### Variáveis de Ambiente

Copie `.env.example` para `.env` e configure:

```env
# Supabase
DATABASE_URL=postgresql://user:pass@host:5432/evocrm
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=your-secret-key
JWT_REFRESH_SECRET=your-refresh-secret

# Evolution API
EVOLUTION_API_URL=http://localhost:8080
EVOLUTION_API_KEY=your-api-key

# App
APP_PORT=3001
FRONTEND_URL=http://localhost:3000
```

## Módulos

- ✅ Autenticação JWT com refresh token
- ✅ Multiempresa/Multitenant
- ✅ RBAC (Role-Based Access Control)
- ✅ Integração WhatsApp via Evolution API
- ✅ Caixa de entrada
- ✅ Conversas em tempo real (WebSocket)
- ✅ Contatos
- ✅ Times e distribuição
- ✅ Funil de vendas (Kanban)
- ✅ Bots e automações
- ✅ Marketing em massa
- ✅ Chamadas de áudio
- ✅ Métricas e relatórios
- ✅ Widget para site
- ✅ Recados internos

## Operação e webhooks

- Disponibilidade: `GET /health/live`
- Prontidão e dependências: `GET /health/ready`
- O provedor de e-mail deve enviar eventos para `POST /api/webhooks/email/events`
  usando o cabeçalho `X-Webhook-Secret` com o valor de `EMAIL_WEBHOOK_SECRET`.
- Eventos aceitos: `sent`, `delivered`, `opened`, `clicked`, `bounced` e
  `complained`. O corpo deve informar `company_id`, `email`, `event` e
  `provider_event_id`; `campaign_id`, `contact_id` e `details` são opcionais.
- Bounce e reclamação adicionam o e-mail automaticamente à lista de supressão.
- Jobs que esgotam as tentativas aparecem em Operações e Segurança para
  diagnóstico e reenvio manual.

## Licença

Proprietário - Todos os direitos reservados.
