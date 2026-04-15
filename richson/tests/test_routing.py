"""Tests for richson.datasources.routing.

These cover the pure routing predicates so we don't accidentally break the
catalog-vs-resolver invariant documented in
``docs/standards/richson-datasource-routing.md``.
"""

from __future__ import annotations

import pytest

from richson.datasources.routing import is_a_share_code, resolve_currency


@pytest.mark.parametrize(
    "code",
    [
        "159915",  # SZSE ChiNext ETF
        "510300",  # SSE CSI 300 ETF
        "518880",  # SSE HuaAn Gold ETF
        "159934",  # SZSE Bosera Gold ETF
        "600519",  # A-share stock (not in MVP catalog but pattern-matches)
        "000001",  # bare 6-digit numeric
    ],
)
def test_is_a_share_code_accepts_six_digit_numeric(code: str) -> None:
    assert is_a_share_code(code) is True


@pytest.mark.parametrize(
    "code",
    [
        "AAPL",  # US ticker
        "GLD",  # US ETF
        "QQQ",
        "^GSPC",  # Yahoo index symbol
        "GC=F",  # Yahoo futures
        "000001.SS",  # Yahoo Shanghai composite
        "12345",  # 5 digits
        "1234567",  # 7 digits
        "159915A",  # numeric + letter
        "",  # empty
    ],
)
def test_is_a_share_code_rejects_other_shapes(code: str) -> None:
    assert is_a_share_code(code) is False


def test_resolve_currency_a_share_returns_cny() -> None:
    assert resolve_currency("518880") == "CNY"
    assert resolve_currency("159915") == "CNY"


def test_resolve_currency_other_returns_usd() -> None:
    assert resolve_currency("AAPL") == "USD"
    assert resolve_currency("GLD") == "USD"
    assert resolve_currency("^GSPC") == "USD"


def test_resolve_currency_matches_routing_predicate() -> None:
    """The two predicates must never drift apart — currency follows routing."""
    for code in ["518880", "AAPL", "159915", "GC=F", "QQQ", "000001"]:
        expected = "CNY" if is_a_share_code(code) else "USD"
        assert resolve_currency(code) == expected, code
