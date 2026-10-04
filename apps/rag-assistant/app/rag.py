"""Retrieval-augmented answers over the platform documentation."""

import re
import time
from dataclasses import dataclass, field
from typing import Protocol

from prometheus_client import Counter, Histogram

from .llm import LLM
from .store import Hit

STAGE_SECONDS = Histogram(
    "rag_stage_duration_seconds", "Time spent per answering stage.", ["stage"],
    buckets=(0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 40, 80),
)
TOKENS = Counter("rag_llm_tokens_total", "Tokens exchanged with the language model.", ["model", "direction"])
ANSWERS = Counter("rag_answers_total", "Questions answered, by outcome.", ["outcome"])

SYSTEM_PROMPT = """You are the developer assistant of paved-road, an internal developer platform.

Answer the question using only the documentation excerpts provided. Answer in the language of the question, even when the excerpts are in another language. Be concise and practical: give the commands or file paths a developer needs.

Cite the excerpts you rely on with their number, like [1] or [2]. If the excerpts do not contain the answer, say so plainly and suggest which part of the documentation to read. Never invent commands, file names or settings that do not appear in the excerpts."""

# Below this similarity, an excerpt is unrelated to the question.
MIN_SCORE = 0.2

# Small local models drift to English when the excerpts are in English. A
# closing instruction written in the question's language keeps them on track
# (measured on qwen3:1.7b: French questions got French answers again).
FRENCH = re.compile(
    r"[àâçéèêëîïôûùœ]|\b(comment|pourquoi|quel|quelle|quels|quelles|est-ce|mon|ma|mes|une|des|les|dans|avec|pour)\b",
    re.IGNORECASE,
)
CLOSING = {
    "fr": "Réponds en français, en citant les extraits utilisés comme [1], [2].",
    "en": "Answer in English, citing the excerpts you use as [1], [2].",
}


def language(question: str) -> str:
    return "fr" if len(FRENCH.findall(question)) >= 2 else "en"


class Retriever(Protocol):
    def search(self, embedding: list[float], limit: int) -> list[Hit]: ...


class Embeds(Protocol):
    def embed(self, texts: list[str]) -> list[list[float]]: ...


@dataclass
class Source:
    ref: int
    source: str
    heading: str
    score: float


@dataclass
class Answer:
    answer: str
    sources: list[Source]
    model: str
    timings: dict[str, float] = field(default_factory=dict)


def build_prompt(question: str, hits: list[Hit]) -> str:
    excerpts = "\n\n".join(
        f'<excerpt ref="{i}" source="{hit.source}" section="{hit.heading}">\n{hit.text}\n</excerpt>'
        for i, hit in enumerate(hits, start=1)
    )
    return (
        f"<documentation>\n{excerpts}\n</documentation>\n\n"
        f"Question: {question}\n\n{CLOSING[language(question)]}"
    )


class Assistant:
    def __init__(self, store: Retriever, embedder: Embeds, llm: LLM, top_k: int = 4):
        self._store = store
        self._embedder = embedder
        self._llm = llm
        self._top_k = top_k

    def ask(self, question: str) -> Answer:
        timings: dict[str, float] = {}

        start = time.perf_counter()
        [embedding] = self._embedder.embed([question])
        hits = [h for h in self._store.search(embedding, self._top_k) if h.score >= MIN_SCORE]
        timings["retrieval"] = time.perf_counter() - start
        STAGE_SECONDS.labels("retrieval").observe(timings["retrieval"])

        if not hits:
            ANSWERS.labels("no_context").inc()
            return Answer(
                answer="I could not find anything about this in the platform documentation.",
                sources=[], model="none", timings=timings,
            )

        start = time.perf_counter()
        try:
            completion = self._llm.complete(SYSTEM_PROMPT, build_prompt(question, hits))
        except Exception:
            ANSWERS.labels("error").inc()
            raise
        timings["generation"] = time.perf_counter() - start
        STAGE_SECONDS.labels("generation").observe(timings["generation"])
        TOKENS.labels(completion.model, "input").inc(completion.input_tokens)
        TOKENS.labels(completion.model, "output").inc(completion.output_tokens)
        ANSWERS.labels("answered").inc()

        return Answer(
            answer=completion.text,
            sources=[Source(i, h.source, h.heading, round(h.score, 3)) for i, h in enumerate(hits, start=1)],
            model=completion.model,
            timings={k: round(v, 3) for k, v in timings.items()},
        )
