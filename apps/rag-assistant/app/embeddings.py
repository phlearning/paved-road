"""Multilingual sentence embeddings, computed in-process.

The model is baked into the image (see the Dockerfile), so it runs offline
and in both platform profiles, with or without the shared LLM server."""

from fastembed import TextEmbedding


class Embedder:
    def __init__(self, model_name: str, cache_dir: str | None = None):
        self._model = TextEmbedding(model_name, cache_dir=cache_dir)
        self.dimensions = len(self.embed(["dimension probe"])[0])

    def embed(self, texts: list[str]) -> list[list[float]]:
        return [vector.tolist() for vector in self._model.embed(texts)]
