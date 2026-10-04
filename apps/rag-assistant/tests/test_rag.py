from app.llm import Completion
from app.rag import Assistant, build_prompt
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
    prompt = build_prompt("How?", HITS)
    assert '<excerpt ref="1" source="docs/guides/create-a-service.md"' in prompt
    assert '<excerpt ref="2" source="README.md"' in prompt
    assert prompt.endswith("Question: How?")
