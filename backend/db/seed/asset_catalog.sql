-- Seed the canonical asset catalog for the market overview page.
-- `name` holds the Chinese display name, `name_en` holds the English display
-- name. The frontend picks the column based on the active locale, so keeping
-- these in sync with their intended languages is load-bearing.
INSERT INTO rm_asset_catalog (code, name, name_en, asset_type, exchange, data_source) VALUES
-- gold_etf
('GLD',    'SPDR 黄金 ETF',       'SPDR Gold Shares',        'gold_etf', 'NYSE', 'yahoo'),
('IAU',    'iShares 黄金信托',    'iShares Gold Trust',      'gold_etf', 'NYSE', 'yahoo'),
('518880', '华安黄金 ETF',        'HuaAn Gold ETF',          'gold_etf', 'SSE',  'akshare'),
('159934', '易方达黄金 ETF',      'Gold ETF',                'gold_etf', 'SZSE', 'akshare'),

-- a_share_broad
('510300', '沪深 300 ETF',        'CSI 300 ETF',             'a_share_broad', 'SSE',  'akshare'),
('510500', '中证 500 ETF',        'CSI 500 ETF',             'a_share_broad', 'SSE',  'akshare'),
('159915', '创业板 ETF',          'ChiNext ETF',             'a_share_broad', 'SZSE', 'akshare'),
('510050', '上证 50 ETF',         'SSE 50 ETF',              'a_share_broad', 'SSE',  'akshare'),
('512100', '中证 1000 ETF',       'CSI 1000 ETF',            'a_share_broad', 'SSE',  'akshare'),

-- a_share_industry
('515030', '新能源 ETF',          'New Energy ETF',          'a_share_industry', 'SSE',  'akshare'),
('512660', '军工 ETF',            'Military ETF',            'a_share_industry', 'SSE',  'akshare'),
('512010', '医药 ETF',            'Medicine ETF',            'a_share_industry', 'SSE',  'akshare'),
('515790', '光伏 ETF',            'Photovoltaic ETF',        'a_share_industry', 'SSE',  'akshare'),
('512480', '半导体 ETF',          'Semiconductor ETF',       'a_share_industry', 'SSE',  'akshare'),
('516160', '新能源车 ETF',        'New Energy Vehicle ETF',  'a_share_industry', 'SSE',  'akshare'),
('512800', '银行 ETF',            'Bank ETF',                'a_share_industry', 'SSE',  'akshare'),
('159766', '旅游 ETF',            'Tourism ETF',             'a_share_industry', 'SZSE', 'akshare'),
('512690', '酒 ETF',              'Liquor ETF',              'a_share_industry', 'SSE',  'akshare'),
('512170', '医疗 ETF',            'Healthcare ETF',          'a_share_industry', 'SSE',  'akshare'),

-- us_stock
('QQQ',  '纳斯达克 100 ETF',      'Invesco QQQ Trust',              'us_stock', 'NASDAQ', 'yahoo'),
('SPY',  '标普 500 ETF',          'SPDR S&P 500 ETF',               'us_stock', 'NYSE',   'yahoo'),
('VOO',  '先锋标普 500 ETF',      'Vanguard S&P 500 ETF',           'us_stock', 'NYSE',   'yahoo'),
('IWM',  '罗素 2000 ETF',         'iShares Russell 2000 ETF',       'us_stock', 'NYSE',   'yahoo'),
('VTI',  '先锋全市场 ETF',        'Vanguard Total Stock Market ETF','us_stock', 'NYSE',   'yahoo'),
('ARKK', 'ARK 创新 ETF',          'ARK Innovation ETF',             'us_stock', 'NYSE',   'yahoo'),
('SOXX', 'iShares 半导体 ETF',    'iShares Semiconductor ETF',      'us_stock', 'NASDAQ', 'yahoo')
ON CONFLICT DO NOTHING;
