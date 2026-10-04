# rag-assistant

Answers developer questions about paved-road from the platform documentation,
in the language of the question. Created with
`platformctl new-service rag-assistant --lang python`, see
[ADR 6](../../docs/adr/0006-rag-assistant.md) for the design and measurements.

```bash
bin/platformctl ask "Comment ajouter une base PostgreSQL à mon service ?"
```

| | |
|---|---|
| URL | https://rag-assistant.localhost (`POST /ask` with `{"question": "..."}`) |
| Dashboard | Grafana, "Service / rag-assistant" (HTTP panels plus answer latency, outcomes, tokens) |
| Status | `platformctl status rag-assistant` |

## How it works

1. The image ships `docs/` and both READMEs (built from the repository root).
2. After each deployment the `ingest` job splits the Markdown into sections,
   embeds them with a multilingual model baked into the image, and stores them
   in PostgreSQL with pgvector. Unchanged documents are skipped.
3. A question is embedded the same way, the 5 closest sections are retrieved,
   and a language model answers from them only, citing them as [1], [2].
   Questions with no relevant section are answered without calling the model.

## Platform capabilities used

All declared in `values.yaml`:

- `postgres` with the `vector` extension: database and `DATABASE_URL`.
- `jobs`: the post-deployment `ingest` job.
- `dashboard.panels`: the assistant's own panels.
- The shared Ollama model server (full profile).

## Using Claude instead of the local model

```bash
bin/platformctl seal rag-assistant ANTHROPIC_API_KEY
```

Then set `LLM_PROVIDER` to `anthropic` in `values.yaml`, commit and push.
`ANTHROPIC_MODEL` (default `claude-opus-5-5`) and `ANTHROPIC_EFFORT` (default
`low`) can be overridden the same way.

## Development

```bash
python -m venv .venv && .venv/bin/pip install -r requirements-dev.txt
.venv/bin/python -m pytest
```

Retrieval quality is measured after every indexing, on the reference
questions in `app/eval_questions.json`:

```bash
kubectl -n rag-assistant logs job/rag-assistant-ingest
```

Avoid running `python -m app.evaluate` inside the serving pod: a second copy
of the embedding model exceeds the container's memory limit and the service
gets restarted.
