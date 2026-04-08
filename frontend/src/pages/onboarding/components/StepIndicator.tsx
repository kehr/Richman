import { Typography } from "@/ui-kit/eat";

const { Text } = Typography;

interface StepIndicatorProps {
	currentStep: number;
	totalSteps: number;
}

// StepIndicator renders a simple numbered dot row for the onboarding flow.
// Completed and current steps use the primary black brand color, upcoming
// steps stay muted. The text label at the end (e.g. "第 2 / 4 步") is kept
// in sync so screen readers and low-vision users get the same information.
export function StepIndicator({ currentStep, totalSteps }: StepIndicatorProps) {
	const dots = Array.from({ length: totalSteps }, (_, index) => index + 1);
	return (
		<div
			data-testid="onboarding-step-indicator"
			style={{
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
				gap: 12,
				marginBottom: 32,
			}}
		>
			{dots.map((step) => {
				const active = step <= currentStep;
				return (
					<div
						key={step}
						data-testid={`step-dot-${step}`}
						aria-current={step === currentStep ? "step" : undefined}
						style={{
							width: 10,
							height: 10,
							borderRadius: 999,
							backgroundColor: active ? "#000" : "#d9d9d9",
							transition: "background-color 0.2s",
						}}
					/>
				);
			})}
			<Text type="secondary" style={{ marginLeft: 8, fontSize: 13 }}>
				第 {currentStep} / {totalSteps} 步
			</Text>
		</div>
	);
}
