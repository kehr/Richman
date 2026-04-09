import { motion, useReducedMotion } from "framer-motion";
import type { CSSProperties } from "react";

// OnboardingBackground is the decorative, viewport-fixed backdrop shared by
// every onboarding step (PRD §3.4, TRD §5.2). It paints two layers behind
// the page content:
//
//   1. A static 64px grid drawn with doubled linear-gradients. This layer
//      never animates, so it does not need the reduced-motion branch.
//   2. A slow radial glow that drifts its background-position over a
//      90-second loop to create a subtle "breathing" feel without pulling
//      the eye away from the content.
//
// An earlier revision also rendered a Welcome-only rotating ring hero in
// the top third of the viewport. It was removed because the fixed
// `top: 12vh` placement collided visually with the page title and
// description area on normal laptop viewports, producing a ghosted "R"
// glyph overlapping the intro copy. If a future iteration wants the brand
// mark back, it should be inlined into the Welcome page content flow (as
// its own stagger child) so it cannot overlap text again.
//
// The whole component is a fixed-position overlay at z-index 0 with
// pointer-events: none, so page content layered at z-index 1 always paints
// on top and never loses clicks to the background. Decorative imagery is
// marked aria-hidden so screen readers stay focused on the real page copy.
//
// Reduced-motion handling follows the framer-motion convention:
// useReducedMotion() returns true when the user has opted in, false when
// they have not, and null when no preference has been detected yet. We only
// disable animations when it returns true; null is treated as "proceed with
// full motion" to match framer-motion's own default.
export function OnboardingBackground() {
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
