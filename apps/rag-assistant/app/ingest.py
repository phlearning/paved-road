"""Indexes the Markdown documentation shipped in the image.

Runs as a Kubernetes Job after every deployment (see values.yaml): only
documents whose content changed are re-embedded, deleted ones are dropped."""

import hashlib
import logging
import sys
import time
from pathlib import Path

from .chunking import split_markdown
from .config import Settings
from .embeddings import Embedder
from .store import Store

log = logging.getLogger("ingest")


def wait_for_database(store: Store, timeout: float = 120) -> None:
    deadline = time.monotonic() + timeout
    while True:
        try:
            store.ping()
            return
        except Exception as err:
            if time.monotonic() > deadline:
                raise
            log.info("waiting for the database: %s", err)
            time.sleep(3)


def ingest(corpus: Path, store: Store, embedder: Embedder) -> dict[str, int]:
    store.ensure_schema(embedder.dimensions)
    known = store.digests()
    seen: set[str] = set()
    stats = {"indexed": 0, "unchanged": 0, "deleted": 0, "chunks": 0}

    for path in sorted(corpus.rglob("*.md")):
        source = path.relative_to(corpus).as_posix()
        text = path.read_text(encoding="utf-8")
        digest = hashlib.sha256(text.encode()).hexdigest()
        seen.add(source)
        if known.get(source) == digest:
            stats["unchanged"] += 1
            continue
        chunks = split_markdown(source, text)
        if not chunks:
            continue
        store.replace(source, digest, chunks, embedder.embed([f"{c.heading}\n{c.text}" for c in chunks]))
        stats["indexed"] += 1
        stats["chunks"] += len(chunks)
        log.info("indexed %s (%d chunks)", source, len(chunks))

    removed = sorted(set(known) - seen)
    store.delete(removed)
    stats["deleted"] = len(removed)
    return stats


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(name)s: %(message)s")
    settings = Settings.from_env()
    if not settings.database_url:
        sys.exit("DATABASE_URL is not set")
    corpus = Path(sys.argv[1] if len(sys.argv) > 1 else settings.corpus_dir)
    store = Store(settings.database_url)
    wait_for_database(store)
    stats = ingest(corpus, store, Embedder(settings.embedding_model, settings.embedding_cache_dir))
    log.info("done: %s", stats)


if __name__ == "__main__":
    main()
