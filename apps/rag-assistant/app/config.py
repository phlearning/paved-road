"""Settings read from the environment, set by values.yaml and the platform."""

import os
from dataclasses import dataclass

DEFAULT_EMBEDDING_MODEL = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"


@dataclass(frozen=True)
class Settings:
    database_url: str | None
    # "ollama" (platform model server, full profile) or "anthropic" (hosted).
    llm_provider: str
    ollama_url: str
    ollama_model: str
    anthropic_model: str
    anthropic_effort: str
    embedding_model: str
    embedding_cache_dir: str | None
    top_k: int
    corpus_dir: str

    @classmethod
    def from_env(cls) -> "Settings":
        env = os.environ.get
        return cls(
            database_url=env("DATABASE_URL"),
            llm_provider=env("LLM_PROVIDER", "ollama"),
            ollama_url=env("OLLAMA_URL", "http://ollama.ai.svc:11434"),
            ollama_model=env("OLLAMA_MODEL", "qwen3:1.7b"),
            anthropic_model=env("ANTHROPIC_MODEL", "claude-opus-5-5"),
            anthropic_effort=env("ANTHROPIC_EFFORT", "low"),
            embedding_model=env("EMBEDDING_MODEL", DEFAULT_EMBEDDING_MODEL),
            embedding_cache_dir=env("EMBEDDING_CACHE_DIR"),
            top_k=int(env("TOP_K", "5")),
            corpus_dir=env("CORPUS_DIR", "/srv/corpus"),
        )
