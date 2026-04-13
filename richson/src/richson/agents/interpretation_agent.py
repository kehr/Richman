"""Layer 3 interpretation agent: generates human-readable analysis text.

Receives final dimension scores plus Layer 2 research summaries and produces
a structured Chinese/English interpretation for the asset analysis.
"""

from __future__ import annotations

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Output schema
# ---------------------------------------------------------------------------


class InterpretationResult(BaseModel):
    """Structured interpretation output from the interpretation agent."""

    market_interpretation: str = Field(
        description=(
            "Concise market interpretation, 100-200 characters. "
            "Lead with conclusion, then key drivers. Professional and opinionated."
        )
    )
    risk_factors: list[str] = Field(
        description=(
            "2-3 specific risk factors that could invalidate the current stance. "
            "Each item 30-50 characters."
        ),
        min_length=2,
        max_length=3,
    )
    regime_summary: str = Field(
        description="One sentence describing the current macro regime from gold's perspective."
    )
    major_change_recap: str | None = Field(
        default=None,
        description=(
            "Only populated when |score_delta| > 20. Explains what changed: "
            "previous thesis, invalidated assumption, and recommended adjustment. "
            "2-3 sentences."
        ),
    )
    change_summary: str | None = Field(
        default=None,
        description=(
            "Compact change summary, format: 'D{n}{+/-delta}(reason)' per dimension "
            "where |delta| >= 3, ordered by absolute delta descending. "
            "Null for first analysis."
        ),
    )


# ---------------------------------------------------------------------------
# Agent factory
# ---------------------------------------------------------------------------


def create_interpretation_agent() -> object:
    """Create the Layer 3 interpretation agent.

    Returns an ADK Agent instance. Model is injected at runtime by the factory.

    Returns:
        An ADK Agent instance configured for text generation (no search tools).
    """
    from google.adk.agents import Agent

    from richson.agents.prompts.interpretation import INTERPRETATION_PROMPT

    # Interpretation agent has no tools — it synthesizes from provided context only.
    return Agent(
        name="interpretation_agent",
        model="",  # model injected at runtime by create_agent factory
        instruction=INTERPRETATION_PROMPT,
        tools=[],
        output_schema=InterpretationResult,
    )
