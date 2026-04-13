import type { HoldingDto } from "@/features/portfolio";
import { Alert, Space } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface ConcentrationAlertProps {
	holdings: HoldingDto[];
}

interface AssetConcentration {
	assetType: string;
	totalRatio: number;
}

// computeConcentrations sums positionRatio by assetType across all holdings.
// Only types whose combined ratio exceeds 10% (the minimum alert threshold)
// are returned, sorted descending by ratio.
function computeConcentrations(holdings: HoldingDto[]): AssetConcentration[] {
	const byType = new Map<string, number>();
	for (const h of holdings) {
		if (h.positionRatio == null || !h.assetType) continue;
		byType.set(h.assetType, (byType.get(h.assetType) ?? 0) + h.positionRatio);
	}
	return Array.from(byType.entries())
		.filter(([, ratio]) => ratio > 10)
		.map(([assetType, totalRatio]) => ({ assetType, totalRatio }))
		.sort((a, b) => b.totalRatio - a.totalRatio);
}

// ConcentrationAlert displays one Alert per asset type that exceeds concentration
// thresholds (TRD SS7.4):
//   >= 35% -> error (red)
//   >= 25% -> warning (orange)
//   >= 15% -> info (blue)
export function ConcentrationAlert({ holdings }: ConcentrationAlertProps) {
	const { t } = useTranslation("app");

	const concentrations = computeConcentrations(holdings);
	if (concentrations.length === 0) return null;

	return (
		<Space direction="vertical" size={8} style={{ width: "100%", marginBottom: 12 }}>
			{concentrations.map(({ assetType, totalRatio }) => {
				const type = totalRatio >= 35 ? "error" : totalRatio >= 25 ? "warning" : "info";

				const assetLabel = t(`common:assetCategory.${assetType}.label`, {
					defaultValue: assetType,
				});

				const messageKey =
					totalRatio >= 35
						? "portfolio.concentrationAlert.error"
						: totalRatio >= 25
							? "portfolio.concentrationAlert.warning"
							: "portfolio.concentrationAlert.info";

				return (
					<Alert
						key={assetType}
						type={type}
						showIcon
						message={t(messageKey, {
							type: assetLabel,
							ratio: totalRatio.toFixed(1),
						})}
					/>
				);
			})}
		</Space>
	);
}
