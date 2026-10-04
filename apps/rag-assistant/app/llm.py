"""Language model backends: the platform's Ollama server or a hosted Claude model."""

from dataclasses import dataclass
from typing import Protocol

import anthropic
import httpx2


@dataclass(frozen=True)
class Completion:
    text: str
    model: str
    input_tokens: int
    output_tokens: int


class LLMError(RuntimeError):
    """The model could not produce an answer."""


class LLM(Protocol):
    def complete(self, system: str, prompt: str) -> Completion: ...


class OllamaLLM:
    """Calls the shared model server installed by the platform (full profile)."""

    def __init__(self, base_url: str, model: str, timeout: float = 120.0):
        self._client = httpx2.Client(base_url=base_url, timeout=timeout)
        self._model = model

    def complete(self, system: str, prompt: str) -> Completion:
        try:
            response = self._client.post(
                "/api/chat",
                json={
                    "model": self._model,
                    "stream": False,
                    # Small models answer faster and stay on topic without a reasoning phase.
                    "think": False,
                    "options": {"temperature": 0.2},
                    "messages": [
                        {"role": "system", "content": system},
                        {"role": "user", "content": prompt},
                    ],
                },
            )
            response.raise_for_status()
        except httpx2.HTTPError as err:
            raise LLMError(f"Ollama request failed: {err}") from err
        body = response.json()
        return Completion(
            text=body["message"]["content"].strip(),
            model=self._model,
            input_tokens=body.get("prompt_eval_count", 0),
            output_tokens=body.get("eval_count", 0),
        )


class AnthropicLLM:
    """Hosted Claude model, used when the platform runs no local model (lite
    profile). The API key comes from ANTHROPIC_API_KEY, sealed in values.yaml."""

    def __init__(self, model: str, effort: str):
        self._client = anthropic.Anthropic()
        self._model = model
        self._effort = effort

    def complete(self, system: str, prompt: str) -> Completion:
        try:
            response = self._client.beta.messages.create(
                model=self._model,
                max_tokens=4096,
                # Documentation Q&A does not need deep reasoning: low effort
                # keeps answers fast and cheap.
                output_config={"effort": self._effort},
                # If a safety classifier declines, the API re-runs the request
                # on its recommended fallback model instead of failing.
                betas=["server-side-fallback-2026-07-01"],
                fallbacks="default",
                system=system,
                messages=[{"role": "user", "content": prompt}],
            )
        except anthropic.RateLimitError as err:
            raise LLMError("Claude API rate limit reached, retry shortly") from err
        except anthropic.APIStatusError as err:
            raise LLMError(f"Claude API error {err.status_code}: {err.message}") from err
        except anthropic.APIConnectionError as err:
            raise LLMError(f"Claude API unreachable: {err}") from err

        if response.stop_reason == "refusal":
            raise LLMError("The model declined to answer this question")
        text = "".join(block.text for block in response.content if block.type == "text")
        return Completion(
            text=text.strip(),
            model=response.model,
            input_tokens=response.usage.input_tokens,
            output_tokens=response.usage.output_tokens,
        )


def from_settings(settings) -> LLM:
    if settings.llm_provider == "anthropic":
        return AnthropicLLM(settings.anthropic_model, settings.anthropic_effort)
    if settings.llm_provider == "ollama":
        return OllamaLLM(settings.ollama_url, settings.ollama_model)
    raise ValueError(f"unknown LLM_PROVIDER {settings.llm_provider!r}: use 'ollama' or 'anthropic'")
