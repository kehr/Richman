-- Align rm_asset_catalog.name with its intended semantics: Chinese display
-- name. Previous seed loaded English copies into both name and name_en, which
-- made the zh locale fall back to English on the market overview page.
--
-- UPDATE rewrites the 26 canonical codes in one statement. Rows with codes
-- outside this set (none today, but possible for per-env additions) are
-- untouched. The corresponding down migration restores English to match the
-- pre-fix seed.
UPDATE rm_asset_catalog
SET name = CASE code
    WHEN 'GLD'    THEN 'SPDR 黄金 ETF'
    WHEN 'IAU'    THEN 'iShares 黄金信托'
    WHEN '518880' THEN '华安黄金 ETF'
    WHEN '159934' THEN '易方达黄金 ETF'
    WHEN '510300' THEN '沪深 300 ETF'
    WHEN '510500' THEN '中证 500 ETF'
    WHEN '159915' THEN '创业板 ETF'
    WHEN '510050' THEN '上证 50 ETF'
    WHEN '512100' THEN '中证 1000 ETF'
    WHEN '515030' THEN '新能源 ETF'
    WHEN '512660' THEN '军工 ETF'
    WHEN '512010' THEN '医药 ETF'
    WHEN '515790' THEN '光伏 ETF'
    WHEN '512480' THEN '半导体 ETF'
    WHEN '516160' THEN '新能源车 ETF'
    WHEN '512800' THEN '银行 ETF'
    WHEN '159766' THEN '旅游 ETF'
    WHEN '512690' THEN '酒 ETF'
    WHEN '512170' THEN '医疗 ETF'
    WHEN 'QQQ'    THEN '纳斯达克 100 ETF'
    WHEN 'SPY'    THEN '标普 500 ETF'
    WHEN 'VOO'    THEN '先锋标普 500 ETF'
    WHEN 'IWM'    THEN '罗素 2000 ETF'
    WHEN 'VTI'    THEN '先锋全市场 ETF'
    WHEN 'ARKK'   THEN 'ARK 创新 ETF'
    WHEN 'SOXX'   THEN 'iShares 半导体 ETF'
    ELSE name
END,
    updated_at = NOW()
WHERE code IN (
    'GLD','IAU','518880','159934',
    '510300','510500','159915','510050','512100',
    '515030','512660','512010','515790','512480','516160','512800','159766','512690','512170',
    'QQQ','SPY','VOO','IWM','VTI','ARKK','SOXX'
);
