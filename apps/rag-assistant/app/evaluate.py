"""Measures retrieval quality on a fixed set of questions.

Each question lists the document sections that answer it. The score is
recall@k: the share of questions with one of those sections in the top k
excerpts. The ingest job runs it after every indexing and logs the result;
locally: python -m app.evaluate (needs DATABASE_URL).
"""

import json
import logging
from dataclasses import dataclass
from pathlib import Path

from .config import Settings
from .embeddings import Embedder
from .store import Store

QUESTIONS = Path(__file__).with_name("eval_questions.json")

log = logging.getLogger("evaluate")


@dataclass
class Report:
    total: int
    found: dict[int, int]
    misses: list[str]

    def recall(self, k: int) -> float:
        return self.found[k] / self.total if self.total else 0.0


def evaluate(store: Store, embedder: Embedder, top_k: int) -> Report:
    cases = json.loads(QUESTIONS.read_text())
    ks = sorted({1, 3, top_k})
    found = {k: 0 for k in ks}
    misses = []
    for case in cases:
        [embedding] = embedder.embed([case["question"]])
        keys = [f"{h.source}#{h.heading}" for h in store.search(embedding, max(ks))]
        rank = next((i for i, key in enumerate(keys, 1) if any(e in key for e in case["expect"])), None)
        for k in ks:
            found[k] += rank is not None and rank <= k
        if rank is None or rank > top_k:
            misses.append(f"{case['question']} (got {keys[0]})")
    return Report(total=len(cases), found=found, misses=misses)


def log_report(report: Report) -> None:
    for k in sorted(report.found):
        log.info("recall@%d: %d/%d = %.0f%%", k, report.found[k], report.total, 100 * report.recall(k))
    for miss in report.misses:
        log.warning("miss: %s", miss)


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(name)s: %(message)s")
    settings = Settings.from_env()
    report = evaluate(
        Store(settings.database_url),
        Embedder(settings.embedding_model, settings.embedding_cache_dir),
        settings.top_k,
    )
    log_report(report)


if __name__ == "__main__":
    main()
