from app.chunking import split_markdown

DOC = """# Secrets

Intro paragraph.

## Sealing

Use `platformctl seal`.

```bash
platformctl seal api KEY

# not a heading inside code
```

## Rotation

Rotate yearly.
"""


def test_chunks_follow_heading_path():
    chunks = split_markdown("docs/secrets.md", DOC)
    assert [c.heading for c in chunks] == ["Secrets", "Secrets > Sealing", "Secrets > Rotation"]
    assert chunks[0].text == "Intro paragraph."


def test_code_blocks_stay_whole():
    sealing = split_markdown("docs/secrets.md", DOC)[1]
    assert "# not a heading inside code" in sealing.text
    assert sealing.text.count("```") == 2


def test_long_sections_are_split_at_paragraphs():
    paragraphs = "\n\n".join(f"Paragraph {i} " + "x" * 300 for i in range(6))
    chunks = split_markdown("doc.md", f"# Long\n\n{paragraphs}", max_chars=700)
    assert len(chunks) == 3
    assert all(len(c.text) <= 700 for c in chunks)
    assert all(c.heading == "Long" for c in chunks)


def test_document_without_headings_uses_its_source():
    [chunk] = split_markdown("README.md", "Just text.")
    assert chunk.heading == "README.md"
