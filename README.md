# notz-search

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: AGPL v3](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)

Serviço de **busca semântica** escrito em Go para as notas do Obsidian. Em vez de procurar por palavras
exatas, ele transforma cada nota em um vetor (*embedding*), guarda esses vetores no
[Qdrant](https://qdrant.tech/) e encontra as notas cujo **significado** é mais próximo da sua pergunta.

Buscar por `sobremesa com cenoura` encontra a nota "Bolo de cenoura" mesmo que a palavra "sobremesa" não
apareça nela.

> **Sobre o projeto.** O notz-search é o *search-service* do **Notz**, um projeto pessoal de estudo. Os
> objetivos são praticar Go idiomático (perto da stdlib, sem framework escondendo o `net/http`), entender
> busca semântica na prática e, mais adiante, combinar similaridade vetorial com os `[[wikilinks]]` que o
> Obsidian já cria entre as notas.

## Sumário

- [Como funciona](#como-funciona)
- [Funcionalidades e roadmap](#funcionalidades-e-roadmap)
- [Arquitetura](#arquitetura)
- [Início rápido](#início-rápido)
- [Configuração](#configuração)
- [Referência da API](#referência-da-api)
- [Testes](#testes)
- [Solução de problemas](#solução-de-problemas)
- [Contribuindo](#contribuindo)
- [Licença](#licença)

## Como funciona

```text
                     Cliente
                        |  POST /notes/index   GET /search
                        v
+-----------------------------------------------+
|                 Handlers HTTP                 |
+-----------------------------------------------+
                        |
                        v
+-----------------------------------------------+
|                  NoteService                  |
+-----------------------------------------------+
           | Embed / EmbedQuery      | Upsert / Search
           v                         v
+---------------------+   +---------------------+
|  EmbeddingProvider  |   |    NoteRepository   |   <- interfaces (domain)
+---------------------+   +---------------------+
           ^ implementado por        ^ implementado por
+---------------------+   +---------------------+
|   Cohere / Gemini   |   |        Qdrant       |   <- adapters (infra)
+---------------------+   +---------------------+
```

**Indexação** (`POST /notes/index`)

1. O handler valida o corpo da requisição (o `id` precisa ser um UUID e o `text` não pode ser vazio).
2. O `NoteService` pede ao provider de embeddings o vetor da nota (título + texto).
3. O repositório grava no Qdrant um *ponto* com o vetor e um *payload* contendo `title`, `text`,
   `links_to`, `created_at` e `updated_at`. O `id` da nota vira o id do ponto.

**Busca** (`GET /search?q=...`)

1. O `NoteService` pede o vetor da pergunta ao provider.
2. O Qdrant devolve as notas mais próximas pela distância de cosseno, da mais para a menos similar.

> **Por que existem dois modos de embedding?** Modelos de recuperação são *assimétricos*: uma pergunta
> curta e um documento longo só ficam próximos no espaço vetorial se cada um for embedado no modo certo.
> Por isso a interface tem `Embed` (notas, modo *documento*) e `EmbedQuery` (perguntas, modo *consulta*).
> Na Cohere isso é `input_type: search_document` / `search_query`; no Gemini, `RETRIEVAL_DOCUMENT` /
> `RETRIEVAL_QUERY`. Embedar a pergunta como documento piora o ranking de forma mensurável.

## Funcionalidades e roadmap

**Disponível**

- [x] Indexação de notas com embedding e upsert no Qdrant
- [x] Busca semântica com limite configurável de resultados
- [x] Providers de embedding intercambiáveis: **Cohere** (padrão) e **Gemini**
- [x] Criação automática da collection no Qdrant na inicialização
- [x] Validação de entrada e mapeamento de erros para status HTTP adequados
- [x] Suporte a Qdrant local (Docker) e em nuvem (Qdrant Cloud, com API key e TLS)

**Planejado**

- [ ] Remoção de notas do índice
- [ ] Sugestão de links que faltam (vizinhos semânticos que ainda não estão linkados)
- [ ] Expansão do contexto da busca pelos links diretos da nota encontrada
- [ ] Dados para visualização em grafo (links reais e arestas semânticas)
- [ ] Lista de notas ordenada por proximidade a uma nota de referência e agrupada por cluster
- [ ] Detecção de notas duplicadas ou redundantes
- [ ] Linha do tempo semântica (datas cruzadas com clusters)

O escopo da v1 do Notz é somente leitura (busca, grafo, lista, timeline); a edição de notas fica para a v2.
O **notes-service**, que será a fonte da verdade das notas e extrairá os `[[wikilinks]]`, é um serviço
separado e ainda não faz parte deste repositório.

## Arquitetura

O código segue uma arquitetura em camadas com **inversão de dependência**: o domínio define interfaces
(*ports*) e a infraestrutura as implementa (*adapters*). Nada no domínio ou no service importa Qdrant,
Cohere ou Gemini.

```
cmd/
  main.go                    Composição: lê a config, monta as dependências e sobe o servidor HTTP
internal/
  domain/                    Entidade Note, interfaces (NoteRepository, EmbeddingProvider) e erros sentinela
  service/                   NoteService: orquestra embedding + armazenamento/busca
  handlers/                  Adapter de entrada: endpoints HTTP, validação e mapeamento de erros
  infra/                     Adapters de saída
    embeddings/              CohereProvider e GeminiProvider (chamadas diretas via net/http)
    qdrant/                  NoteRepository sobre o client oficial do Qdrant
```

| Camada | Papel | Depende de |
|---|---|---|
| `domain` | Regras e contratos do sistema | nada |
| `service` | Casos de uso: indexar e buscar | `domain` |
| `handlers` | Traduz HTTP para chamadas ao service | `service`, `domain` |
| `infra/*` | Implementa os contratos do domínio com tecnologias concretas | `domain` |

`handlers` é adapter de **entrada** (o mundo externo chama a aplicação); `infra` são adapters de **saída**
(a aplicação chama serviços externos). Por isso ficam em pastas separadas.

**Decisões de design**

- **[Chi](https://github.com/go-chi/chi)** como roteador: fica próximo do `net/http` (handlers com a
  assinatura padrão `func(http.ResponseWriter, *http.Request)`), sem Context proprietário que prenda a
  lógica de negócio ao framework.
- **Providers atrás de uma interface** (`domain.EmbeddingProvider`): trocar de Cohere para Gemini é uma
  variável de ambiente, e adicionar outro provider não toca em service, handlers nem repositório.
- **Sem SDKs de embedding**: os providers usam `net/http` e `encoding/json` diretamente, mantendo poucas
  dependências.
- **Erros sentinela no domínio** (`ErrEmbeddingGeneration`, `ErrUpsert`, `ErrSearch`, `ErrInvalidPayload`,
  `ErrEnsureCollection`): as camadas embrulham com `%w` e os handlers decidem o status HTTP com `errors.Is`.

## Início rápido

### Pré-requisitos

- **Go 1.22.2** ou superior
- Uma instância do **Qdrant**: local via Docker ou em nuvem
- Uma chave de API de um provider de embeddings. A [Cohere](https://dashboard.cohere.com/api-keys) tem
  *trial key* gratuita e é o provider padrão

### Passo a passo

```bash
# 1. Clonar e entrar no projeto
git clone https://github.com/Kevinmso/notz-search.git
cd notz-search

# 2. Subir um Qdrant local (portas: 6333 = REST, 6334 = gRPC, usada pela aplicação)
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# 3. Configurar o ambiente (em outro terminal)
cp .env.example .env
# edite o .env e preencha COHERE_API_KEY; deixe QDRANT_* vazio para usar o Qdrant local

# 4. Rodar a aplicação
go run ./cmd
```

A aplicação sobe em `http://localhost:8080` e cria a collection `notes` no Qdrant se ela ainda não existir.

### Testando

```bash
# Saúde
curl http://localhost:8080/health

# Indexar uma nota (o id precisa ser um UUID)
curl -X POST http://localhost:8080/notes/index \
  -H "Content-Type: application/json" \
  -d '{
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "Bolo de cenoura",
    "text": "Receita de bolo de cenoura: bata cenoura, ovos e óleo no liquidificador, depois misture farinha e fermento e asse.",
    "links_to": ["receitas"]
  }'

# Buscar por significado
curl -G http://localhost:8080/search \
  --data-urlencode "q=sobremesa com cenoura" \
  --data-urlencode "limit=5"
```

Há também uma collection pronta para **Postman/Insomnia** em
[`notz-search.postman_collection.json`](notz-search.postman_collection.json).

## Configuração

A configuração é feita por variáveis de ambiente. Se existir um arquivo `.env` na raiz, ele é carregado
automaticamente na inicialização (copie o [`.env.example`](.env.example) como ponto de partida). O `.env`
está no `.gitignore`: **nunca versione chaves de API**.

| Variável | Obrigatória | Padrão | Descrição |
|---|---|---|---|
| `EMBEDDING_PROVIDER` | não | `cohere` | Provider de embeddings: `cohere` ou `gemini`. Qualquer outro valor impede a inicialização |
| `COHERE_API_KEY` | se `cohere` | — | Chave da API da Cohere |
| `GEMINI_API_KEY` | se `gemini` | — | Chave da API do Google Gemini |
| `QDRANT_HOST` | não | `localhost` | **Somente o hostname**, sem `https://` e sem porta |
| `QDRANT_API_KEY` | não | vazio | Chave do cluster. Necessária no Qdrant Cloud, desnecessária no Docker local |
| `QDRANT_USE_TLS` | não | `false` | `true` para conectar via TLS (Qdrant Cloud exige) |

Valores fixos no código: servidor HTTP em `:8080`, porta gRPC do Qdrant `6334` e collection `notes`.

### Usando o Qdrant Cloud

```env
QDRANT_HOST=<id-do-cluster>.<regiao>.aws.cloud.qdrant.io
QDRANT_API_KEY=<sua-chave>
QDRANT_USE_TLS=true
```

### Providers e dimensão dos vetores

A collection é criada com a dimensão do provider ativo e distância de cosseno:

| Provider | Modelo | Dimensões |
|---|---|---|
| `cohere` | `embed-multilingual-v3.0` (bom suporte a português) | 1024 |
| `gemini` | `gemini-embedding-001` | 3072 |

> **Atenção ao trocar de provider.** O `EnsureCollection` só cria a collection quando ela **não existe**;
> ele não confere a dimensão de uma collection já criada. Trocar de `cohere` para `gemini` (ou o inverso)
> com a collection `notes` existente faz os upserts falharem por diferença de dimensão. Apague a collection
> (ou aponte para outro cluster) e reindexe as notas. Vetores de providers diferentes também não são
> comparáveis entre si.

## Referência da API

### `GET /health`

Verifica se o serviço está de pé. Responde `200` com o corpo `ok`.

### `POST /notes/index`

Indexa uma nota: gera o embedding e grava no Qdrant. É um **upsert**: indexar um `id` que já existe
sobrescreve a nota anterior (inclusive `created_at`, que é definido pelo servidor a cada indexação).

**Corpo (JSON)**

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---|---|
| `id` | string | sim | UUID da nota (é usado como id do ponto no Qdrant) |
| `title` | string | não | Título; entra no texto que é embedado |
| `text` | string | sim | Conteúdo da nota |
| `links_to` | string[] | não | Notas linkadas por esta (`[[wikilinks]]`) |

**Respostas**

| Status | Quando |
|---|---|
| `201 Created` | Nota indexada (sem corpo) |
| `400 Bad Request` | JSON inválido, `text` vazio ou `id` que não é UUID |
| `502 Bad Gateway` | Falha no provider de embeddings ou no Qdrant |
| `500 Internal Server Error` | Qualquer outra falha |

### `GET /search`

Busca as notas semanticamente mais próximas da pergunta.

**Parâmetros de query**

| Parâmetro | Obrigatório | Padrão | Descrição |
|---|---|---|---|
| `q` | sim | — | Pergunta ou texto de busca |
| `limit` | não | `10` | Máximo de resultados; precisa ser um inteiro ≥ 1 |

**Resposta `200 OK`**: array JSON ordenado da nota mais para a menos similar (o score não é exposto).
Uma busca sem resultados devolve `[]`.

```json
[
  {
    "ID": "550e8400-e29b-41d4-a716-446655440000",
    "Title": "Bolo de cenoura",
    "Text": "Receita de bolo de cenoura: bata cenoura, ovos e óleo no liquidificador...",
    "LinksTo": ["receitas"],
    "CreatedAt": "2026-09-20T10:20:34-03:00",
    "UpdatedAt": "2026-09-20T10:20:34-03:00"
  }
]
```

> As chaves da resposta ainda usam os nomes dos campos Go (`ID`, `LinksTo`...), enquanto o corpo do
> `POST /notes/index` usa `snake_case`. A padronização do formato de resposta está no roadmap de API.

**Erros**

| Status | Quando |
|---|---|
| `400 Bad Request` | `q` ausente ou vazio, ou `limit` não numérico ou menor que 1 |
| `502 Bad Gateway` | Falha no provider de embeddings ou no Qdrant |
| `500 Internal Server Error` | Qualquer outra falha (inclui payload corrompido no índice) |

## Testes

```bash
go test ./...
```

Nenhum teste precisa de chave de API, de rede ou de um Qdrant no ar:

- **Service e handlers** usam fakes das interfaces do domínio (`NoteRepository`, `EmbeddingProvider`) e
  `httptest`.
- **Providers de embedding** são testados contra um `httptest.Server`, conferindo o request enviado e o
  tratamento de erros da API.
- **Conversão Note ↔ payload do Qdrant** é testada como funções puras, isoladas da rede.

## Solução de problemas

| Sintoma | Causa provável e solução |
|---|---|
| `too many colons in address` ao iniciar | `QDRANT_HOST` contém `https://` e/ou a porta. Use só o hostname |
| `PermissionDenied ... forbidden` do Qdrant | `QDRANT_API_KEY` ausente ou incorreta, ou faltou `QDRANT_USE_TLS=true` em um cluster na nuvem |
| `502` em `/notes/index` ou `/search` | O provider de embeddings ou o Qdrant falhou. O motivo real aparece no **log do servidor** (o cliente recebe só uma mensagem genérica). Confira chaves e cotas |
| Erro de dimensão de vetor no upsert | A collection foi criada com outro provider. Veja [Providers e dimensão dos vetores](#providers-e-dimensão-dos-vetores) |
| `id must be a valid UUID` | O `id` da nota precisa ser um UUID, exigência do Qdrant para ids de ponto |
| `address already in use` | Outro processo usa a porta 8080. Encerre-o (`lsof -ti:8080`) |
| Aviso `Client version is not compatible with server version` | O client Go do Qdrant e o servidor têm versões distantes. É apenas um aviso e não impede o funcionamento |

## Contribuindo

- **Branches e PRs**: cada mudança em uma branch própria, com PR seguindo o
  [template](.github/PULL_REQUEST_TEMPLATE.md). Mudanças dependentes podem ser empilhadas (*stacked PRs*),
  cada uma contra a branch anterior.
- **Commits** no estilo *Conventional Commits*: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`.
- **Versionamento** por [SemVer](https://semver.org/lang/pt-BR/) com releases no GitHub. Enquanto a versão
  for `0.x`, a API ainda pode mudar; funcionalidade nova incrementa o *minor* e correção incrementa o *patch*.
- Rode `go test ./...` e `go vet ./...` antes de abrir o PR.
- O arquivo [`CLAUDE.md`](CLAUDE.md) reúne o contexto e as decisões do projeto (usado como guia pelo
  Claude Code).

## Licença

Distribuído sob a licença **GNU AGPL v3**. Veja o arquivo [LICENSE](LICENSE).
