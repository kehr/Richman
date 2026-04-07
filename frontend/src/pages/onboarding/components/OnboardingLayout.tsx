import { Typography } from "@/ui-kit/eat";
import type { ReactNode } from "react";
import { StepIndicator } from "./StepIndicator";

const { Title, Paragraph } = Typography;

interface OnboardingLayoutProps {
	currentStep: number;
	totalSteps?: number;
	title: string;
	description?: ReactNode;
	children: ReactNode;
	footer?: ReactNode;
}

// OnboardingLayout is the shared single-column shell used by every onboarding
// page (PRD §2.3). It renders the progress indicator, page title and optional
// description, the main content slot, and a sticky-ish footer slot for the
// primary action button pair. The layout is intentionally static (no animated
// transitions between steps) so focus management and scroll state stay
// predictable.
export function OnboardingLayout({
	currentStep,
	totalSteps = 4,
	title,
	description,
	children,
	footer,
}: OnboardingLayoutProps) {
	return (
		<div
			style={{
				minHeight: "100vh",
				display: "flex",
				justifyContent: "center",
				padding: "48px 24px",
				backgroundColor: "#fafafa",
			}}
		>
			<div style={{ width: "100%", maxWidth: 720 }}>
				<StepIndicator currentStep={currentStep} totalSteps={totalSteps} />
				<Title level={2} style={{ textAlign: "center", marginBottom: 12 }}>
					{title}
				</Title>
				{description ? (
					<Paragraph
						type="secondary"
						style={{ textAlign: "center", marginBottom: 32, fontSize: 15 }}
					>
						{description}
					</Paragraph>
				) : (
					<div style={{ marginBottom: 24 }} />
				)}
				<div>{children}</div>
				{footer ? (
					<div
						style={{
							display: "flex",
							justifyContent: "flex-end",
							gap: 12,
							marginTop: 40,
						}}
					>
						{footer}
					</div>
				) : null}
			</div>
		</div>
	);
}
