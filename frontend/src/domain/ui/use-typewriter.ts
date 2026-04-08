import { useEffect, useState } from "react";

// useTypewriter drives a looping typewriter state machine across a list of
// multi-line slogans. It types each slogan forward char-by-char, holds, deletes
// it backward, pauses, then advances to the next slogan and loops forever.
//
// The hook is a pure primitive: callers pass any string matrix and own the
// content. Do not add slogan selection, persistence, or business logic here.
//
// Reduced-motion policy: if the user has `prefers-reduced-motion: reduce`, the
// hook snaps to the first slogan at full length and stops there. No animation,
// no loop, no cursor recommendation (the caller decides what to do with
// showCursor=false).

// A slogan is a tuple of lines. Each line is rendered on its own visual row.
// Keep line count consistent across slogans so the container can reserve a
// stable min-height and avoid layout shift between slogans.
export type TypewriterSlogan = readonly string[];

export interface UseTypewriterOptions {
	/** Time per character typed forward, in ms. Default 80. */
	typeSpeedMs?: number;
	/** Time per character deleted backward, in ms. Default 40. */
	deleteSpeedMs?: number;
	/** How long to hold a fully-typed slogan before deleting, in ms. Default 1500. */
	holdMs?: number;
	/** Pause between finishing deletion and starting the next slogan, in ms. Default 600. */
	pauseMs?: number;
	/** Delay on first mount before typing the first slogan, in ms. Default 200. */
	startDelayMs?: number;
	/** Random +/- jitter applied to each tick, in ms. Default 20. Set 0 for deterministic tests. */
	jitterMs?: number;
	/** Extra hold after CJK punctuation marks, in ms. Default 180. */
	punctuationHoldMs?: number;
}

export interface UseTypewriterResult {
	/** Current visible text for each line of the active slogan. */
	displayed: readonly string[];
	/** Index of the line where the cursor should currently render. */
	cursorLine: number;
	/** Whether the hook is suppressing animation due to reduced-motion. */
	isReducedMotion: boolean;
}

type Phase =
	| { kind: "idle" }
	| { kind: "typing"; sloganIdx: number; charCount: number }
	| { kind: "holding"; sloganIdx: number }
	| { kind: "deleting"; sloganIdx: number; charCount: number }
	| { kind: "pausing"; nextSloganIdx: number };

// CJK punctuation marks that deserve an extra pause after typing, to make the
// rhythm feel like someone taking a breath. Keep this set small on purpose.
const PUNCTUATION_HOLD_CHARS = new Set(["。", "，", "·", "；", "！", "？", "、"]);

// Array.from splits a string into grapheme-safe single characters for CJK and
// common emoji surrogate pairs without requiring Intl.Segmenter, which is
// missing in some mobile WebViews.
function splitChars(line: string): readonly string[] {
	return Array.from(line);
}

function countSloganChars(slogan: TypewriterSlogan): number {
	let total = 0;
	for (const line of slogan) {
		total += splitChars(line).length;
	}
	return total;
}

// sliceSlogan returns the slogan truncated to the first `count` characters
// across all lines. Used for both typing (progressing) and deleting (regressing).
function sliceSlogan(slogan: TypewriterSlogan, count: number): readonly string[] {
	let remaining = Math.max(0, count);
	const out: string[] = [];
	for (const line of slogan) {
		const chars = splitChars(line);
		const take = Math.min(remaining, chars.length);
		out.push(chars.slice(0, take).join(""));
		remaining -= take;
	}
	return out;
}

// charAtOffset returns the character currently being typed at the given
// offset across all lines, so we can detect punctuation and apply an extra
// hold after it.
function charAtOffset(slogan: TypewriterSlogan, offset: number): string | undefined {
	let cursor = 0;
	for (const line of slogan) {
		const chars = splitChars(line);
		if (offset < cursor + chars.length) {
			return chars[offset - cursor];
		}
		cursor += chars.length;
	}
	return undefined;
}

// resolveReducedMotion reads prefers-reduced-motion once. We intentionally do
// not subscribe to change events: the OS setting rarely flips mid-session, and
// subscription adds complexity without proportional UX value.
function resolveReducedMotion(): boolean {
	if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
		return false;
	}
	return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

// findCursorLine walks lines from the end, returning the index of the last
// line that has any characters. If all lines are empty, the cursor sits on
// line 0.
function findCursorLine(displayed: readonly string[]): number {
	for (let i = displayed.length - 1; i >= 0; i--) {
		if ((displayed[i]?.length ?? 0) > 0) return i;
	}
	return 0;
}

export function useTypewriter(
	slogans: readonly TypewriterSlogan[],
	options: UseTypewriterOptions = {},
): UseTypewriterResult {
	const {
		typeSpeedMs = 80,
		deleteSpeedMs = 40,
		holdMs = 1500,
		pauseMs = 600,
		startDelayMs = 200,
		jitterMs = 20,
		punctuationHoldMs = 180,
	} = options;

	// Read reduced-motion once on mount. `useState` with an initializer fn
	// avoids calling matchMedia on every render.
	const [isReducedMotion] = useState<boolean>(resolveReducedMotion);
	const [phase, setPhase] = useState<Phase>({ kind: "idle" });

	// NOTE: slogans is expected to be a stable reference (module-level const
	// or useMemo'd). Passing a fresh array literal on every render restarts
	// the animation, which is almost never the desired behavior.
	useEffect(() => {
		if (slogans.length === 0) return;

		// Reduced-motion path: pin the first slogan fully typed and stop.
		if (isReducedMotion) {
			setPhase({ kind: "holding", sloganIdx: 0 });
			return () => undefined;
		}

		let cancelled = false;
		let timerId: number | undefined;

		const schedule = (ms: number, fn: () => void) => {
			if (cancelled) return;
			timerId = window.setTimeout(() => {
				timerId = undefined;
				if (!cancelled) fn();
			}, ms);
		};

		const applyJitter = (base: number): number => {
			if (jitterMs <= 0) return base;
			const offset = Math.round((Math.random() * 2 - 1) * jitterMs);
			return Math.max(10, base + offset);
		};

		const advance = (current: Phase) => {
			if (cancelled) return;
			setPhase(current);
			switch (current.kind) {
				case "idle":
					schedule(startDelayMs, () => advance({ kind: "typing", sloganIdx: 0, charCount: 0 }));
					return;

				case "typing": {
					const slogan = slogans[current.sloganIdx];
					if (!slogan) return;
					const total = countSloganChars(slogan);
					if (current.charCount >= total) {
						advance({ kind: "holding", sloganIdx: current.sloganIdx });
						return;
					}
					const nextChar = charAtOffset(slogan, current.charCount);
					const punctBonus =
						nextChar && PUNCTUATION_HOLD_CHARS.has(nextChar) ? punctuationHoldMs : 0;
					schedule(applyJitter(typeSpeedMs) + punctBonus, () =>
						advance({
							kind: "typing",
							sloganIdx: current.sloganIdx,
							charCount: current.charCount + 1,
						}),
					);
					return;
				}

				case "holding": {
					const slogan = slogans[current.sloganIdx];
					if (!slogan) return;
					const total = countSloganChars(slogan);
					schedule(holdMs, () =>
						advance({
							kind: "deleting",
							sloganIdx: current.sloganIdx,
							charCount: total,
						}),
					);
					return;
				}

				case "deleting": {
					if (current.charCount <= 0) {
						const nextIdx = (current.sloganIdx + 1) % slogans.length;
						advance({ kind: "pausing", nextSloganIdx: nextIdx });
						return;
					}
					schedule(applyJitter(deleteSpeedMs), () =>
						advance({
							kind: "deleting",
							sloganIdx: current.sloganIdx,
							charCount: current.charCount - 1,
						}),
					);
					return;
				}

				case "pausing":
					schedule(pauseMs, () =>
						advance({
							kind: "typing",
							sloganIdx: current.nextSloganIdx,
							charCount: 0,
						}),
					);
					return;
			}
		};

		advance({ kind: "idle" });

		return () => {
			cancelled = true;
			if (timerId !== undefined) {
				window.clearTimeout(timerId);
				timerId = undefined;
			}
		};
	}, [
		slogans,
		isReducedMotion,
		typeSpeedMs,
		deleteSpeedMs,
		holdMs,
		pauseMs,
		startDelayMs,
		jitterMs,
		punctuationHoldMs,
	]);

	if (slogans.length === 0) {
		return { displayed: [], cursorLine: 0, isReducedMotion };
	}

	const firstSlogan = slogans[0];
	if (!firstSlogan) {
		return { displayed: [], cursorLine: 0, isReducedMotion };
	}

	const emptyLines = (slogan: TypewriterSlogan): readonly string[] => slogan.map(() => "");

	switch (phase.kind) {
		case "idle": {
			const displayed = emptyLines(firstSlogan);
			return { displayed, cursorLine: 0, isReducedMotion };
		}
		case "typing": {
			const slogan = slogans[phase.sloganIdx] ?? firstSlogan;
			const displayed = sliceSlogan(slogan, phase.charCount);
			return { displayed, cursorLine: findCursorLine(displayed), isReducedMotion };
		}
		case "holding": {
			const slogan = slogans[phase.sloganIdx] ?? firstSlogan;
			return {
				displayed: slogan,
				cursorLine: slogan.length - 1,
				isReducedMotion,
			};
		}
		case "deleting": {
			const slogan = slogans[phase.sloganIdx] ?? firstSlogan;
			const displayed = sliceSlogan(slogan, phase.charCount);
			return { displayed, cursorLine: findCursorLine(displayed), isReducedMotion };
		}
		case "pausing": {
			const nextSlogan = slogans[phase.nextSloganIdx] ?? firstSlogan;
			return { displayed: emptyLines(nextSlogan), cursorLine: 0, isReducedMotion };
		}
	}
}
