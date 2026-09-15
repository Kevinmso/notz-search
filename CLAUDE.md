# Notz

Projeto pessoal de estudo, usando Go, com o objetivo de aprender busca semântica
aplicada às próprias notas do Obsidian, usando Qdrant como banco vetorial.

## Objetivo

Estudar na prática: Go (idiomático, sem framework pesado escondendo a stdlib),
busca semântica/embeddings, e como combinar similaridade vetorial com a estrutura
de links explícitos que o Obsidian já cria entre notas ([[wikilinks]]).

## Arquitetura

Dois serviços:

- **notes-service** — a fonte da verdade. CRUD das notas, extrai os `[[wikilinks]]`
  ao salvar/reindexar uma nota.
- **search-service** — gera embeddings, consulta o Qdrant, devolve resultados de
  busca/cluster/grafo/timeline. É o serviço por onde o desenvolvimento vai começar.

Comunicação entre os dois: começa **síncrona** (chamada direta HTTP após salvar
uma nota), com espaço pra evoluir depois pra fila/evento assíncrono (ex.: outbox
pattern ou uma fila tipo NATS/RabbitMQ) sem precisar redesenhar nada.

Repositórios: pode ser em repos separados (`notz-notes-service`, `notz-search-service`,
`notz-web`) ou um monorepo único `notz/` com pastas `notes-service/`, `search-service/`,
`web/` — ainda em aberto, mas o desenvolvimento começa pelo `search-service`
independente da escolha.

## Escopo da v1

- **v1**: só leitura das notas pela interface (busca, grafo, lista, timeline).
- **v2**: edição das notas pela interface.

## Stack do search-service

- Linguagem: **Go**
- Framework HTTP: **Chi** — escolhido em vez de Gin/Echo/Fiber porque:
  - Fica próximo do `net/http` puro (handlers `func(w http.ResponseWriter, r *http.Request)`),
    sem um Context proprietário — isso facilita manter a lógica de negócio desacoplada
    do framework (inversão de dependência / arquitetura limpa).
  - Performance de roteamento não é um fator real aqui — o gargalo do serviço é
    sempre a chamada ao Qdrant/API de embeddings (milissegundos), não o roteamento
    (nanosegundos).
  - Fiber foi descartado por quebrar compatibilidade com `net/http` (usa fasthttp),
    criando lock-in de ecossistema sem ganho real pro caso de uso.
- Cliente Qdrant: `github.com/qdrant/go-client`
- Embeddings: via API (OpenAI/Cohere) inicialmente — mais rápido pra sair do zero
  que rodar um modelo local
- Qdrant local via Docker (`docker run qdrant/qdrant`) pra desenvolvimento

### Estrutura inicial sugerida

```
search-service/
  cmd/main.go                    → sobe o servidor HTTP
  internal/domain/               → entidades e interfaces (ports), sem dependência de infra
    note.go                      → struct Note
  internal/infra/
    qdrant/                      → cliente/wrapper do Qdrant (conectar, criar coleção, upsert, buscar)
    embeddings/                  → função que transforma texto em vetor
  internal/handlers/             → os endpoints HTTP
  docker-compose.yml             → Qdrant local pra dev
  go.mod
```

Adapters de infra (`internal/infra/*`) implementam interfaces definidas em `internal/domain`
(ex. `NoteRepository`) — mantém a lógica de negócio desacoplada do Qdrant/provider de embeddings.

### Primeiros endpoints (MVP, antes de qualquer feature extra)

- `POST /notes/index` — recebe id + texto de uma nota, gera embedding, faz upsert no Qdrant
- `GET /search?q=` — recebe uma pergunta, gera embedding, busca similares, devolve resultado

## Features planejadas (além da busca base)

1. **Sugestão de links que faltam** — compara os vizinhos semânticos de uma nota
   com os links reais dela e sugere conexões que ainda não existem.
2. **Expansão de contexto de busca via links** — ao achar uma nota relevante,
   também considera as notas diretamente linkadas a ela como contexto extra.
3. **Visualização em grafo** — mostra os links reais e as "arestas semânticas"
   sugeridas (não linkadas ainda) em estilo visual diferente.
4. **Lista de notas ordenável semanticamente** — tela secundária, com dois modos:
   ordenar por proximidade a uma nota de referência, e agrupar por cluster semântico.
5. **Detecção de notas duplicadas/redundantes** — identifica notas semanticamente
   quase idênticas e sugere unir.
6. **Linha do tempo semântica** — cruza data de criação/edição com o cluster de
   cada nota pra visualizar a evolução dos temas ao longo do tempo.

## Modelo de dados (payload no Qdrant, por nota)

Cada nota vira um vetor no Qdrant, com payload contendo pelo menos:

- `id`, `titulo`, `texto` (ou referência a onde o texto completo mora, se a fonte
  da verdade for separada — ex. arquivo/banco no notes-service)
- `links_para: []` — ids/títulos das notas linkadas por esta nota
- `criado_em`, `editado_em`
- `cluster_id` — recalculado periodicamente, não a cada busca

## Decisões em aberto

- Repos separados vs. monorepo (ver "Arquitetura" acima)
- Onde hospedar o Qdrant e o search-service quando sair do ambiente local
  (Qdrant Cloud free tier + Railway/Fly.io/Render são as opções cogitadas)
- Mecanismo de fila/evento a usar quando a comunicação entre serviços deixar de
  ser síncrona