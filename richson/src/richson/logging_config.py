"""structlog JSON logging configuration (TRD SS11.1).

Configures structlog for production-ready JSON output aligned with richman's
zap format. Call configure_logging() once at application startup.
"""

from __future__ import annotations

import logging
import sys

import structlog


def configure_logging(log_level: str = "info") -> None:
    """Configure structlog for JSON output.

    Args:
        log_level: one of debug, info, warning, error (case insensitive).
    """
    level = getattr(logging, log_level.upper(), logging.INFO)

    # Configure standard library logging to capture third-party logs
    logging.basicConfig(
        format="%(message)s",
        stream=sys.stdout,
        level=level,
    )

    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.processors.TimeStamper(fmt="iso", utc=True, key="ts"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(level),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )
