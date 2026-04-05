INSERT INTO asset_catalog (code, name, name_en, asset_type, exchange, data_source) VALUES
-- gold_etf
('GLD', 'SPDR Gold Shares', 'SPDR Gold Shares', 'gold_etf', 'NYSE', 'yahoo'),
('IAU', 'iShares Gold Trust', 'iShares Gold Trust', 'gold_etf', 'NYSE', 'yahoo'),
('518880', 'HuaAn Gold ETF', 'HuaAn Gold ETF', 'gold_etf', 'SSE', 'akshare'),
('159934', 'Gold ETF', 'Gold ETF', 'gold_etf', 'SZSE', 'akshare'),

-- a_share_broad
('510300', 'CSI 300 ETF', 'CSI 300 ETF', 'a_share_broad', 'SSE', 'akshare'),
('510500', 'CSI 500 ETF', 'CSI 500 ETF', 'a_share_broad', 'SSE', 'akshare'),
('159915', 'ChiNext ETF', 'ChiNext ETF', 'a_share_broad', 'SZSE', 'akshare'),
('510050', 'SSE 50 ETF', 'SSE 50 ETF', 'a_share_broad', 'SSE', 'akshare'),
('512100', 'CSI 1000 ETF', 'CSI 1000 ETF', 'a_share_broad', 'SSE', 'akshare'),

-- a_share_industry
('515030', 'New Energy ETF', 'New Energy ETF', 'a_share_industry', 'SSE', 'akshare'),
('512660', 'Military ETF', 'Military ETF', 'a_share_industry', 'SSE', 'akshare'),
('512010', 'Medicine ETF', 'Medicine ETF', 'a_share_industry', 'SSE', 'akshare'),
('515790', 'Photovoltaic ETF', 'Photovoltaic ETF', 'a_share_industry', 'SSE', 'akshare'),
('512480', 'Semiconductor ETF', 'Semiconductor ETF', 'a_share_industry', 'SSE', 'akshare'),
('516160', 'NEV ETF', 'New Energy Vehicle ETF', 'a_share_industry', 'SSE', 'akshare'),
('512800', 'Bank ETF', 'Bank ETF', 'a_share_industry', 'SSE', 'akshare'),
('159766', 'Tourism ETF', 'Tourism ETF', 'a_share_industry', 'SZSE', 'akshare'),
('512690', 'Liquor ETF', 'Liquor ETF', 'a_share_industry', 'SSE', 'akshare'),
('512170', 'Healthcare ETF', 'Healthcare ETF', 'a_share_industry', 'SSE', 'akshare'),

-- us_stock
('QQQ', 'Invesco QQQ Trust', 'Invesco QQQ Trust', 'us_stock', 'NASDAQ', 'yahoo'),
('SPY', 'SPDR S&P 500 ETF', 'SPDR S&P 500 ETF', 'us_stock', 'NYSE', 'yahoo'),
('VOO', 'Vanguard S&P 500 ETF', 'Vanguard S&P 500 ETF', 'us_stock', 'NYSE', 'yahoo'),
('IWM', 'iShares Russell 2000 ETF', 'iShares Russell 2000 ETF', 'us_stock', 'NYSE', 'yahoo'),
('VTI', 'Vanguard Total Stock Market ETF', 'Vanguard Total Stock Market ETF', 'us_stock', 'NYSE', 'yahoo'),
('ARKK', 'ARK Innovation ETF', 'ARK Innovation ETF', 'us_stock', 'NYSE', 'yahoo'),
('SOXX', 'iShares Semiconductor ETF', 'iShares Semiconductor ETF', 'us_stock', 'NASDAQ', 'yahoo')
ON CONFLICT DO NOTHING;
