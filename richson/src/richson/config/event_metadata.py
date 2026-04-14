"""Static metadata for FRED economic releases.

Each entry maps a FRED release_id to display metadata used by the event radar
endpoint (api/events.py). Adding a release here automatically opts it into
the event radar - there is no separate registration.

release_id values are verified against https://fred.stlouisfed.org/release?rid=N
The verification URL is recorded in each entry's comment.

impact: "high" | "medium" | "low" - drives UI tag color
gold_direction: "bullish" | "bearish" | "neutral" | None
zh_title / en_title: human-readable display names per locale
category: free-form string for future grouping (matches existing values)
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

EventImpact = Literal["high", "medium", "low"]
GoldDirection = Literal["bullish", "bearish", "neutral"]


@dataclass(frozen=True)
class ReleaseMeta:
    impact: EventImpact
    gold_direction: GoldDirection | None
    zh_title: str
    en_title: str
    category: str


# release_id -> ReleaseMeta
# All ids verified at https://fred.stlouisfed.org/release?rid=N as of 2026-04-15.
FRED_RELEASE_METADATA: dict[int, ReleaseMeta] = {
    10: ReleaseMeta(  # Consumer Price Index
        impact="high",
        gold_direction="bullish",
        zh_title="美国 CPI 数据公布",
        en_title="US CPI Data Release",
        category="inflation",
    ),
    46: ReleaseMeta(  # Producer Price Index
        impact="medium",
        gold_direction="neutral",
        zh_title="美国 PPI 数据公布",
        en_title="US PPI Data Release",
        category="inflation",
    ),
    50: ReleaseMeta(  # Employment Situation (Non-Farm Payrolls)
        impact="high",
        gold_direction="neutral",
        zh_title="美国非农就业数据",
        en_title="US Non-Farm Payrolls",
        category="employment",
    ),
    54: ReleaseMeta(  # Personal Income & Outlays (PCE)
        impact="high",
        gold_direction="bullish",
        zh_title="美国 PCE 通胀数据",
        en_title="US PCE Inflation Data",
        category="inflation",
    ),
    53: ReleaseMeta(  # Gross Domestic Product
        impact="high",
        gold_direction="neutral",
        zh_title="美国 GDP 数据",
        en_title="US GDP Data",
        category="growth",
    ),
    101: ReleaseMeta(  # FOMC Press Release
        impact="high",
        gold_direction="bullish",
        zh_title="FOMC 利率决议与新闻发布",
        en_title="FOMC Press Release",
        category="monetary_policy",
    ),
    13: ReleaseMeta(  # Industrial Production & Capacity Utilization (G.17)
        impact="medium",
        gold_direction="neutral",
        zh_title="美国工业产出数据",
        en_title="US Industrial Production",
        category="growth",
    ),
    9: ReleaseMeta(  # Advance Monthly Sales for Retail & Food Services
        impact="medium",
        gold_direction="neutral",
        zh_title="美国零售销售数据",
        en_title="US Retail Sales",
        category="growth",
    ),
    291: ReleaseMeta(  # Existing Home Sales
        impact="low",
        gold_direction="neutral",
        zh_title="美国成屋销售数据",
        en_title="US Existing Home Sales",
        category="growth",
    ),
}


def fred_release_url(release_id: int) -> str:
    """Build the canonical FRED release page URL.

    Verified URL pattern: https://fred.stlouisfed.org/release?rid={id}
    """
    return f"https://fred.stlouisfed.org/release?rid={release_id}"


def polymarket_event_url(slug: str) -> str:
    """Build the Polymarket event page URL from a market slug."""
    return f"https://polymarket.com/event/{slug}"
