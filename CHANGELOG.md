# Changelog

Todas as mudanças visíveis do projeto são documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/),
e o projeto adere a [SemVer](https://semver.org/lang/pt-BR/).

**Contrato dos agentes:** todo commit com mudança visível ao usuário DEVE
adicionar uma entrada na seção `[Não lançado]`. Entradas são escritas em
inglês (o changelog acompanha o release, que é global).

## [Não lançado]

## [0.1.0] - 2026-08-23

### Adicionado

- **Bootstrap do projeto**: repositório público, licença MIT, CI (GitHub Actions
  com gofmt/vet/test/build), Makefile.
- **Sistema de documentação vivo**: `docs/` com arquitetura, filosofia,
  benchmarks de engenharia e UI/UX, handoff (`docs/handoff.md`) e ADRs
  (`docs/decisions/`) — com contrato explícito de que a documentação evolui
  junto com o código (ver `AGENTS.md`).
- **Harness para agentes Pi**: `AGENTS.md` na raiz (contrato operacional),
  skills de qualidade em `.pi/skills/` (`quality-gate`, `uiux-review`,
  `handoff-update`) e settings do projeto em `.pi/`.
- **Esqueleto do servidor Go**: binário `picode` com UI embutida via
  `go:embed`, endpoints `/api/health` e `/api/version`, página inicial
  dark-first de placeholder com health check ao vivo.
- **Documentação de decisões iniciais (ADRs)**: app no browser servido por
  binário único em Go (0001), controle dual-channel tmux+RPC (0002),
  dependência do `pi` instalado pelo usuário (0003).

[Não lançado]: https://github.com/cfpperche/picode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
