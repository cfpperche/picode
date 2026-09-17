# 2026-09-17 — browser-tool-answers: o teste do full + * achou o defeito

O dono deu `full` + `*` para a minha sessão (terminal `desktop-51c42d`,
`PICODE_TERM_ID` — ADR-0143) e pediu o teste. Rodei a ferramenta `browser` de
verdade contra o x.com aberto na aba de trabalho dele:

| Verbo | Resultado |
|---|---|
| `snapshot` (read) | ✅ árvore do x.com logado-out |
| `evaluate` (act) | ✅ rodou — `document.title = "PI-CODE-EVAL-OK"` apareceu na árvore seguinte — **mas a resposta veio como "no accessible content"** |
| `navigate` (act) | ✅ navegou para `example.com` (árvore + screenshot provam) — mesma resposta vazia |
| `screenshot` (read) | ✅ PNG real da aba (lido) |
| `events` (read) | ⚠️ primeira chamada devolve "nothing since the cursor (last #0)": o anel **se arma na primeira chamada**, então o que aconteceu antes não está nele (comportamento, não bug) |
| `cdp` (raw) | ⛔ recusado, como desenhado: "raw CDP is off — turn on Developer mode in Settings ▸ Browser (Elevated risk)" (ADR-0144) |

**O defeito**: `evaluate`, `navigate` e `cdp` caíam no renderizador da árvore
de acessibilidade — a única formatação que existia. Ou seja: a permissão
funcionava de ponta a ponta e o agente não recebia nada. Corrigido em
`src/logic.ts` (`summarizeEvaluate` — valor, `undefined`, ou "the page threw:
…"; `summarizeNavigate` — para onde foi, ou o `errorText`; `summarizeCdp` —
JSON com teto de 4000 chars) e usado no `extensions/browser.ts`, com 4 testes
novos (12 no pacote).

Importante para re-testar: a ferramenta vem do working tree (`.pi/settings.json`
aponta `../packages/pi-browser`), então **uma sessão nova** já pega a correção
— a minha não, porque o pacote é carregado no início da sessão.

## Next up

- Owner: abrir uma aba de trabalho com uma página e pedir o teste numa sessão
  nova — `evaluate` deve responder o valor, `navigate` "navigated to …".
- Developer mode continua desligado de propósito; o caminho cru segue recusando
  até você ligar em Settings ▸ Browser ▸ Developer mode.

## Debts

- `events` na primeira chamada é sempre vazio (o anel se arma ali). Vale a
  mensagem dizer "recording from now" em vez de "nothing since the cursor", ou
  armar o anel na criação da aba.
- A árvore do x.com tem ~35 nós duplicados (a página tem duas regiões
  "Notifications alt+T"): é da página, não nosso.
