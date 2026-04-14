"""Layer 2 research agent: information retrieval and qualitative judgment per dimension.

This agent is called once per dimension (D1, D2, D3). D4 is purely quantitative
and does not use LLM research.

The agent uses Google Search and web page loading tools to retrieve current
market information, then produces a structured qualitative judgment that feeds
into the numeric adjustment calculation.
"""

from __future__ import annotations

from typing import Literal

import structlog
from pydantic import BaseModel, Field, field_validator

logger = structlog.get_logger(__name__)

# ---------------------------------------------------------------------------
# Output schemas
# ---------------------------------------------------------------------------


class ResearchEvent(BaseModel):
    """A single researched event with source attribution."""

    source_url: str = Field(description="URL of the source article or report")
    source_name: str = Field(description="Name of the publication or source")
    date: str = Field(description="Publication date, ISO format preferred")
    summary: str = Field(description="Brief factual summary of the event (1-2 sentences)")


class QualitativeJudgment(BaseModel):
    """Structured qualitative judgment output from the research agent."""

    direction: Literal["bullish", "bearish", "neutral"] = Field(
        description="Overall directional judgment for gold"
    )
    magnitude: Literal["major", "moderate", "minor"] = Field(
        description="Strength of the signal; 'major' requires >= 2 independent sources"
    )
    confidence: Literal["high", "medium", "low"] = Field(
        description="Confidence level based on data quality and source agreement"
    )
    rationale: str = Field(
        description="1-2 sentence explanation of the judgment, citing key evidence"
    )


class ResearchResult(BaseModel):
    """Full output of the research agent for one dimension."""

    dimension: Literal["D1", "D2", "D3"] = Field(
        description="The dimension being assessed"
    )
    events: list[ResearchEvent] = Field(
        description="List of relevant events found; only include events with source_url"
    )
    judgment: QualitativeJudgment = Field(
        description="Structured qualitative judgment for this dimension"
    )

    @field_validator("events")
    @classmethod
    def filter_events_without_url(cls, events: list[ResearchEvent]) -> list[ResearchEvent]:
        """Remove any events that lack a verifiable source URL."""
        valid = [e for e in events if e.source_url and e.source_url.startswith("http")]
        if len(valid) < len(events):
            logger.warning(
                "research_events_filtered_missing_url",
                filtered_count=len(events) - len(valid),
            )
        return valid

    def validate_major_magnitude(self) -> ResearchResult:
        """Downgrade 'major' to 'moderate' if fewer than 2 independent sources."""
        if self.judgment.magnitude == "major" and len(self.events) < 2:
            logger.warning(
                "magnitude_downgraded_insufficient_sources",
                dimension=self.dimension,
                source_count=len(self.events),
                minimum_required=2,
            )
            self.judgment.magnitude = "moderate"  # type: ignore[assignment]
        return self


# ---------------------------------------------------------------------------
# Agent factory (deferred import to avoid circular deps)
# ---------------------------------------------------------------------------


def create_research_agent(dimension: Literal["D1", "D2", "D3"]) -> object:
    """Create a research agent for the given dimension.

    Returns an ADK Agent instance configured with Google Search and web page tools.
    The agent is created without LLM config (which is injected at runtime by the factory).

    Args:
        dimension: The dimension identifier to configure the agent for.

    Returns:
        An ADK Agent instance ready to use with a runner.
    """
    # Deferred import: ADK + tools may not be installed in test environments
    from google.adk.agents import Agent
    from google.adk.tools.google_search_tool import google_search
    from google.adk.tools.load_web_page import load_web_page

    from richson.agents.prompts.research import (
        D1_RESEARCH_PROMPT,
        D2_RESEARCH_PROMPT,
        D3_RESEARCH_PROMPT,
    )

    prompt_map = {
        "D1": D1_RESEARCH_PROMPT,
        "D2": D2_RESEARCH_PROMPT,
        "D3": D3_RESEARCH_PROMPT,
    }
    instruction_template = prompt_map[dimension]

    # Note: the instruction template contains {quant_score} and {quant_context}
    # placeholders that are filled at call time (not at agent creation time).
    # The agent is created with a base instruction; the pipeline fills placeholders
    # in the user message content passed to run_async.
    return Agent(
        name=f"research_agent_{dimension.lower()}",
        model="",  # model is set by create_agent factory at call time
        instruction=instruction_template,
        tools=[google_search, load_web_page],
        output_schema=ResearchResult,
    )
