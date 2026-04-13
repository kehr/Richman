import { useCurrentUser } from "@/domain/auth/use-current-user";
import type { AssetDetailDto } from "@/features/asset-detail";
import { useHoldingByAssetCode } from "@/features/portfolio";
import { DemoPlanAddHoldingCta } from "./demo-plan-add-holding-cta";
import { DemoPlanRegisterCta } from "./demo-plan-register-cta";
import { FullExecutionPlan } from "./full-execution-plan";

interface Props {
	detail: AssetDetailDto;
}

// ExecutionTab implements the three-state logic from TRD SS5.5:
// 1. Unauthenticated: demo plan + register CTA
// 2. Authenticated, no holding: demo plan + add-holding CTA
// 3. Authenticated, has holding: full personalized execution plan
export function ExecutionTab({ detail }: Props) {
	const { data: user } = useCurrentUser();
	const { data: holding } = useHoldingByAssetCode(detail.code);

	if (!user) {
		return <DemoPlanRegisterCta code={detail.code} />;
	}

	if (!holding) {
		return <DemoPlanAddHoldingCta code={detail.code} />;
	}

	return <FullExecutionPlan detail={detail} holding={holding} />;
}
