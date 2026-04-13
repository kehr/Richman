"""Degradation templates for Layer 3 text generation without LLM.

When the interpretation agent fails or is unavailable, these templates generate
structured analysis text from quantitative scores only, without any LLM calls.
The generated text is clearly structured but less nuanced than LLM output.

Usage:
    from richson.templates import render_interpretation_zh, render_interpretation_en
"""

from richson.templates.interpretation_en import render_interpretation_en
from richson.templates.interpretation_zh import render_interpretation_zh

__all__ = [
    "render_interpretation_zh",
    "render_interpretation_en",
]
