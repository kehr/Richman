import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import type { CSSProperties, ReactNode } from "react";

interface OnboardingPageTransitionProps {
	stepKey: string | number;
	direction: "forward" | "backward";
	children: ReactNode;
}

// OnboardingPageTransition wraps the page content of each onboarding step and
// provides direction-aware page-swap transitions. When stepKey changes, the
// exiting page is animated out in one direction, and the entering page is
// animated in from the opposite direction. This creates the illusion of
// navigating a sequential flow.
//
// Two variant sets are computed dynamically based on direction:
//
//   1. forward: enter from right (x: 40), exit to left (x: -40). Used when
//      the user clicks next or presses ArrowRight.
//   2. backward: enter from left (x: -40), exit to right (x: 40). Used when
//      the user clicks back or presses ArrowLeft.
//
// If the user has enabled reduced-motion (via system or browser preference),
// both directions collapse to opacity-only animation, with no x movement.
// This respects accessibility best practices while keeping visual feedback.
//
// The motion.div is keyed on stepKey, so AnimatePresence mode="wait" will
// animate the exit of the old key before the enter of the new one. This
// avoids the two pages rendering at the same time.
//
// Transition timing is set to 0.35s easeOut to feel snappy but not jarring.
export function OnboardingPageTransition({
	stepKey,
	direction,
	children,
}: OnboardingPageTransitionProps) {
	const reducedMotion = useReducedMotion();
	const variants = reducedMotion
		? PAGE_TRANSITION_VARIANTS.reduced
		: PAGE_TRANSITION_VARIANTS[direction];

	return (
		<AnimatePresence mode="wait">
			<motion.div
				key={stepKey}
				initial={variants.initial}
				animate={variants.animate}
				exit={variants.exit}
				transition={{ duration: 0.35, ease: "easeOut" }}
				style={{ width: "100%" }}
			>
				{children}
			</motion.div>
		</AnimatePresence>
	);
}

// PAGE_TRANSITION_VARIANTS defines the three variant sets used by
// OnboardingPageTransition. Each set has initial, animate, and exit states.
//
// Tests can import and assert on these variants without poking into the
// component internals.
export const PAGE_TRANSITION_VARIANTS = {
	forward: {
		initial: { x: 40, opacity: 0 },
		animate: { x: 0, opacity: 1 },
		exit: { x: -40, opacity: 0 },
	},
	backward: {
		initial: { x: -40, opacity: 0 },
		animate: { x: 0, opacity: 1 },
		exit: { x: 40, opacity: 0 },
	},
	reduced: {
		initial: { opacity: 0 },
		animate: { opacity: 1 },
		exit: { opacity: 0 },
	},
} as const;
