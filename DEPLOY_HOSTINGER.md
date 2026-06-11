# Deploy EvoCRM Pro na Hostinger VPS

## Pré-requisitos

- VPS Hostinger com template **Docker** (Ubuntu 24.04 com Docker + Docker Compose pré-instalados)
- Repositório GitHub: https://github.com/douglasoliveira21/vgon-crm
- Supabase configurado (já feito ✅)
- Domínio apontando para o IP do VPS

---

## Método 1: Docker Manager da Hostinger (Mais fácil)

### Passo 1 - Acessar o Docker Manager

1. Acesse o **hPanel** da Hostinger
2. Vá em **VPS** → clique em **Manage** no seu servidor
3. No menu lateral, clique em **Docker Manager**
4. Clique no botão **Compose**

### Passo 2 - Deploy via URL do GitHub

1. Selecione **"Compose from URL"**
2. Cole a URL direta do docker-compose:
   ```
   https://raw.githubusercontent.com/douglasoliveira21/vgon-crm/main/docker-compose.yml
   ```
3. O sistema vai carregar o arquivo
4. **Edite as variáveis de ambiente** antes de deployar (veja abaixo)
5. Clique em **Deploy**

### Passo 3 - Configurar Variáveis

Antes de confirmar o deploy, edite o compose e substitua as variáveis:

```yaml
environment:
  - DATABASE_URL=postgresql://postgres.[ref]:[sua-senha]@aws-0-us-east-1.pooler.supabase.com:6543/postgres
  - REDIS_URL=redis://redis:6379
  - JWT_SECRET=gere-string-aleatoria-32-chars
  - JWT_REFRESH_SECRET=outra-string-aleatoria-32-chars
  - EVOLUTION_API_URL=http://seu-ip:8080
  - EVOLUTION_API_KEY=sua-chave-evolution
  - EVOLUTION_WEBHOOK_URL=http://seu-ip:3001/api/webhooks/evolution
  - APP_PORT=3001
  - FRONTEND_URL=http://seu-ip:3000
```

---

## Método 2: SSH + Docker Compose (Mais controle)

### Passo 1 - Conectar via SSH

```bash
ssh root@seu-ip-do-vps
```

### Passo 2 - Clonar o repositório

```bash
cd /opt
git clone https://github.com/douglasoliveira21/vgon-crm.git
cd vgon-crm
```

### Passo 3 - Criar o arquivo .env

```bash
nano .env
```

Cole o conteúdo:

```env
# Supabase
DATABASE_URL=postgresql://postgres.gdjhtodwqakknwhlmldp:[SUA-SENHA]@aws-0-us-east-1.pooler.supabase.com:6543/postgres

# Redis (interno via Docker)
REDIS_URL=redis://redis:6379

# JWT (gere strings aleatórias seguras)
JWT_SECRET=MUDE-PARA-STRING-ALEATORIA-SEGURA-32-CHARS
JWT_REFRESH_SECRET=OUTRA-STRING-ALEATORIA-SEGURA-32-CHARS

# Evolution API
EVOLUTION_API_URL=http://seu-ip:8080
EVOLUTION_API_KEY=sua-chave-da-evolution
EVOLUTION_WEBHOOK_URL=http://seu-ip:3001/api/webhooks/evolution

# App
APP_PORT=3001
APP_ENV=production
FRONTEND_URL=http://seu-ip:3000
```

Para gerar JWT secrets seguros:
```bash
openssl rand -base64 32
```

### Passo 4 - Build e Deploy

```bash
docker compose up -d --build
```

Isso vai:
- Subir o Redis
- Buildar e subir o Backend (Go) na porta 3001
- Buildar e subir o Frontend (Next.js) na porta 3000

### Passo 5 - Verificar se está rodando

```bash
docker compose ps
```

Deve mostrar 3 containers `running`:
```
NAME          IMAGE              STATUS
backend       vgon-crm-backend   Up
frontend      vgon-crm-frontend  Up
redis         redis:7-alpine     Up
```

### Passo 6 - Ver logs

```bash
# Backend
docker compose logs backend -f

# Frontend
docker compose logs frontend -f
```

### Passo 7 - Acessar

- **Frontend**: `http://seu-ip:3000`
- **Backend API**: `http://seu-ip:3001/health`

---

## Configurar Domínio + HTTPS (Nginx Reverse Proxy)

### Passo 1 - Instalar Nginx e Certbot

```bash
apt update
apt install nginx certbot python3-certbot-nginx -y
```

### Passo 2 - Configurar Nginx

```bash
nano /etc/nginx/sites-available/vgon-crm
```

```nginx
# Frontend
server {
    listen 80;
    server_name app.seudominio.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_bypass $http_upgrade;
    }
}

# Backend API
server {
    listen 80;
    server_name api.seudominio.com;

    client_max_body_size 50M;

    location / {
        proxy_pass http://127.0.0.1:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_bypass $http_upgrade;
    }

    # WebSocket
    location /ws {
        proxy_pass http://127.0.0.1:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 86400;
    }
}
```

### Passo 3 - Ativar e gerar HTTPS

```bash
ln -s /etc/nginx/sites-available/vgon-crm /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx

# Gerar certificado SSL
certbot --nginx -d app.seudominio.com -d api.seudominio.com
```

### Passo 4 - Atualizar variáveis de ambiente

Após ter HTTPS, atualize o `.env`:

```env
FRONTEND_URL=https://app.seudominio.com
EVOLUTION_WEBHOOK_URL=https://api.seudominio.com/api/webhooks/evolution
```

E no frontend, atualize no `docker-compose.yml`:
```yaml
environment:
  - NEXT_PUBLIC_API_URL=https://api.seudominio.com
  - NEXT_PUBLIC_WS_URL=wss://api.seudominio.com
```

Rebuild:
```bash
docker compose up -d --build
```

---

## Método 3: GitHub Actions (Deploy automático)

A Hostinger suporta deploy automático via GitHub Actions. Crie o workflow:

### `.github/workflows/deploy.yml`

```yaml
name: Deploy to Hostinger VPS

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.VPS_HOST }}
          username: root
          key: ${{ secrets.VPS_SSH_KEY }}
          script: |
            cd /opt/vgon-crm
            git pull origin main
            docker compose up -d --build
```

Configure os secrets no GitHub:
- `VPS_HOST`: IP do seu VPS
- `VPS_SSH_KEY`: Chave SSH privada

---

## Resumo

| Método | Dificuldade | Quando usar |
|--------|-------------|-------------|
| Docker Manager (hPanel) | ⭐ Fácil | Deploy rápido sem SSH |
| SSH + Docker Compose | ⭐⭐ Médio | Mais controle, domínio customizado |
| GitHub Actions | ⭐⭐⭐ Avançado | CI/CD automático a cada push |

---

## Portas necessárias (Firewall)

Libere no firewall do VPS:
- `80` - HTTP
- `443` - HTTPS
- `3000` - Frontend (se não usar Nginx)
- `3001` - Backend (se não usar Nginx)

Na Hostinger: **hPanel > VPS > Firewall** e adicione as regras.

---

## Checklist Final

- [ ] VPS com Docker instalado
- [ ] Repositório clonado em `/opt/vgon-crm`
- [ ] Arquivo `.env` configurado com credenciais reais
- [ ] `docker compose up -d --build` executado
- [ ] 3 containers running
- [ ] DNS apontando para IP do VPS
- [ ] Nginx configurado com HTTPS
- [ ] Acessar `https://app.seudominio.com` e criar conta
- [ ] Conectar WhatsApp pela tela de Canais
