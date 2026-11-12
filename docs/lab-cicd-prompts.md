# Prompts utilizados (texto integral)

Prompts P1 e P2 (meus, ao Claude Code) estão no relatório de entrega. Abaixo estão os prompts que o Claude Code enviou ao subagente, como foram enviados. A única diferença é que, no P3, o caminho local do binário do actionlint (numa pasta temporária) foi trocado pela palavra "actionlint".

## P3 — Criação da pipeline (subagente)

Contexto: estou fazendo um laboratório de DevOps (Instituto J&F) e preciso criar uma pipeline CI/CD com GitHub Actions para este repositório Go (diretório de trabalho: /home/kevinmso/projetos-pessoais/notz-search, branch ci/github-actions-pipeline já criada a partir de origin/main). Leia README.md, CLAUDE.md, go.mod e cmd/main.go para entender o projeto: é uma API HTTP em Go (chi) de busca semântica que usa Qdrant Cloud e APIs de embeddings (Cohere/Gemini), configurada por variáveis de ambiente, com endpoint GET /health que responde "ok".

Requisitos do enunciado:
- CI roda a cada push na main e também em pull_request para a main;
- CI faz checkout, configura o ambiente (Go), instala dependências, e executa teste/validação;
- CD roda após push na main, SOMENTE se a CI concluir com sucesso;
- CD faz o deploy de uma nova versão da aplicação num ambiente acessível, e o fluxo CI -> CD -> aplicação publicada tem que ser automatizado.

Decisão já tomada: deploy no Render como Web Service usando Docker (o Render faz o build do Dockerfile do repo), disparado por Deploy Hook (URL guardada no secret RENDER_DEPLOY_HOOK_URL do GitHub). A URL pública do serviço ficará na variável de repositório APP_URL (vars.APP_URL) e o CD deve, depois de disparar o deploy, verificar com retry/espera que GET $APP_URL/health responde "ok", falhando o job se não responder.

Tarefa: crie
1. .github/workflows/ci-cd.yml (jobs `ci` e `cd`, cd com `needs: ci` e `if` que só deixa rodar em push na main; permissões mínimas; actions oficiais com versões fixas por tag major);
2. Dockerfile multi-stage para o serviço (build com a versão do Go do go.mod, binário estático, imagem final mínima e sem rodar como root) e .dockerignore (não pode copiar .env);
3. ajuste mínimo em cmd/main.go para ouvir na porta da variável PORT (fallback 8080), pois o Render injeta PORT.
Não faça commit nem push. Valide o que for possível localmente: `gofmt -l .`, `go vet ./...`, `go build ./...`, `go test ./...` e o linter de workflows (actionlint; docker NÃO está disponível neste ambiente, então diga explicitamente o que você não conseguiu validar). No final, responda com: lista de arquivos criados/alterados, decisões tomadas e o que ficou sem validação.

## P4 — Validar o Dockerfile no CI e testar port() (subagente)

Prompt 2 do laboratório: revisei sua entrega. As tags das actions (checkout@v5, setup-go@v6) e das imagens (golang:1.27.1-alpine, distroless/static-debian12:nonroot) existem, e o actionlint passou. O ponto fraco é que o Dockerfile não foi validado em lugar nenhum (sem docker aqui), então um erro nele só apareceria no deploy do Render, depois do merge. Faça duas mudanças, sem commit e sem push:
1. No job `ci` de .github/workflows/ci-cd.yml, adicione um step que faz `docker build` do Dockerfile (sem push, tag local). Assim a imagem é validada em todo push e PR, antes do CD. Não adicione login em registry nem secrets.
2. Crie um teste unitário para a função port() de cmd/main.go (PORT vazio retorna 8080; PORT definido retorna o valor), usando t.Setenv, em cmd/main_test.go.
Rode gofmt -l ., go vet, go test ./... e o actionlint de novo. Responda com os arquivos alterados e o que continua sem validação.
