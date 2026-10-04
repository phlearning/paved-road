import pytest

from app.config import Settings
from app.llm import AnthropicLLM, OllamaLLM, from_settings


def settings(provider):
    return Settings(
        database_url=None, llm_provider=provider, ollama_url="http://ollama:11434", ollama_model="m",
        anthropic_model="claude-opus-5-5", anthropic_effort="low", embedding_model="e",
        embedding_cache_dir=None, top_k=4, corpus_dir="/c",
    )


def test_provider_selection(monkeypatch):
    monkeypatch.setenv("ANTHROPIC_API_KEY", "test")
    assert isinstance(from_settings(settings("ollama")), OllamaLLM)
    assert isinstance(from_settings(settings("anthropic")), AnthropicLLM)
    with pytest.raises(ValueError):
        from_settings(settings("other"))
