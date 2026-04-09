import { App, Button, LeftOutlined, Typography } from "@/ui-kit/eat";
import { motion } from "framer-motion";
import { type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { SHAKE_EVENT_NAME, useOnboardingNav } from "../use-onboarding-nav";
import { OnboardingBackground } from "./OnboardingBackground";
import { OnboardingPageTransition } from "./OnboardingPageTransition";
import { StepIndicator } from "./StepIndicator";

const { Title, Paragraph } = Typography;

interface OnboardingLayoutProps {
	// currentStep is narrowed to the four-step literal union so every caller
	// and every downstream consumer (StepIndicator, OnboardingBackground,
	// OnboardingPageTransition) shares one domain. Pages pass numeric literals
	// already; tsc narrows them automatically.
	currentStep: 1 | 2 | 3 | 4;
	title: string;
	description?: ReactNode;
	children: ReactNode;
	footer?: ReactNode;
}

const TOTAL_STEPS = 4;

// OnboardingLayout is the shared three-region shell used by every onboarding
// page (PRD §3.3, TRD §5.1). It renders:
//
//   1. A fixed decorative background (OnboardingBackground) at z-index 0
//   2. A sticky header bar with ← prev / clickable StepIndicator / skip link
//   3. A centered title + optional description
//   4. The animated content slot (OnboardingPageTransition) wrapping children
//   5. An optional footer slot that shakes when next() is rejected
//
// The layout owns cross-cutting wiring that must behave identically across
// all four pages: the skip confirm Modal (with setTimeout(0) deferral to
// avoid focus-trap fights with framer-motion, see PRD App. D bug #1), the
// global keyboard handler for ArrowLeft/ArrowRight/Escape (filtered out in
// form fields and while a Modal is open), and the shake event subscription
// that retriggers a key-driven horizontal nudge on the footer slot.
export function OnboardingLayout({
	currentStep,
	title,
	description,
	children,
	footer,
}: OnboardingLayoutProps) {
	const nav = useOnboardingNav();
	// Use the App-scoped Modal and message hooks so the confirm dialog and the
	// error toast render through the active ConfigProvider context. Static
	// Modal.confirm / message.error are broken under React 19 + antd 5 without
	// the compat patch; App.useApp() is the officially recommended path.
	const { modal, message } = App.useApp();

	// Track the previous step so we can tell forward vs backward navigation and
	// feed the direction into OnboardingPageTransition. Default to "forward" on
	// first render — the user is entering the flow.
	const previousStepRef = useRef<number>(currentStep);
	const direction: "forward" | "backward" =
		currentStep >= previousStepRef.current ? "forward" : "backward";
	// Update the ref after computing direction so the next render's comparison
	// uses the previous value. A layout effect is not required here — reading
	// and writing in render is safe because the ref does not participate in
	// reconciliation.
	previousStepRef.current = currentStep;

	// shakeKey increments on every shake event so remounting the motion.div via
	// the key prop retriggers the shake animation. Pages can rely on this
	// mechanism without owning any state themselves; the footer slot they pass
	// in is wrapped automatically below.
	const [shakeKey, setShakeKey] = useState(0);

	// handleSkip opens the skip confirm Modal. The setTimeout(0) is critical:
	// it defers Modal mounting until the current framer-motion animation frame
	// has drained, avoiding the focus-trap vs page-transition race documented
	// in PRD Appendix D, Pre-mortem bug #1.
	const handleSkip = useCallback(() => {
		setTimeout(() => {
			modal.confirm({
				title: "跳过引导？",
				content:
					"你可以稍后在 Settings 重新发起引导，或在 Dashboard 顶部的提示条点击「开始引导」回到这里。",
				okText: "跳过",
				cancelText: "继续引导",
				okType: "primary",
				onOk: async () => {
					try {
						await nav.skip();
					} catch (err) {
						// Re-throw so antd keeps the Modal open on failure; the toast
						// above tells the user what went wrong and lets them retry.
						message.error("跳过失败，请稍后重试");
						throw err;
					}
				},
			});
		}, 0);
	}, [modal, message, nav]);

	// Shake subscription: bump the key counter on every CustomEvent so the
	// motion.div wrapping the footer remounts and replays its animate sequence.
	useEffect(() => {
		const handler = () => setShakeKey((k) => k + 1);
		window.addEventListener(SHAKE_EVENT_NAME, handler);
		return () => window.removeEventListener(SHAKE_EVENT_NAME, handler);
	}, []);

	// Global keyboard handler. The form-field check uses tagName because input
	// focus-trapping inside antd selects/checkboxes bubbles their native events
	// up to the document, and we do not want ArrowLeft in a text field to drag
	// the user to the previous step. The Modal-open check uses a DOM query
	// because antd does not expose a global "modal open" signal; the
	// .ant-modal-mask class is part of antd's stable public DOM.
	useEffect(() => {
		const handler = (e: KeyboardEvent) => {
			const target = e.target as HTMLElement | null;
			if (target) {
				const tag = target.tagName;
				if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
					return;
				}
				if (target.isContentEditable) {
					return;
				}
			}
			if (document.querySelector(".ant-modal-mask")) {
				return;
			}
			if (e.key === "ArrowLeft") {
				e.preventDefault();
				nav.prev();
			} else if (e.key === "ArrowRight") {
				e.preventDefault();
				void nav.next();
			} else if (e.key === "Escape") {
				e.preventDefault();
				handleSkip();
			}
		};
		window.addEventListener("keydown", handler);
		return () => window.removeEventListener("keydown", handler);
	}, [nav, handleSkip]);

	return (
		<div style={rootStyle}>
			<OnboardingBackground currentStep={currentStep} />
			<div style={contentStyle}>
				<div style={containerStyle}>
					<header style={headerStyle}>
						<div style={headerLeftStyle}>
							{currentStep > 1 ? (
								<Button
									type="text"
									icon={<LeftOutlined />}
									data-testid="onboarding-back-button"
									onClick={nav.prev}
								>
									上一步
								</Button>
							) : null}
						</div>
						<div style={headerCenterStyle}>
							<StepIndicator
								currentStep={currentStep}
								totalSteps={TOTAL_STEPS}
								reachedStep={nav.reachedStep}
								onStepClick={(step) => nav.jumpTo(step as 1 | 2 | 3 | 4)}
							/>
						</div>
						<div style={headerRightStyle}>
							<Button type="text" data-testid="onboarding-skip-button" onClick={handleSkip}>
								跳过引导
							</Button>
						</div>
					</header>

					<Title level={2} style={titleStyle}>
						{title}
					</Title>
					{description ? (
						<Paragraph type="secondary" style={descriptionStyle}>
							{description}
						</Paragraph>
					) : (
						<div style={{ marginBottom: 24 }} />
					)}

					<OnboardingPageTransition stepKey={currentStep} direction={direction}>
						{children}
					</OnboardingPageTransition>

					{footer ? (
						<motion.div
							key={shakeKey}
							data-testid="onboarding-footer-slot"
							style={footerStyle}
							animate={shakeKey > 0 ? { x: [0, -4, 4, -4, 0] } : undefined}
							transition={{ duration: 0.3 }}
						>
							{footer}
						</motion.div>
					) : null}
				</div>
			</div>
		</div>
	);
}

// Root wrapper covers the full viewport so the fixed OnboardingBackground
// paints against a known backdrop and the content column always has room to
// scroll. backgroundColor provides a fallback for browsers that fail to
// composite the background layer.
const rootStyle = {
	position: "relative" as const,
	minHeight: "100vh",
	backgroundColor: "#fafafa",
};

// Content sits at z-index 1 above the decorative background. padding matches
// the original static layout so step 11-14 page rewrites do not see a
// vertical rhythm shift.
const contentStyle = {
	position: "relative" as const,
	zIndex: 1,
	display: "flex",
	justifyContent: "center",
	padding: "32px 24px 48px",
};

const containerStyle = {
	width: "100%",
	maxWidth: 720,
};

const headerStyle = {
	display: "flex",
	alignItems: "center",
	gap: 12,
	marginBottom: 32,
};

const headerLeftStyle = {
	flex: "0 0 auto",
	minWidth: 96,
	display: "flex",
	justifyContent: "flex-start",
};

const headerCenterStyle = {
	flex: "1 1 auto",
	display: "flex",
	justifyContent: "center",
};

const headerRightStyle = {
	flex: "0 0 auto",
	minWidth: 96,
	display: "flex",
	justifyContent: "flex-end",
};

const titleStyle = {
	textAlign: "center" as const,
	marginBottom: 12,
};

const descriptionStyle = {
	textAlign: "center" as const,
	marginBottom: 32,
	fontSize: 15,
};

const footerStyle = {
	display: "flex",
	justifyContent: "flex-end",
	gap: 12,
	marginTop: 40,
};
