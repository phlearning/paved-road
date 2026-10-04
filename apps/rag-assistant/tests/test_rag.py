from app.llm import Completion
from app.rag import Assistant, build_prompt, language
from app.store import Hit


class FakeEmbedder:
    def embed(self, texts):
        return [[1.0, 0.0] for _ in texts]


class FakeStore:
    def __init__(self, hits):
        self.hits = hits

    def search(self, embedding, limit):
        return self.hits[:limit]


class FakeLLM:
    def __init__(self):
        self.prompts = []

    def complete(self, system, prompt):
        self.prompts.append(prompt)
        return Completion(text="Use platformctl new-service [1].", model="fake", input_tokens=10, output_tokens=5)


HITS = [
    Hit("docs/guides/create-a-service.md", "Create a service", "Run platformctl new-service NAME.", 0.8),
    Hit("README.md", "paved-road", "A local platform.", 0.1),
]


def test_answer_cites_relevant_excerpts_only():
    llm = FakeLLM()
    answer = Assistant(FakeStore(HITS), FakeEmbedder(), llm).ask("Comment créer un service ?")

    assert answer.answer == "Use platformctl new-service [1]."
    # The low-score excerpt is dropped before reaching the model.
    assert [s.source for s in answer.sources] == ["docs/guides/create-a-service.md"]
    assert "A local platform." not in llm.prompts[0]


def test_no_relevant_context_skips_the_model():
    llm = FakeLLM()
    answer = Assistant(FakeStore(HITS[1:]), FakeEmbedder(), llm).ask("What is the weather?")
    assert answer.sources == []
    assert llm.prompts == []


def test_prompt_numbers_excerpts_and_keeps_the_question():
    prompt = build_prompt("How do I create a service?", HITS)
    assert '<excerpt ref="1" source="docs/guides/create-a-service.md"' in prompt
    assert '<excerpt ref="2" source="README.md"' in prompt
    assert "Question: How do I create a service?" in prompt
    assert prompt.endswith("Answer in English, citing the excerpts you use as [1], [2].")


def test_prompt_closes_in_the_language_of_the_question():
    assert build_prompt("Comment créer un service ?", HITS).endswith("Réponds en français, en citant les extraits utilisés comme [1], [2].")
    assert build_prompt("Comment ajouter une base pour mon service ?", HITS).startswith("<documentation>")
    assert language("Comment ajouter une base pour mon service ?") == "fr"
    assert language("Where are the logs of my service?") == "en"
    assert language("Postgres?") == "en"


def test_evaluation_accepts_any_listed_section(tmp_path, monkeypatch):
    import json

    from app import evaluate as ev

    questions = tmp_path / "q.json"
    questions.write_text(json.dumps([
        {"question": "create?", "expect": ["guide.md#Create", "README.md#Golden path"]},
        {"question": "weather?", "expect": ["nowhere.md"]},
    ]))
    monkeypatch.setattr(ev, "QUESTIONS", questions)
    store = FakeStore([Hit("README.md", "Golden path", "x", 0.9), Hit("other.md", "Other", "y", 0.5)])

    report = ev.evaluate(store, FakeEmbedder(), top_k=5)
    assert report.found == {1: 1, 3: 1, 5: 1}
    assert len(report.misses) == 1 and report.misses[0].startswith("weather?")
