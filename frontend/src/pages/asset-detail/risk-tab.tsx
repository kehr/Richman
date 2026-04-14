import type { AssetDetailDto } from "@/features/asset-detail";
import { DrawdownReference } from "./drawdown-reference";
import { EventCalendar } from "./event-calendar";
import { KeyPriceLevels } from "./key-price-levels";
import { RiskFactorList } from "./risk-factor-list";

interface Props {
	detail: AssetDetailDto;
}

// RiskTab is lazy-loaded (enabled only when the user clicks the Risk tab).
// Each child accepts optional inputs because backend MVP may omit sections.
export function RiskTab({ detail }: Props) {
	return (
		<div style={{ padding: "16px 0" }}>
			<RiskFactorList factors={detail.riskFactors ?? []} />
			<KeyPriceLevels
				levels={detail.keyPriceLevels ?? []}
				currentPrice={detail.currentPrice}
				currency={detail.currency}
				usdExchangeRate={detail.usdExchangeRate ?? null}
			/>
			{detail.drawdownReference && <DrawdownReference drawdown={detail.drawdownReference} />}
			<EventCalendar />
		</div>
	);
}
