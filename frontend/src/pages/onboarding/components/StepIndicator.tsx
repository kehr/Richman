import { Typography } from "@/ui-kit/eat";
import { motion, useReducedMotion } from "framer-motion";
import type { CSSProperties } from "react";

const { Text } = Typography;

interface StepIndicatorProps {
	currentStep: number;
	totalSteps: number;
	// reachedStep is the monotonic watermark of steps the user has unlocked.
	// When combined with onStepClick, dots whose step number is <= reachedStep
	// become clickable shortcuts back to earlier steps. Forward jumps past
	// reachedStep are silently disallowed (matching useOnboardingNav.jumpTo).
	reachedStep?: number;
	// onStepClick, when provided, turns each reachable dot into a button-like
	// clickable element. Pages that use StepIndicator without navigation (e.g.
	// the standalone component playground) can omit this to keep the existing
	// non-interactive rendering.
	onStepClick?: (step: number) => void;
}

// StepIndicator renders a simple numbered dot row for the onboarding flow.
// Completed and current steps use the primary black brand color, upcoming
// steps stay muted. The text label at the end (e.g. "第 2 / 4 步") is kept
// in sync so screen readers and low-vision users get the same information.
//
// When `onStepClick` is provided, every dot whose step number is <=
// `reachedStep` (defaulting to currentStep) renders as a <button> with a
// pointer cursor. The active dot also receives a gentle scale pulse via
// framer-motion to reinforce "you are here" — suppressed if the user has
// enabled reduced-motion.
export function StepIndicator({
	currentStep,
	totalSteps,
	reachedStep,
	onStepClick,
}: StepIndicatorProps) {
	const reducedMotion = useReducedMotion();
	const watermark = reachedStep ?? currentStep;
	const dots = Array.from({ length: totalSteps }, (_, index) => index + 1);

	return (
		<div
			data-testid="onboarding-step-indicator"
			style={{
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
				gap: 12,
			}}
		>
			{dots.map((step) => {
				const active = step <= currentStep;
				const isCurrent = step === currentStep;
				const reachable = onStepClick !== undefined && step <= watermark;
				const backgroundColor = active ? "#000" : "#d9d9d9";

				// When clickable, render a <button> so keyboard + screen-reader users
				// get the same navigation affordance as mouse users. The base dot
				// styling is kept on the inner motion.div so pulse animation can target
				// it without the button chrome interfering.
				if (reachable && onStepClick) {
					return (
						<button
							key={step}
							type="button"
							data-testid={`step-dot-${step}`}
							aria-current={isCurrent ? "step" : undefined}
							aria-label={`第 ${step} / ${totalSteps} 步`}
							onClick={() => onStepClick(step)}
							style={buttonResetStyle}
						>
							<motion.span
								style={{ ...dotStyle, backgroundColor }}
								animate={isCurrent && !reducedMotion ? { scale: [1, 1.15, 1] } : { scale: 1 }}
								transition={
									isCurrent && !reducedMotion
										? {
												duration: 1.4,
												repeat: Number.POSITIVE_INFINITY,
												ease: "easeInOut",
											}
										: { duration: 0 }
								}
							/>
						</button>
					);
				}

				return (
					<motion.div
						key={step}
						data-testid={`step-dot-${step}`}
						aria-current={isCurrent ? "step" : undefined}
						style={{ ...dotStyle, backgroundColor }}
						animate={isCurrent && !reducedMotion ? { scale: [1, 1.15, 1] } : { scale: 1 }}
						transition={
							isCurrent && !reducedMotion
								? {
										duration: 1.4,
										repeat: Number.POSITIVE_INFINITY,
										ease: "easeInOut",
									}
								: { duration: 0 }
						}
					/>
				);
			})}
			<Text type="secondary" style={{ marginLeft: 8, fontSize: 13 }}>
				第 {currentStep} / {totalSteps} 步
			</Text>
		</div>
	);
}

// Shared dot styling lifted to module scope so both the clickable and the
// static render paths stay byte-identical.
const dotStyle: CSSProperties = {
	display: "inline-block",
	width: 10,
	height: 10,
	borderRadius: 999,
	transition: "background-color 0.2s",
};

// Button reset for clickable dots — strip the default browser chrome while
// keeping keyboard focusability and hit area parity with the static dot.
const buttonResetStyle: CSSProperties = {
	display: "inline-flex",
	alignItems: "center",
	justifyContent: "center",
	padding: 4,
	margin: 0,
	border: "none",
	background: "transparent",
	cursor: "pointer",
};
