# 2026-09-16 — board-view: the board is a bounded view (ADR-0145, primeiro item do refactor)

Owner: "essa porra de board não cabe nada … o que precisa ser refatorado nesse
fluxo?" — e aprovou os itens 1+2 (este) e 3+4 (próxima sessão).

O que mudou em `scripts/handoff-board.mjs`:
- **Quota por tópico**: 2 bullets de `## Next`, depois `…+N more in
  docs/handoff/open/<topic>.md`. O resto fica no arquivo (ADR-0131 C vale para
  os dois títulos agora).
- **Nota é handoff, não ledger**: 1 bullet por seção por nota, janela de **7
  dias** (era 30), teto de 8 bullets de notas por seção — o resto vira um
  ponteiro.
- **Dívida tem estado**: `- [ ]`/bullet simples = aberta, `- [x]` = paga. O
  board conta só as abertas e diz quantas pagas; o arquivo guarda o histórico.
- **Acima do alvo (120 linhas/12 KB) avisa e escreve assim mesmo.** `make
  handoff` só falha quando a *fonte* não é legível (o arquivo de tópico
  invisível do ADR-0131 A) — tamanho nunca mais bloqueia `make close`.
- Resultado medido: **78 linhas / 7,0 KB, 16 bullets atrás de ponteiros** (era
  12,1 KB e falhando; 25 tópicos × 1-2 + notas cabendo sem poda manual).

Docs: ADR-0145 (aceita, refina 0123/0131/0140), AGENTS.md §1, o README dos
tópicos (convenção `- [ ]`/`- [x]`), o skill `handoff-update` (nunca podar
bullet de outra sessão), e o bullet do tópico `process` marcado `[x]` — a
dívida do cap foi paga por esta mudança.

Verificado: `node --test scripts/*.test.mjs` 82/82 (novos: quota+ponteiro,
nota de 5 dias ainda aparece, tópico verboso não estoura, muitos tópicos
avisam por nome e ainda renderizam); `make handoff` exit 0; `make ci-scoped`
PASS.

## Next up

- Owner: abrir o board (sidebar) — 78 linhas, cada tópico com no máximo 2
  próximos passos e um ponteiro; dívidas só as abertas.
- Itens 3+4 do refactor (aprovados): mover bullets da nota para o tópico no
  `make close`, e `make land BRANCH=x` (recusa índice sujo/path estrangeiro,
  ff-only, roda ci).

## Debts

- A seleção fica sendo o risco: se os dois primeiros bullets de um tópico
  envelhecem, o board mente por seleção. Mitigação é a mesma de antes (pagar e
  subir o próximo), agora visível num arquivo só.
- 379 notas antigas continuam contribuindo até 7 dias depois da data delas — a
  migração mecânica delas para os tópicos é o item 3.
