"""Measures retrieval quality on a fixed set of questions.

Each question names the document section that should be retrieved. The score
is recall@k: the share of questions whose expected section is among the top
k excerpts. Run it in the cluster after an ingestion:

    kubectl -n rag-assistant exec deploy/rag-assistant -- python -m app.evaluate
"""

import json
import sys
from pathlib import Path

from .config import Settings
from .embeddings import Embedder
from .store import Store

QUESTIONS = Path(__file__).with_name("eval_questions.json")


def main() -> None:
    settings = Settings.from_env()
    store = Store(settings.database_url)
    embedder = Embedder(settings.embedding_model, settings.embedding_cache_dir)
    cases = json.loads(QUESTIONS.read_text())
    ks = (1, 3, settings.top_k)

    found = {k: 0 for k in ks}
    for case in cases:
        [embedding] = embedder.embed([case["question"]])
        hits = store.search(embedding, max(ks))
        keys = [f"{h.source}#{h.heading}" for h in hits]
        rank = next((i for i, key in enumerate(keys, 1) if case["expect"] in key), None)
        for k in ks:
            found[k] += rank is not None and rank <= k
        mark = f"rank {rank}" if rank else "MISS"
        print(f"{mark:>7}  {case['question']}")
        if not rank or rank > settings.top_k:
            print(f"         expected {case['expect']}, got {keys[0]}")

    print()
    for k in ks:
        print(f"recall@{k}: {found[k]}/{len(cases)} = {found[k] / len(cases):.0%}")
    sys.exit(0 if found[settings.top_k] == len(cases) else 1)


if __name__ == "__main__":
    main()
