"""ADK agent layer for richson: research, interpretation, and execution agents.

This module provides the factory functions and agent classes for Layer 2 and Layer 3
of the three-tier analysis pipeline.

Layer 2 (research): research_agent runs once per dimension (D1, D2, D3).
Layer 3 (interpretation + execution): interpretation_agent and execution_agent.

All agents are created via the create_agent() factory which handles LLM provider
resolution and model injection via LiteLlm.
"""

from __future__ import annotations

import json
import logging
import uuid
from typing import Any

from pydantic import BaseModel

from richson.schemas.common import LLMConfig

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# LLM provider -> LiteLlm model string resolution
# ---------------------------------------------------------------------------

_PROVIDER_PREFIXES: dict[str, str] = {
    "claude": "anthropic",
    "openai": "openai",
    "gemini": "",  # gemini does not go through LiteLlm
}


def _resolve_model(llm_config: LLMConfig) -> Any:
    """Resolve LLMConfig to an ADK-compatible model object.

    For Gemini models, returns a plain model name string (ADK handles natively).
    For all other providers, returns a LiteLlm wrapper instance.

    Args:
        llm_config: Provider and model configuration from the incoming request.

    Returns:
        A model string (Gemini) or LiteLlm instance (Claude, OpenAI, etc.).
    """
    from google.adk.models.lite_llm import LiteLlm

    provider = llm_config.provider.lower()

    if provider == "gemini":
        # ADK handles Gemini natively — return plain model name
        return llm_config.model

    if provider == "claude":
        model_str = f"anthropic/{llm_config.model}"
    elif provider == "openai" or provider == "openai_compatible":
        model_str = f"openai/{llm_config.model}"
    else:
        # Fallback: pass through as-is; LiteLlm will attempt to resolve
        model_str = llm_config.model
        logger.warning("Unknown provider '%s'; passing model string as-is", provider)

    kwargs: dict[str, Any] = {}
    if llm_config.api_key:
        kwargs["api_key"] = llm_config.api_key
    if llm_config.api_base and provider == "openai_compatible":
        kwargs["api_base"] = llm_config.api_base

    return LiteLlm(model=model_str, **kwargs)


# ---------------------------------------------------------------------------
# Agent factory
# ---------------------------------------------------------------------------


def create_agent(
    name: str,
    llm_config: LLMConfig,
    instruction: str,
    tools: list[Any],
    output_schema: type[BaseModel] | None = None,
) -> Any:
    """Create an ADK Agent with the resolved LLM configuration.

    This factory is the single point of model resolution for all richson agents.

    Args:
        name: Agent name (must be unique within a runner).
        llm_config: LLM provider and model configuration.
        instruction: System instruction for the agent.
        tools: List of ADK-compatible tools.
        output_schema: Optional Pydantic model class for structured output.

    Returns:
        An ADK Agent (LlmAgent) instance ready for use with InMemoryRunner.
    """
    from google.adk.agents import Agent

    model = _resolve_model(llm_config)
    return Agent(
        name=name,
        model=model,
        instruction=instruction,
        tools=tools,
        output_schema=output_schema,
    )


# ---------------------------------------------------------------------------
# Agent runner
# ---------------------------------------------------------------------------


async def run_agent(
    agent: Any,
    user_input: str,
    *,
    app_name: str = "richson",
    timeout_seconds: float = 60.0,
) -> dict[str, Any]:
    """Execute an ADK agent via InMemoryRunner and return the structured output.

    Creates a fresh session per invocation (stateless execution). Collects all
    events and extracts the final response from the last event with is_final_response().

    Args:
        agent: An ADK Agent instance.
        user_input: The user message to send to the agent.
        app_name: Application name for the runner (used for session namespacing).
        timeout_seconds: Maximum seconds to wait for agent completion.

    Returns:
        Parsed output dict from the agent's final response.

    Raises:
        RuntimeError: If the agent produces no final response within the timeout.
        ValueError: If the agent response cannot be parsed as structured output.
    """
    import asyncio

    from google.adk.runners import InMemoryRunner
    from google.genai import types

    runner = InMemoryRunner(agent=agent, app_name=app_name)

    user_id = "system"
    session_id = str(uuid.uuid4())

    # Create session explicitly (auto_create_session not set on InMemoryRunner)
    await runner.session_service.create_session(
        app_name=app_name,
        user_id=user_id,
        session_id=session_id,
    )

    new_message = types.Content(
        role="user",
        parts=[types.Part.from_text(text=user_input)],
    )

    final_event = None
    async def _collect_events() -> None:
        nonlocal final_event
        async for event in runner.run_async(
            user_id=user_id,
            session_id=session_id,
            new_message=new_message,
        ):
            if event.is_final_response():
                final_event = event

    try:
        await asyncio.wait_for(_collect_events(), timeout=timeout_seconds)
    except TimeoutError:
        logger.error(
            "Agent '%s' timed out after %.0fs",
            getattr(agent, "name", "unknown"),
            timeout_seconds,
        )
        raise RuntimeError(
            f"Agent '{getattr(agent, 'name', 'unknown')}' timed out after {timeout_seconds}s"
        ) from None

    if final_event is None:
        raise RuntimeError(
            f"Agent '{getattr(agent, 'name', 'unknown')}' produced no final response"
        )

    # Extract text content from the final event
    content = final_event.content
    if content is None or not content.parts:
        raise ValueError("Agent final response has no content parts")

    # Collect text from all parts
    text_parts = [p.text for p in content.parts if p.text]
    if not text_parts:
        raise ValueError("Agent final response contains no text output")

    raw_text = "".join(text_parts).strip()

    # Parse as JSON (ADK structured output wraps the schema in JSON)
    try:
        return json.loads(raw_text)
    except json.JSONDecodeError:
        # Some models may return markdown code blocks
        if "```json" in raw_text:
            start = raw_text.index("```json") + 7
            end = raw_text.rindex("```")
            raw_text = raw_text[start:end].strip()
        elif "```" in raw_text:
            start = raw_text.index("```") + 3
            end = raw_text.rindex("```")
            raw_text = raw_text[start:end].strip()
        return json.loads(raw_text)


# ---------------------------------------------------------------------------
# Convenience re-exports
# ---------------------------------------------------------------------------

from richson.agents.execution_agent import ExecutionPlanOutput, ScenarioOutput  # noqa: E402
from richson.agents.interpretation_agent import InterpretationResult  # noqa: E402
from richson.agents.research_agent import (  # noqa: E402
    QualitativeJudgment,
    ResearchEvent,
    ResearchResult,
)

__all__ = [
    # Factory
    "create_agent",
    "run_agent",
    # Research agent schemas
    "ResearchResult",
    "ResearchEvent",
    "QualitativeJudgment",
    # Interpretation agent schemas
    "InterpretationResult",
    # Execution agent schemas
    "ExecutionPlanOutput",
    "ScenarioOutput",
]
