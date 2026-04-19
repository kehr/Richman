"""Static metadata for FRED economic releases and FOMC meeting calendar.

This module provides two event sources used by api/events.py:

1. FRED_RELEASE_METADATA: maps FRED release_id -> ReleaseMeta for data
   releases that fire on a known cadence (CPI, NFP, PCE, GDP, etc.).
   Adding a release here automatically opts it into the event radar -
   there is no separate registration.

2. FOMC_MEETINGS: hand-maintained list of FOMC meeting dates. The FRED
   `releases/dates` API cannot be used for FOMC meetings because the
   corresponding release_ids return an entry every business day (every
   associated series update), not only on actual meeting days. The
   Federal Reserve publishes the meeting schedule at
   https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm -
   update this list once a year when the next year's schedule is posted.

release_id values are verified against https://fred.stlouisfed.org/release?rid=N
The verification URL is recorded in each entry's comment.

impact: "high" | "medium" | "low" - drives UI tag color
gold_direction: "bullish" | "bearish" | "neutral" | None
zh_title / en_title: human-readable display names per locale
category: free-form string for future grouping (matches existing values)
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date as Date
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


@dataclass(frozen=True)
class FOMCMeeting:
    """A single FOMC meeting whose press release lands on `date`.

    FOMC meetings span two days; `date` is the concluding day on which
    the policy statement is released (typically 2pm ET / 18:00 UTC).
    """

    date: Date
    en_title: str
    zh_title: str


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
    # release_id=101 ("H.15 Selected Interest Rates") was previously mapped to
    # "FOMC Press Release" here, but the FRED releases/dates API returns that
    # id on nearly every business day (H.15 updates daily), flooding the event
    # radar with duplicate "FOMC Press Release" rows. Real FOMC meetings are
    # tracked in FOMC_MEETINGS below instead.
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


# Hand-maintained FOMC meeting schedule.
# Source: https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm
# Each entry is the concluding day of a two-day meeting, on which the policy
# statement (press release) is issued. Update annually when the Fed posts the
# next year's schedule.
FOMC_MEETINGS: list[FOMCMeeting] = [
    FOMCMeeting(
        date=Date(2026, 1, 28),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 3, 18),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 4, 29),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 6, 17),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 7, 29),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 9, 16),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 10, 28),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
    FOMCMeeting(
        date=Date(2026, 12, 9),
        en_title="FOMC Meeting Decision",
        zh_title="FOMC 利率决议",
    ),
]

# Canonical FOMC calendar landing page used as sourceUrl for FOMC entries.
FOMC_CALENDAR_URL = "https://www.federalreserve.gov/monetarypolicy/fomccalendars.htm"


def fred_release_url(release_id: int) -> str:
    """Build the canonical FRED release page URL.

    Verified URL pattern: https://fred.stlouisfed.org/release?rid={id}
    """
    return f"https://fred.stlouisfed.org/release?rid={release_id}"


def polymarket_event_url(slug: str) -> str:
    """Build the Polymarket event page URL from a market slug."""
    return f"https://polymarket.com/event/{slug}"
