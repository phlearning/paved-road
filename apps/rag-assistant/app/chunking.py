"""Splits Markdown documents into retrieval chunks that follow their headings."""

import re
from dataclasses import dataclass

HEADING = re.compile(r"^(#{1,4})\s+(.*\S)\s*$")
FENCE = re.compile(r"^\s*(```|~~~)")


@dataclass(frozen=True)
class Chunk:
    source: str
    heading: str
    text: str


def split_markdown(source: str, markdown: str, max_chars: int = 1200) -> list[Chunk]:
    """Cuts at headings first, then at paragraph boundaries when a section is
    longer than max_chars. Code blocks are never cut in the middle. Each chunk
    keeps its heading path ("ADR 3 > Decision") so retrieved text has context."""
    chunks: list[Chunk] = []
    path: list[tuple[int, str]] = []
    paragraphs: list[str] = []

    def flush() -> None:
        heading = " > ".join(title for _, title in path) or source
        for text in _pack(paragraphs, max_chars):
            chunks.append(Chunk(source=source, heading=heading, text=text))
        paragraphs.clear()

    for block in _blocks(markdown):
        match = HEADING.match(block)
        if match and "\n" not in block:
            flush()
            level = len(match.group(1))
            path[:] = [(lvl, title) for lvl, title in path if lvl < level]
            path.append((level, match.group(2)))
        else:
            paragraphs.append(block)
    flush()
    return chunks


def _blocks(markdown: str) -> list[str]:
    """Paragraphs separated by blank lines, with fenced code kept whole and
    headings always on their own."""
    blocks: list[str] = []
    current: list[str] = []
    in_fence = False

    def close() -> None:
        if current:
            blocks.append("\n".join(current).strip())
            current.clear()

    for line in markdown.splitlines():
        if FENCE.match(line):
            in_fence = not in_fence
            current.append(line)
            continue
        if in_fence:
            current.append(line)
        elif HEADING.match(line):
            close()
            blocks.append(line.strip())
        elif not line.strip():
            close()
        else:
            current.append(line)
    close()
    return [b for b in blocks if b]


def _pack(paragraphs: list[str], max_chars: int) -> list[str]:
    packed: list[str] = []
    current = ""
    for paragraph in paragraphs:
        if current and len(current) + len(paragraph) + 2 > max_chars:
            packed.append(current)
            current = paragraph
        else:
            current = f"{current}\n\n{paragraph}" if current else paragraph
    if current:
        packed.append(current)
    return packed
