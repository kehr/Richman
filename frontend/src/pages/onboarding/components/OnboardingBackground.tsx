import { motion, useReducedMotion } from "framer-motion";
import type { CSSProperties } from "react";

interface OnboardingBackgroundProps {
	currentStep: 1 | 2 | 3 | 4;
}

// OnboardingBackground is the decorative, viewport-fixed backdrop shared by
// every onboarding step (PRD §3.4, TRD §5.2). It paints three layers behind
// the page content:
//
//   1. A static 64px grid drawn with doubled linear-gradients. This layer
//      never animates, so it does not need the reduced-motion branch.
//   2. A slow radial glow that drifts its background-position over a
//      90-second loop to create a subtle "breathing" feel without pulling
//      the eye away from the content.
//   3. A 120x120 conic-gradient ring with the centered Richman logo, shown
//      only on Welcome (currentStep === 1). The ring rotates on a 30-second
//      linear loop; the logo stays static in the middle.
//
// The whole component is a fixed-position overlay at z-index 0 with
// pointer-events: none, so page content layered at z-index 1 always paints
// on top and never loses clicks to the background. Decorative imagery
// (logo + hero) is marked aria-hidden so screen readers stay focused on the
// real page copy.
//
// Reduced-motion handling follows the framer-motion convention:
// useReducedMotion() returns true when the user has opted in, false when
// they have not, and null when no preference has been detected yet. We only
// disable animations when it returns true; null is treated as "proceed with
// full motion" to match framer-motion's own default.
//
// The conic-gradient + radial-gradient mask combination has been known to
// upgrade the ring into its own GPU layer unpredictably on Safari/iPadOS
// (see PRD Appendix D, Pre-mortem bug 3). We pin `will-change: transform`
// only on the ring element itself so the hint stays scoped and the other
// layers do not end up with unexpected compositor promotions.
export function OnboardingBackground({ currentStep }: OnboardingBackgroundProps) {
	const reducedMotion = useReducedMotion();

	return (
		<div aria-hidden="true" data-testid="onboarding-background" style={containerStyle}>
			<div style={gridStyle} />
			<motion.div
				style={glowStyle}
				animate={
					reducedMotion
						? undefined
						: {
								backgroundPosition: ["50% 50%", "55% 45%", "45% 55%", "50% 50%"],
							}
				}
				transition={{ duration: 90, repeat: Number.POSITIVE_INFINITY, ease: "easeInOut" }}
			/>
			{currentStep === 1 && (
				<div style={ringWrapperStyle}>
					<motion.div
						style={ringStyle}
						animate={reducedMotion ? undefined : { rotate: 360 }}
						transition={{ duration: 30, repeat: Number.POSITIVE_INFINITY, ease: "linear" }}
					/>
					<img src="/logo.svg" alt="" aria-hidden="true" style={ringLogoStyle} />
				</div>
			)}
		</div>
	);
}

// Fixed overlay covering the full viewport. z-index 0 keeps it behind the
// page content (which sits at z-index 1 inside OnboardingLayout), and
// pointer-events: none makes sure the decoration never swallows clicks.
const containerStyle: CSSProperties = {
	position: "fixed",
	inset: 0,
	zIndex: 0,
	pointerEvents: "none",
	overflow: "hidden",
};

// Layer 1: static 64px grid drawn with two linear-gradients (one horizontal,
// one vertical). Pure CSS, no animation.
const gridStyle: CSSProperties = {
	position: "absolute",
	inset: 0,
	backgroundImage:
		"linear-gradient(to right, #0000000a 1px, transparent 1px), " +
		"linear-gradient(to bottom, #0000000a 1px, transparent 1px)",
	backgroundSize: "64px 64px",
};

// Layer 2: slow radial glow. The 200% background-size gives the animation
// headroom to drift its background-position without revealing hard edges.
const glowStyle: CSSProperties = {
	position: "absolute",
	inset: 0,
	backgroundImage: "radial-gradient(circle at 50% 50%, #00000008, transparent 60%)",
	backgroundSize: "200% 200%",
};

// Layer 3a: ring wrapper. Positioned near the top third of the viewport and
// horizontally centered. Sized to match the ring so the logo can be absolute-
// centered inside it.
const ringWrapperStyle: CSSProperties = {
	position: "absolute",
	top: "12vh",
	left: "50%",
	transform: "translateX(-50%)",
	display: "flex",
	alignItems: "center",
	justifyContent: "center",
	width: 120,
	height: 120,
};

// Layer 3b: the ring itself. A conic-gradient supplies the glowing arc, and
// a radial-gradient mask carves out the inner 58px so only a 2px stroke
// remains. `will-change: transform` is scoped to this element to keep the
// GPU layer hint from leaking onto neighbouring layers.
const ringStyle: CSSProperties = {
	width: 120,
	height: 120,
	borderRadius: "50%",
	background: "conic-gradient(from 0deg, #000 0deg, transparent 120deg, transparent 360deg)",
	mask: "radial-gradient(circle, transparent 58px, #000 60px)",
	WebkitMask: "radial-gradient(circle, transparent 58px, #000 60px)",
	willChange: "transform",
};

// Layer 3c: the centered logo. Stays static — only the ring rotates.
const ringLogoStyle: CSSProperties = {
	position: "absolute",
	width: 56,
	height: 56,
	userSelect: "none",
};
