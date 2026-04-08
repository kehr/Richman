import { App, Button } from "@/ui-kit/eat";
import { useProbeLLMSettings } from "./hooks";

interface LLMProbeButtonProps {
	// label overrides the default "测试连通性" copy. The FailingCard surface
	// uses "重新测试" to match the PRD wording.
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
export function LLMProbeButton({ label = "测试连通性", disabled = false }: LLMProbeButtonProps) {
	const { message } = App.useApp();
	const probeMutation = useProbeLLMSettings();

	const handleClick = async () => {
		try {
			const result = await probeMutation.mutateAsync();
			if (result.healthy) {
				message.success(`测试通过（${result.latencyMs} ms）`);
			} else {
				message.error(`测试失败：${result.error ?? "未知错误"}`);
			}
		} catch (err) {
			const msg = err instanceof Error ? err.message : "请稍后再试";
			message.error(`测试请求失败：${msg}`);
		}
	};

	return (
		<Button
			onClick={handleClick}
			loading={probeMutation.isPending}
			disabled={disabled || probeMutation.isPending}
			data-testid="llm-probe-button"
		>
			{label}
		</Button>
	);
}
