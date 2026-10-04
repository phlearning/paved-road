from dataclasses import asdict
from functools import cache

from fastapi import Depends, FastAPI, HTTPException
from pydantic import BaseModel, Field

from . import llm
from .config import Settings
from .embeddings import Embedder
from .metrics import instrument
from .rag import Assistant
from .store import Store

app = FastAPI(title="rag-assistant")
instrument(app)


class Question(BaseModel):
    question: str = Field(min_length=3, max_length=2000)


@cache
def settings() -> Settings:
    return Settings.from_env()


def get_store() -> Store | None:
    url = settings().database_url
    return Store(url) if url else None


@cache
def get_assistant() -> Assistant:
    store = get_store()
    if store is None:
        raise HTTPException(503, "DATABASE_URL is not configured")
    s = settings()
    return Assistant(store, Embedder(s.embedding_model, s.embedding_cache_dir), llm.from_settings(s), s.top_k)


@app.get("/")
def root() -> dict[str, str]:
    return {"service": "rag-assistant", "message": "Ask questions about the platform at POST /ask"}


@app.post("/ask")
def ask(body: Question, assistant: Assistant = Depends(get_assistant)) -> dict:
    try:
        return asdict(assistant.ask(body.question))
    except llm.LLMError as err:
        raise HTTPException(502, str(err)) from err


@app.get("/healthz", include_in_schema=False)
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/readyz", include_in_schema=False)
def readyz(store: Store | None = Depends(get_store)) -> dict[str, str]:
    if store is not None:
        try:
            store.ping()
        except Exception as err:
            raise HTTPException(503, f"database unavailable: {err}") from err
    return {"status": "ready"}
