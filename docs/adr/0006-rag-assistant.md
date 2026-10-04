# 6. Documentation assistant (RAG) as a platform workload

- Status: accepted
- Date: 2026-10-04

## Context

Developers ask the platform team the same questions: how to create a
service, add a database, store a secret, fix a certificate warning. The
answers are in `docs/`, but searching them takes time, and many developers
ask in French while the documentation is in English.

The assistant is also the first real workload of the golden path: it needs a
database, a batch job and a language model, which makes it a good test of the
platform's self-service capabilities.

## Decision

`rag-assistant` is created with `platformctl new-service --lang python` and
deployed like any other service. It only uses capabilities the platform
offers to every team.

| Concern | Choice | Why |
|---|---|---|
| Vector store | PostgreSQL 18 + pgvector, through `postgres.enabled` | A database the bank already knows how to run, back up and audit; no new datastore to operate |
| Embeddings | `paraphrase-multilingual-MiniLM-L12-v2` via fastembed (ONNX), baked into the image | Runs on CPU in both profiles, offline; matched French questions to English sections 4 out of 4 in a first test |
| Chunking | Markdown sections, split at paragraphs above 1,200 characters, code blocks kept whole, heading path kept with each chunk | Excerpts are self-contained and cite a precise section |
| Indexing | `python -m app.ingest` as a post-deployment Job (`jobs` capability) | Every new image re-indexes the documentation it ships; unchanged documents are skipped by digest |
| Generation | Pluggable: the platform's Ollama server (`qwen3:1.7b`, full profile) or Claude through the API (`claude-opus-5-5`, low effort) | Local by default; the API when quality matters or no local model fits |
| Guardrails | Answers only from excerpts, citations required, no model call when no excerpt scores above 0.2 | Off-topic questions are refused without spending tokens or inventing an answer |
| Observability | Stage latency, answers by outcome and tokens, as panels in the service dashboard | Same dashboard as every service, plus three panels declared in `values.yaml` |

The image is built from the repository root (`build.context`) so it carries
`docs/` and both READMEs, and rebuilt when they change (`build.watch`).

## Measurements

Retrieval is measured on 12 questions (9 in French, 3 in English), each
listing the sections that answer it. The ingest job runs this evaluation after
every indexing and logs recall@1, @3 and @5
(`kubectl -n rag-assistant logs job/rag-assistant-ingest`).

| Change | Documents | recall@1 | recall@3 | recall@k |
|---|---|---|---|---|
| First version (k=4) | 12 | 67% | 83% | 83% |
| Explicit headings ("Scaffold it" became "Create the service with platformctl new-service") | 12 | 67% | 83% | 100% |
| Context raised to 5 excerpts; READMEs and ADR 6 added to the corpus | 14 | 67% | 75% | 92% |
| Expected sections widened: the READMEs' own "create a service" sections are valid answers too | 14 | 67% | 75% | 100% |

The third row is a regression caught by the evaluation: the new French
README section answered the French question better than the English guide,
and the reference set only accepted the guide. The correct sections are
almost always retrieved, but often not first; a re-ranking step would be the
next improvement.

Generation, on a 4-vCPU laptop VM:

| Model | Speed | Quality on 3 reference questions |
|---|---|---|
| `qwen3:1.7b` (Ollama) | about 35 tokens/s, 3 to 14 s per answer | Correct commands and settings when the excerpt is retrieved, but sometimes answers in English to a French question and once invented a command |
| `qwen3:4b` (Ollama) | not measured | Killed for lack of memory at a 4 GiB limit on an 8 GB VM |

## Consequences

- The assistant runs fully offline on the full profile; answer quality is
  limited by what a 1.7B-parameter model can do on CPU.
- With an Anthropic API key sealed in `values.yaml` (`platformctl seal`) and
  `LLM_PROVIDER=anthropic`, the same retrieval feeds a much stronger model.
  This is also the only option on the lite profile.
- The retrieval evaluation runs in the ingest job, against the live index, but
  does not fail the deployment; gating on it in CI needs a test database there.
- Running the evaluation inside the serving container loads a second copy of
  the embedding model and exceeds its 1 GiB limit: it lives in the job instead.
- Documentation quality directly drives answer quality: vague headings hurt
  retrieval, as measured above.
