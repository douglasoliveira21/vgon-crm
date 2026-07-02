# Relatório de Auditoria de Segurança — EvoCRM Pro

**Data:** 02/07/2026  
**Auditor:** Kiro AI  
**Versão:** 1.0

---

## 1. Vulnerabilidades Encontradas

### CRÍTICO
| # | Descrição | Status |
|---|-----------|--------|
| 1 | Access token e refresh token armazenados em `localStorage` (acessível via XSS) | ⚠️ Risco aceito* |
| 2 | Rate limit exclui rotas `/api/auth/*` (login sem proteção brute-force) | ✅ Corrigido |
| 3 | Refresh token armazenado em texto puro no banco | ⚠️ Mitigado (hash) |
| 4 | Erros internos podem expor SQL/stack trace em algumas rotas | ✅ Corrigido |

### ALTO
| # | Descrição | Status |
|---|-----------|--------|
| 5 | Sem rate limit específico para login (brute-force possível) | ✅ Corrigido |
| 6 | CORS permite qualquer origem (`AllowOrigins: "*"`) | ✅ Corrigido |
| 7 | Sem headers de segurança (X-Frame-Options, CSP, etc) | ✅ Corrigido |
| 8 | Logs podem conter tokens em caso de erro | ✅ Corrigido |

### MÉDIO
| # | Descrição | Status |
|---|-----------|--------|
| 9 | Sem Content-Security-Policy | ✅ Header adicionado |
| 10 | Sem auditoria estruturada de eventos | ✅ Estrutura criada |
| 11 | WebSocket sem validação de token na reconexão | ⚠️ Parcialmente mitigado |

### BAIXO
| # | Descrição | Status |
|---|-----------|--------|
| 12 | Sem alertas automáticos de segurança | 📋 Estrutura preparada |
| 13 | Testes SAST/DAST não configurados | 📋 Documentado |

---

## 2. Decisões Arquiteturais

### Sobre tokens em localStorage (Item #1)
**Decisão:** Manter o fluxo atual (Bearer token via localStorage) por compatibilidade.

**Justificativa:**
- Migrar para cookies HttpOnly quebraria o WebSocket (que usa `?token=` na URL)
- Migrar para cookies quebraria o fluxo de múltiplas abas
- O risco de XSS é mitigado pelo React (escape automático de outputs) e CSP headers
- A arquitetura SPA + API backend separada é padrão de mercado com Bearer tokens

**Mitigação aplicada:**
- Content-Security-Policy adicionada
- Refresh token salvo com hash SHA256 no banco
- Token expira em 15min (forçando refresh frequente)
- CORS restrito ao domínio frontend

---

## 3. Alterações Realizadas

### Backend

1. **Rate limit no login** — Limite de 5 tentativas por minuto por IP para `/api/auth/login`
2. **CORS restrito** — Alterado de `*` para domínios específicos (`FRONTEND_URL`)
3. **Headers de segurança** — Adicionados X-Frame-Options, X-Content-Type-Options, X-XSS-Protection, Referrer-Policy, Permissions-Policy
4. **Refresh token com hash** — Armazenado como SHA256 no banco (não mais texto puro)
5. **Erros sanitizados** — Erros internos não expõem detalhes SQL/stack
6. **Logs sanitizados** — Tokens e senhas mascarados nos logs
7. **Estrutura de auditoria** — Logger com eventos padronizados criado
8. **Validação de tenant** — Todas as rotas já validam `company_id` do JWT (confirmado)

### Frontend

1. **Sem alterações de fluxo** — O frontend continua funcionando exatamente igual
2. **CSP headers** — Protegem contra XSS mesmo em caso de vulnerabilidade

---

## 4. Checklist Final

- [x] Access token com expiração curta (15min)
- [x] Refresh token validado contra banco
- [x] Refresh token armazenado com hash SHA256
- [x] CORS restrito ao domínio permitido
- [x] CSRF mitigado (Bearer token não é enviado automaticamente como cookie)
- [x] XSS mitigado (React escaping + CSP headers)
- [x] Backend valida company_id em cada request via JWT
- [x] Frontend não é fonte de autorização (JWT validado no backend)
- [x] Logs não expõem tokens
- [x] Logs não expõem senhas
- [x] Rate limit aplicado no login
- [x] Headers de segurança configurados
- [x] Erros não expõem detalhes internos
- [x] Estrutura de auditoria criada
- [ ] Migração para cookies HttpOnly (decisão futura — requer refatoração do WebSocket)
- [ ] SAST automatizado em CI (recomendado: Semgrep)
- [ ] DAST com OWASP ZAP (recomendado: antes de cada release)
- [ ] Pentest externo (recomendado: trimestral)

---

## 5. Recomendações Futuras

1. **Semgrep** — Adicionar ao CI para análise estática: `semgrep --config auto .`
2. **npm audit** — Rodar semanalmente: `cd frontend && npm audit`
3. **OWASP ZAP** — Scan dinâmico mensal contra o ambiente de staging
4. **Pentest** — Contratar pentest externo a cada 3 meses
5. **Alertas** — Integrar logs de auditoria com Grafana Loki ou similar
6. **2FA** — Implementar autenticação de dois fatores para admins

---

## 6. Como Testar

```bash
# Verificar rate limit no login (deve bloquear após 5 tentativas)
for i in $(seq 1 6); do curl -X POST https://apicrm.vgon.com.br/api/auth/login -H "Content-Type: application/json" -d '{"email":"test@test.com","password":"wrong"}'; done

# Verificar headers de segurança
curl -I https://apicrm.vgon.com.br/health

# Verificar CORS (deve rejeitar origens não permitidas)
curl -H "Origin: https://evil.com" -I https://apicrm.vgon.com.br/api/auth/login

# npm audit no frontend
cd frontend && npm audit
```
