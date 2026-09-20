# 2026-09-16 — site-exceptions: a permissão ganha o "+ Add" da referência

Pergunta do dono ("estamos inventando engenharia pro browser?") + "sigo sua
recomendação". Fui checar antes de construir a tabela que eu tinha proposto e a
conclusão mudou a fatia:

**A tabela da referência já existe, transposta.** Nosso diálogo "Site settings"
tem as mesmas células: uma linha por *kind* com o tri-estado (Platform default |
Allow | Block | Ask) — que é a "Default row" — e embaixo a lista por site com
Reset. O que faltava de verdade era o **"+ Add"**: criar uma exceção por site à
mão, em vez de só reagir a um prompt. Nada de coluna Browsing/Downloads/Uploads:
isso exigiria gate novo (o Browsing é o par domains+tier; Downloads é global;
Uploads não tem gancho) — exatamente o que a recomendação diz para **não** fazer.

O que entrou (UI sobre o que já existia — `POST /api/browser/permissions` já
aceitava `{origin, kind, decision, standing}`, e a página já empurra rows com
`standing` para o shell vivo, então a exceção passa a valer na hora):

- schema `browserSiteSchema` + `BROWSER_PERMISSION_KINDS` (os 10 kinds da store,
  não só os 6 com título: um kind que o shell mapeia e o schema recusa é uma
  linha que ninguém consegue escrever);
- a linha "Add an exception" (input + kind + decision + Add) acima do log,
  com erro de validação em texto e o botão desabilitado sem site;
- normalização igual à do campo de domínios: `https://*.Example.com/path` →
  `*.example.com`.

Verificado no scratch (screenshots lidos): `meet.example.com` + microphone +
Ask e `*.example.com` + camera + Always allow viraram linhas `saved` com o
`standing: true` na API; site inválido mostra a mensagem e marca o campo; botão
desabilitado sem site; `__picodeOverlayAudit()` ok. `node --test` 394. A
primeira versão tinha dois defeitos que o print pegou: o botão caía para a
segunda linha e o cabeçalho dizia "Recent decisions" acima do formulário — os
dois corrigidos antes do commit.

Deixado de fora de propósito (e por quê, para o próximo que olhar o print da
referência): colunas Browsing/Downloads/Uploads exigem enforcement novo; a
coluna Uploads só faz sentido quando existir gancho do seletor de arquivo.

## Next up

- Owner: abrir Settings ▸ Browser ▸ Site settings, adicionar uma exceção e
  verificar que uma pergunta real do site a respeita (com o app aberto, a linha
  chega ao shell na hora).

## Debts

- O kind escolhido no menu é o nome cru da store (`autoplay`, `fonts`,
  `filesystem`…): os seis com título têm descrição, os outros quatro aparecem
  sem explicação. Vale um título para cada quando alguém precisar deles.
