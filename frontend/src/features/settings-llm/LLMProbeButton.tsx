import { App, Button } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useProbeLLMSettings } from "./hooks";

interface LLMProbeButtonProps {
	// label overrides the default "test connectivity" copy. The FailingCard
	// surface uses "retest" to match the PRD wording.
	label?: string;
	// disabled forces the button into a disabled state. The form uses this
	// while upsert is mid-flight so a user cannot trigger overlapping
	// probe + save round-trips against the same row.
	disabled?: boolean;
}

// LLMProbeButton wraps useProbeLLMSettings with a Button and a toast. The
// button is stateless from the caller's perspective — click it, get a
// green or red toast. The probe endpoint already persists the updated
// health status to the row, so the llm-settings cache is invalidated
// automatically by the hook's onSettled callback.
export function LLMProbeButton({ label, disabled = false }: LLMProbeButtonProps) {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const probeMutation = useProbeLLMSettings();

	const buttonLabel = label ?? t("llm.probeButton.default");

	const handleClick = async () => {
		try {
			const result = await probeMutation.mutateAsync();
			if (result.healthy) {
				message.success(t("llm.probeButton.success", { latency: result.latencyMs }));
			} else {
				message.error(
					t("llm.probeButton.failure", { error: result.error ?? t("llm.failingCard.unknown") }),
				);
			}
		} catch (err) {
			const msg = err instanceof Error ? err.message : t("action.retry", { ns: "common" });
			message.error(t("llm.probeButton.requestError", { msg }));
		}
	};

	return (
		<Button
			onClick={handleClick}
			loading={probeMutation.isPending}
			disabled={disabled || probeMutation.isPending}
			data-testid="llm-probe-button"
		>
			{buttonLabel}
		</Button>
	);
}
