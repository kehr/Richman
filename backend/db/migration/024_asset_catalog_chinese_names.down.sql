-- Revert rm_asset_catalog.name to the pre-024 English values. name_en was
-- never modified, so copying name_en back into name restores the original
-- seed state for the 26 canonical codes.
UPDATE rm_asset_catalog
SET name = name_en,
    updated_at = NOW()
WHERE code IN (
    'GLD','IAU','518880','159934',
    '510300','510500','159915','510050','512100',
    '515030','512660','512010','515790','512480','516160','512800','159766','512690','512170',
    'QQQ','SPY','VOO','IWM','VTI','ARKK','SOXX'
);
