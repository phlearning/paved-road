from fastapi.testclient import TestClient

from app.main import app, get_assistant
from app.rag import Answer, Source

client = TestClient(app)


class FakeAssistant:
    def ask(self, question: str) -> Answer:
        return Answer(answer=f"echo: {question}", sources=[Source(1, "docs/a.md", "A", 0.9)], model="fake")


def test_probes():
    assert client.get("/healthz").status_code == 200
    # No DATABASE_URL in tests: ready without a database check.
    assert client.get("/readyz").status_code == 200


def test_ask_returns_answer_and_sources():
    app.dependency_overrides[get_assistant] = FakeAssistant
    try:
        response = client.post("/ask", json={"question": "How do I create a service?"})
    finally:
        app.dependency_overrides.clear()
    assert response.status_code == 200
    body = response.json()
    assert body["answer"] == "echo: How do I create a service?"
    assert body["sources"][0]["source"] == "docs/a.md"


def test_ask_validates_the_question():
    app.dependency_overrides[get_assistant] = FakeAssistant
    try:
        assert client.post("/ask", json={"question": ""}).status_code == 422
    finally:
        app.dependency_overrides.clear()


def test_metrics_track_routes_not_probes():
    client.get("/")
    client.get("/healthz")
    body = client.get("/metrics").text
    assert 'http_requests_total{handler="/",method="GET",status="2xx"}' in body
    assert 'handler="/healthz"' not in body
