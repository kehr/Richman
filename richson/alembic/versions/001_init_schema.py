"""Init richson schema: rs_* tables and seed data.

Revision ID: 001
Revises:
Create Date: 2026-04-14
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision = "001"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    # ------------------------------------------------------------------
    # rs_asset_analyses
    # ------------------------------------------------------------------
    op.create_table(
        "rs_asset_analyses",
        sa.Column("asset_analysis_id", sa.BigInteger(), nullable=False),
        sa.Column("asset_code", sa.String(32), nullable=False),
        sa.Column("locale", sa.String(8), nullable=False, server_default="zh"),
        # overall
        sa.Column("overall_score", sa.Numeric(5, 2), nullable=False),
        sa.Column("signal_level", sa.String(32), nullable=False),
        sa.Column("confidence", sa.Numeric(5, 2), nullable=False),
        sa.Column("confidence_band_low", sa.Numeric(5, 2), nullable=False),
        sa.Column("confidence_band_high", sa.Numeric(5, 2), nullable=False),
        sa.Column("model_version", sa.String(32), nullable=False),
        # Layer 3 text output
        sa.Column("market_interpretation", sa.Text(), nullable=False, server_default=""),
        sa.Column(
            "risk_factors",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default="[]",
        ),
        sa.Column("regime_summary", sa.Text(), nullable=False, server_default=""),
        # dimension scores (denormalized)
        sa.Column("d1_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d1_base_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d1_llm_adjustment", sa.Numeric(5, 2), nullable=True, server_default="0"),
        sa.Column("d2_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d2_base_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d2_llm_adjustment", sa.Numeric(5, 2), nullable=True, server_default="0"),
        sa.Column("d3_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d3_base_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d3_llm_adjustment", sa.Numeric(5, 2), nullable=True, server_default="0"),
        sa.Column("d4_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("d4_base_score", sa.Numeric(5, 2), nullable=True),
        # weights snapshot at analysis time
        sa.Column("d1_weight", sa.Numeric(4, 2), nullable=False, server_default="0.30"),
        sa.Column("d2_weight", sa.Numeric(4, 2), nullable=False, server_default="0.25"),
        sa.Column("d3_weight", sa.Numeric(4, 2), nullable=False, server_default="0.25"),
        sa.Column("d4_weight", sa.Numeric(4, 2), nullable=False, server_default="0.20"),
        # degradation
        sa.Column("llm_skipped", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("data_coverage", sa.String(16), nullable=False, server_default="full"),
        # conflict
        sa.Column("conflict_type", sa.String(16), nullable=True),
        sa.Column("conflict_message", sa.Text(), nullable=True),
        # change tracking
        sa.Column("prev_analysis_id", sa.BigInteger(), nullable=True),
        sa.Column("score_delta", sa.Numeric(5, 2), nullable=True),
        sa.Column("change_summary", sa.Text(), nullable=True),
        sa.Column("major_change_recap", sa.Text(), nullable=True),
        # currency
        sa.Column("usd_exchange_rate", sa.Numeric(12, 6), nullable=True),
        # data freshness
        sa.Column("data_snapshot_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column("price_at_analysis", sa.Numeric(20, 6), nullable=True),
        # pre-computed demo plan
        sa.Column(
            "demo_plan",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=True,
        ),
        # extensible metadata
        sa.Column(
            "analysis_metadata",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default="{}",
        ),
        # generation mode
        sa.Column("generated_by", sa.String(16), nullable=False, server_default="full"),
        # audit
        sa.Column("source", sa.String(16), nullable=False, server_default="scheduled"),
        sa.Column("job_id", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column(
            "analyzed_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("creator", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("modifier", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("is_deleted", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.PrimaryKeyConstraint("asset_analysis_id"),
    )
    # Sequence must start at 100000 to isolate from richman ID space (SS21.4)
    op.execute(
        "CREATE SEQUENCE IF NOT EXISTS rs_asset_analyses_asset_analysis_id_seq "
        "START WITH 100000 OWNED BY rs_asset_analyses.asset_analysis_id"
    )
    op.execute(
        "ALTER TABLE rs_asset_analyses ALTER COLUMN asset_analysis_id "
        "SET DEFAULT nextval('rs_asset_analyses_asset_analysis_id_seq')"
    )
    op.create_index(
        "idx_rsa_asset_latest",
        "rs_asset_analyses",
        ["asset_code", sa.text("analyzed_at DESC")],
        postgresql_where=sa.text("is_deleted = 0"),
    )
    op.create_index(
        "idx_rsa_asset_date",
        "rs_asset_analyses",
        ["asset_code", "analyzed_at"],
        postgresql_where=sa.text("is_deleted = 0"),
    )

    # ------------------------------------------------------------------
    # rs_asset_analysis_dimensions
    # ------------------------------------------------------------------
    op.create_table(
        "rs_asset_analysis_dimensions",
        sa.Column("id", sa.BigInteger(), nullable=False),
        sa.Column("asset_analysis_id", sa.BigInteger(), nullable=False),
        sa.Column("dimension", sa.String(8), nullable=False),
        sa.Column("sub_indicator", sa.String(64), nullable=False),
        sa.Column("raw_value", sa.Numeric(20, 6), nullable=True),
        sa.Column("percentile_1y", sa.Numeric(5, 2), nullable=True),
        sa.Column("percentile_5y", sa.Numeric(5, 2), nullable=True),
        sa.Column("blended_percentile", sa.Numeric(5, 2), nullable=True),
        sa.Column("normalized_score", sa.Numeric(5, 2), nullable=True),
        sa.Column("weight_in_dimension", sa.Numeric(4, 2), nullable=True),
        sa.Column("data_source", sa.String(32), nullable=True),
        sa.Column("data_as_of", sa.Date(), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("creator", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("modifier", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("is_deleted", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.ForeignKeyConstraint(
            ["asset_analysis_id"],
            ["rs_asset_analyses.asset_analysis_id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    # SS21.4: explicit sequence RESTART WITH 100000
    op.execute(
        "CREATE SEQUENCE IF NOT EXISTS rs_asset_analysis_dimensions_id_seq "
        "START WITH 100000 OWNED BY rs_asset_analysis_dimensions.id"
    )
    op.execute(
        "ALTER TABLE rs_asset_analysis_dimensions ALTER COLUMN id "
        "SET DEFAULT nextval('rs_asset_analysis_dimensions_id_seq')"
    )
    op.create_index(
        "idx_rsad_analysis",
        "rs_asset_analysis_dimensions",
        ["asset_analysis_id"],
        postgresql_where=sa.text("is_deleted = 0"),
    )

    # ------------------------------------------------------------------
    # rs_analysis_jobs
    # ------------------------------------------------------------------
    op.create_table(
        "rs_analysis_jobs",
        sa.Column("job_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("asset_code", sa.String(32), nullable=False),
        sa.Column("job_type", sa.String(32), nullable=False, server_default="asset_analysis"),
        sa.Column("status", sa.String(16), nullable=False, server_default="pending"),
        sa.Column("progress", sa.Numeric(4, 2), nullable=False, server_default="0"),
        sa.Column("current_step", sa.String(64), nullable=True),
        sa.Column(
            "steps",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default="[]",
        ),
        sa.Column("error_message", sa.Text(), nullable=True),
        sa.Column("error_code", sa.String(64), nullable=True),
        sa.Column("asset_analysis_id", sa.BigInteger(), nullable=True),
        sa.Column(
            "expires_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW() + INTERVAL '1 hour'"),
        ),
        sa.Column("started_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("completed_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("request_id", postgresql.UUID(as_uuid=True), nullable=True),
        sa.Column("locale", sa.String(8), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("creator", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("modifier", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("is_deleted", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.PrimaryKeyConstraint("job_id"),
    )
    # Partial unique index: prevent duplicate concurrent jobs for same asset
    op.create_index(
        "uq_rsj_asset_active",
        "rs_analysis_jobs",
        ["asset_code"],
        unique=True,
        postgresql_where=sa.text("status IN ('pending', 'running') AND is_deleted = 0"),
    )
    op.create_index(
        "idx_rsj_status",
        "rs_analysis_jobs",
        ["status"],
        postgresql_where=sa.text("is_deleted = 0"),
    )
    op.create_index(
        "idx_rsj_expires",
        "rs_analysis_jobs",
        ["expires_at"],
        postgresql_where=sa.text("status IN ('pending', 'running') AND is_deleted = 0"),
    )

    # ------------------------------------------------------------------
    # rs_event_alerts
    # ------------------------------------------------------------------
    op.create_table(
        "rs_event_alerts",
        sa.Column("id", sa.BigInteger(), nullable=False),
        sa.Column("event_slug", sa.String(128), nullable=False),
        sa.Column("event_title", sa.Text(), nullable=False),
        sa.Column("source", sa.String(32), nullable=False, server_default="polymarket"),
        sa.Column("prev_probability", sa.Numeric(5, 4), nullable=False),
        sa.Column("curr_probability", sa.Numeric(5, 4), nullable=False),
        sa.Column("delta", sa.Numeric(5, 4), nullable=False),
        sa.Column("threshold", sa.Numeric(5, 4), nullable=False, server_default="0.20"),
        sa.Column("gold_direction", sa.String(16), nullable=True),
        sa.Column("alerted", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column(
            "detected_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("creator", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("modifier", sa.String(64), nullable=False, server_default="richson"),
        sa.Column("is_deleted", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.PrimaryKeyConstraint("id"),
    )
    # SS21.4: explicit sequence RESTART WITH 100000
    op.execute(
        "CREATE SEQUENCE IF NOT EXISTS rs_event_alerts_id_seq "
        "START WITH 100000 OWNED BY rs_event_alerts.id"
    )
    op.execute(
        "ALTER TABLE rs_event_alerts ALTER COLUMN id "
        "SET DEFAULT nextval('rs_event_alerts_id_seq')"
    )
    op.create_index(
        "uq_rsea_slug_active",
        "rs_event_alerts",
        ["event_slug"],
        unique=True,
        postgresql_where=sa.text("is_deleted = 0 AND alerted = FALSE"),
    )
    op.create_index(
        "idx_rsea_unalerted",
        "rs_event_alerts",
        ["alerted"],
        postgresql_where=sa.text("alerted = FALSE AND is_deleted = 0"),
    )

    # ------------------------------------------------------------------
    # rs_dimension_definitions
    # ------------------------------------------------------------------
    op.create_table(
        "rs_dimension_definitions",
        sa.Column("id", sa.BigInteger(), nullable=False),
        sa.Column("asset_type", sa.String(32), nullable=False),
        sa.Column("dimension", sa.String(8), nullable=False),
        sa.Column("name_zh", sa.String(32), nullable=False),
        sa.Column("name_en", sa.String(32), nullable=False),
        sa.Column("weight", sa.Numeric(4, 2), nullable=False),
        sa.Column("description_zh", sa.Text(), nullable=True),
        sa.Column("description_en", sa.Text(), nullable=True),
        sa.Column("display_order", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.Column("model_version", sa.String(32), nullable=False),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("creator", sa.String(64), nullable=False, server_default="system"),
        sa.Column("modifier", sa.String(64), nullable=False, server_default="system"),
        sa.Column("is_deleted", sa.SmallInteger(), nullable=False, server_default="0"),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index(
        "uq_rsdd_type_dim_version",
        "rs_dimension_definitions",
        ["asset_type", "dimension", "model_version"],
        unique=True,
        postgresql_where=sa.text("is_deleted = 0"),
    )

    # ------------------------------------------------------------------
    # Seed: gold_v1.0 dimension definitions
    # ------------------------------------------------------------------
    op.execute("""
        INSERT INTO rs_dimension_definitions
            (asset_type, dimension, name_zh, name_en, weight,
             description_zh, description_en, display_order, model_version)
        VALUES
            ('gold', 'D1', '宏观利率', 'Macro Rates', 0.30,
             '实际利率与通胀预期对黄金机会成本的影响',
             'Impact of real rates and inflation expectations on gold opportunity cost',
             1, 'gold_v1.0'),
            ('gold', 'D2', '美元流动性', 'Dollar Liquidity', 0.25,
             '美元指数和全球流动性环境',
             'USD index and global liquidity conditions',
             2, 'gold_v1.0'),
            ('gold', 'D3', '结构性需求', 'Structural Demand', 0.25,
             '央行购金、ETF持仓、地缘风险溢价',
             'Central bank buying, ETF holdings, geopolitical risk premium',
             3, 'gold_v1.0'),
            ('gold', 'D4', '技术位置', 'Technical Position', 0.20,
             '价格动量、均线、唐奇安通道、相对强弱',
             'Price momentum, moving averages, Donchian channel, relative strength',
             4, 'gold_v1.0')
    """)


def downgrade() -> None:
    op.drop_table("rs_dimension_definitions")
    op.drop_table("rs_event_alerts")
    op.drop_table("rs_analysis_jobs")
    op.drop_table("rs_asset_analysis_dimensions")
    op.drop_table("rs_asset_analyses")
