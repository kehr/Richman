"""Prompt templates for ADK agents."""

from .execution import EXECUTION_PROMPT
from .interpretation import INTERPRETATION_PROMPT
from .research import D1_RESEARCH_PROMPT, D2_RESEARCH_PROMPT, D3_RESEARCH_PROMPT

__all__ = [
    "D1_RESEARCH_PROMPT",
    "D2_RESEARCH_PROMPT",
    "D3_RESEARCH_PROMPT",
    "INTERPRETATION_PROMPT",
    "EXECUTION_PROMPT",
]
