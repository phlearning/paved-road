"""Chunks and their embeddings in PostgreSQL with pgvector."""

from dataclasses import dataclass

import psycopg
from pgvector.psycopg import register_vector

from .chunking import Chunk


@dataclass(frozen=True)
class Hit:
    source: str
    heading: str
    text: str
    score: float


class Store:
    def __init__(self, database_url: str):
        self._url = database_url

    def _connect(self) -> psycopg.Connection:
        conn = psycopg.connect(self._url, connect_timeout=5)
        register_vector(conn)
        return conn

    def ping(self) -> None:
        with psycopg.connect(self._url, connect_timeout=2) as conn:
            conn.execute("SELECT 1")

    def ensure_schema(self, dimensions: int) -> None:
        # The vector extension itself is created by the platform
        # (postgres.extensions in values.yaml): the app role cannot.
        with self._connect() as conn:
            conn.execute(
                f"""
                CREATE TABLE IF NOT EXISTS documents (
                    source text PRIMARY KEY,
                    digest text NOT NULL,
                    indexed_at timestamptz NOT NULL DEFAULT now()
                );
                CREATE TABLE IF NOT EXISTS chunks (
                    id bigserial PRIMARY KEY,
                    source text NOT NULL REFERENCES documents (source) ON DELETE CASCADE,
                    heading text NOT NULL,
                    content text NOT NULL,
                    embedding vector({int(dimensions)}) NOT NULL
                );
                CREATE INDEX IF NOT EXISTS chunks_embedding_idx
                    ON chunks USING hnsw (embedding vector_cosine_ops);
                """
            )

    def digests(self) -> dict[str, str]:
        with self._connect() as conn:
            return dict(conn.execute("SELECT source, digest FROM documents").fetchall())

    def replace(self, source: str, digest: str, chunks: list[Chunk], embeddings: list[list[float]]) -> None:
        """Swaps a document's chunks in one transaction: readers never see a
        half-indexed document."""
        with self._connect() as conn, conn.transaction():
            conn.execute("DELETE FROM documents WHERE source = %s", (source,))
            conn.execute("INSERT INTO documents (source, digest) VALUES (%s, %s)", (source, digest))
            with conn.cursor() as cur:
                cur.executemany(
                    "INSERT INTO chunks (source, heading, content, embedding) VALUES (%s, %s, %s, %s)",
                    [(c.source, c.heading, c.text, e) for c, e in zip(chunks, embeddings, strict=True)],
                )

    def delete(self, sources: list[str]) -> None:
        if sources:
            with self._connect() as conn:
                conn.execute("DELETE FROM documents WHERE source = ANY(%s)", (sources,))

    def search(self, embedding: list[float], limit: int) -> list[Hit]:
        with self._connect() as conn:
            rows = conn.execute(
                """
                SELECT source, heading, content, 1 - (embedding <=> %s::vector) AS score
                FROM chunks ORDER BY embedding <=> %s::vector LIMIT %s
                """,
                (embedding, embedding, limit),
            ).fetchall()
        return [Hit(source=r[0], heading=r[1], text=r[2], score=float(r[3])) for r in rows]
