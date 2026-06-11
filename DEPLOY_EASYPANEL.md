# Deploy EvoCRM Pro no EasyPanel

## Pré-requisitos

- Servidor com EasyPanel instalado (https://easypanel.io/docs)
- Conta no GitHub com o repositório: https://github.com/douglasoliveira21/vgon-crm
- Supabase configurado (já feito ✅)
- Redis (vamos criar no EasyPanel)
- Evolution API rodando (pode ser no mesmo servidor ou externo)

---

## Passo a Passo

### 1. Criar Projeto no EasyPanel

1. Acesse seu EasyPanel (ex: `https://seu-servidor:3000`)
2. Clique em **"Create Project"**
3. Nome: `vgon-crm`

---

### 2. Criar Serviço Redis

1. Dentro do projeto `vgon-crm`, clique em **"+ Service"**
2. Escolha **"Database" > "Redis"**
3. Nome: `redis`
4. Clique em **Create**
5. Anote a connection URL interna: `redis://redis:6379`

---

### 3. Criar Serviço Backend (Go API)

1. Clique em **"+ Service" > "App"**
2. Nome: `backend`
3. **Source**: GitHub
4. **Repository**: `douglasoliveira21/vgon-crm`
5. **Branch**: `main`
6. **Root Directory**: `backend`
7. **Build Method**: Dockerfile (vai detectar o `backend/Dockerfile` automaticamente)

#### Configurar Variáveis de Ambiente:

Vá em **Environment** e adicione:

```
DATABASE_URL=postgresql://postgres.[ref]:[password]@aws-0-us-east-1.pooler.supabase.com:6543/postgres
REDIS_URL=redis://redis:6379
JWT_SECRET=gere-uma-string-aleatoria-segura-aqui
JWT_REFRESH_SECRET=outra-string-aleatoria-segura
EVOLUTION_API_URL=http://sua-evolution-api:8080
EVOLUTION_API_KEY=sua-api-key-evolution
EVOLUTION_WEBHOOK_URL=https://backend.seu-dominio.com/api/webhooks/evolution
APP_PORT=3001
APP_ENV=production
FRONTEND_URL=https://app.seu-dominio.com
```

#### Configurar Porta:

- Vá em **Domains**
- Porta do container: `3001`
- Adicione seu domínio: `backend.seu-dominio.com` ou `api.seu-dominio.com`
- Ative HTTPS (Let's Encrypt automático)

#### Deploy:

- Clique em **Deploy**

---

### 4. Criar Serviço Frontend (Next.js)

1. Clique em **"+ Service" > "App"**
2. Nome: `frontend`
3. **Source**: GitHub
4. **Repository**: `douglasoliveira21/vgon-crm`
5. **Branch**: `main`
6. **Root Directory**: `frontend`
7. **Build Method**: Dockerfile

#### Configurar Variáveis de Ambiente:

```
NEXT_PUBLIC_API_URL=https://api.seu-dominio.com
NEXT_PUBLIC_WS_URL=wss://api.seu-dominio.com
```

#### Configurar Porta:

- Porta do container: `3000`
- Domínio: `app.seu-dominio.com`
- HTTPS ativo

#### Deploy:

- Clique em **Deploy**

---

### 5. (Opcional) Evolution API no mesmo EasyPanel

Se quiser rodar a Evolution API no mesmo servidor:

1. **"+ Service" > "App"**
2. Nome: `evolution`
3. **Source**: Docker Image
4. **Image**: `atendai/evolution-api:latest`
5. Porta: `8080`

#### Variáveis de Ambiente da Evolution API:

```
AUTHENTICATION_API_KEY=sua-chave-global
AUTHENTICATION_EXPOSE_IN_FETCH_INSTANCES=true
DEL_INSTANCE=false
DATABASE_ENABLED=true
DATABASE_PROVIDER=postgresql
DATABASE_CONNECTION_URI=postgresql://postgres.[ref]:[pass]@host:6543/postgres
```

#### Domínio:

- `evolution.seu-dominio.com` com HTTPS

Depois atualize no backend:
```
EVOLUTION_API_URL=https://evolution.seu-dominio.com
EVOLUTION_WEBHOOK_URL=https://api.seu-dominio.com/api/webhooks/evolution
```

---

## Resumo da Arquitetura no EasyPanel

```
Projeto: vgon-crm
├── frontend    (Next.js)     → app.seu-dominio.com
├── backend     (Go/Fiber)    → api.seu-dominio.com
├── redis       (Redis 7)     → interno (redis:6379)
└── evolution   (Evolution)   → evolution.seu-dominio.com (opcional)
```

Banco de dados: **Supabase** (externo, já configurado)

---

## Após o Deploy

1. Acesse `https://app.seu-dominio.com`
2. Crie uma conta (registrar)
3. Vá em **Canais** > **Conectar WhatsApp**
4. Escaneie o QR Code
5. Comece a atender! 🚀

---

## Dicas

- O EasyPanel detecta o Dockerfile automaticamente no diretório raiz de cada serviço
- Auto-deploy: ative o webhook do GitHub no EasyPanel para deploy automático a cada push
- Logs: acesse via aba "Logs" de cada serviço no EasyPanel
- Escalar: ajuste CPU/RAM de cada serviço nas configurações
- Backup: o Supabase já faz backup automático do banco

---

## Variáveis que você precisa preencher

| Variável | Onde conseguir |
|----------|---------------|
| DATABASE_URL | Supabase > Settings > Database > Connection String (URI) |
| JWT_SECRET | Gere com: `openssl rand -base64 32` |
| JWT_REFRESH_SECRET | Gere com: `openssl rand -base64 32` |
| EVOLUTION_API_URL | URL da sua Evolution API |
| EVOLUTION_API_KEY | Chave configurada na Evolution API |
| Domínios | Configure DNS apontando para o IP do servidor |
