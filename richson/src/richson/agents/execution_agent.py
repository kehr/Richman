"""Layer 3 execution agent: generates conditional execution plan with scenarios.

Receives asset scores, user holding context, and risk parameters, then produces
a structured execution plan with stop-loss, take-profit, and conditional scenarios.

Design decision: holding data (cost price, position ratio) is included in the LLM
request per TRD SS8.4 — this data is only used in-memory for this request and is
never persisted by richson.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

# ---------------------------------------------------------------------------
# Output schemas
# ---------------------------------------------------------------------------


class ScenarioOutput(BaseModel):
    """A single conditional execution scenario."""

    condition: str = Field(description="Market condition that triggers this action")
    action: str = Field(description="Specific action to take (e.g., '加仓 2%')")
    lot_count: int = Field(
        description="Number of lots to buy (positive) or sell (negative)"
    )
    rationale: str = Field(
        description="Brief rationale for this scenario, including risk notes if applicable"
    )
    priority: int = Field(
        ge=1,
        description="Execution priority; 1 = highest. Stop-loss scenario is always priority 1.",
    )
    exclusion_group: str | None = Field(
        default=None,
        description=(
            "Scenarios with the same group are mutually exclusive. "
            "Use 'long_add' for buy scenarios, 'long_reduce' for sell scenarios."
        ),
    )


class ExecutionPlanOutput(BaseModel):
    """Full execution plan output from the execution agent."""

    action: str = Field(
        description="Machine-readable action key (e.g., 'hold', 'scale_in_on_dip', 'reduce')"
    )
    action_label: str = Field(
        description="Human-readable action label in the requested language"
    )
    default_action: str = Field(
        description=(
            "What to do if no scenario triggers, in the requested language. "
            "Must include 'hold' or equivalent as the default."
        )
    )
    current_position: float = Field(
        description="Current position ratio as percentage (0-100)"
    )
    target_position: float = Field(
        description="Target position ratio after all scenarios execute (%)"
    )
    scenarios: list[ScenarioOutput] = Field(
        description="Conditional execution scenarios, ordered by significance",
        min_length=1,
    )
    stop_loss: float = Field(
        description="Stop-loss price level (ATR-based, rounded to 2 decimal places)"
    )
    take_profit: float = Field(
        description="Take-profit price level (nearest resistance or +8% fallback)"
    )
    valid_days: int = Field(
        default=7,
        ge=1,
        le=90,
        description="Number of days this execution plan is valid",
    )
    no_trigger_note: str = Field(
        description=(
            "Instructions for when no scenario triggers. Must include valid period "
            "and explicit 'maintain current position' or equivalent default."
        )
    )
    concentration_level: Literal["green", "yellow", "blue", "red"] | None = Field(
        default=None,
        description="Concentration warning level if peer exposure is high",
    )
    concentration_message: str | None = Field(
        default=None,
        description="Human-readable concentration warning message in the requested language",
    )


# ---------------------------------------------------------------------------
# Agent factory
# ---------------------------------------------------------------------------


def create_execution_agent() -> object:
    """Create the Layer 3 execution agent.

    Returns an ADK Agent instance. Model is injected at runtime by the factory.

    Returns:
        An ADK Agent instance configured for execution plan generation (no search tools).
    """
    from google.adk.agents import Agent

    from richson.agents.prompts.execution import EXECUTION_PROMPT

    # Execution agent has no search tools — it operates on provided data only.
    return Agent(
        name="execution_agent",
        model="",  # model injected at runtime by create_agent factory
        instruction=EXECUTION_PROMPT,
        tools=[],
        output_schema=ExecutionPlanOutput,
    )
