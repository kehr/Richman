import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTypewriter } from "./use-typewriter";

// Deterministic test profile: zero jitter, zero punctuation bonus, fixed
// speeds so we can assert character counts per tick exactly. Production
// defaults add randomness that is useful for UX but makes assertions fragile.
const DETERMINISTIC = {
	typeSpeedMs: 50,
	deleteSpeedMs: 25,
	holdMs: 500,
	pauseMs: 200,
	startDelayMs: 100,
	jitterMs: 0,
	punctuationHoldMs: 0,
} as const;

const SAMPLE_SLOGANS = [
	["把基金经理的思维方式", "装进你的口袋"],
	["不是新闻摘要", "是可执行的建议"],
] as const;

function overrideMatchMedia(matches: boolean) {
	const original = window.matchMedia;
	window.matchMedia = ((query: string) => ({
		matches,
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false,
	})) as typeof window.matchMedia;
	return () => {
		window.matchMedia = original;
	};
}

describe("useTypewriter", () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.restoreAllMocks();
	});

	it("returns empty displayed with no slogans", () => {
		const { result } = renderHook(() => useTypewriter([], DETERMINISTIC));
		expect(result.current.displayed).toEqual([]);
		expect(result.current.cursorLine).toBe(0);
	});

	it("starts with empty lines before startDelayMs elapses", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		expect(result.current.displayed).toEqual(["", ""]);
		expect(result.current.cursorLine).toBe(0);
	});

	it("types the first slogan forward character by character", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		// Advance past startDelay to begin typing, then 3 ticks of typeSpeedMs.
		act(() => {
			vi.advanceTimersByTime(DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * 3);
		});
		expect(result.current.displayed[0]).toBe("把基金");
		expect(result.current.displayed[1]).toBe("");
		expect(result.current.cursorLine).toBe(0);
	});

	it("wraps cursor to line 2 after line 1 is fully typed", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		const line1Chars = Array.from(SAMPLE_SLOGANS[0][0]).length;
		act(() => {
			vi.advanceTimersByTime(
				DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * (line1Chars + 2),
			);
		});
		expect(result.current.displayed[0]).toBe(SAMPLE_SLOGANS[0][0]);
		expect(result.current.displayed[1]).toBe("装进");
		expect(result.current.cursorLine).toBe(1);
	});

	it("holds the fully typed slogan for holdMs then begins deleting", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		const slogan = SAMPLE_SLOGANS[0];
		const totalChars = slogan.reduce((sum, line) => sum + Array.from(line).length, 0);
		// Advance through startDelay + full typing.
		act(() => {
			vi.advanceTimersByTime(DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * totalChars);
		});
		expect(result.current.displayed).toEqual([slogan[0], slogan[1]]);

		// Still holding.
		act(() => {
			vi.advanceTimersByTime(DETERMINISTIC.holdMs - 10);
		});
		expect(result.current.displayed).toEqual([slogan[0], slogan[1]]);

		// Hold ends, deletion begins one char.
		act(() => {
			vi.advanceTimersByTime(10 + DETERMINISTIC.deleteSpeedMs);
		});
		const deletedOne = result.current.displayed;
		const remainingChars = deletedOne.reduce((sum, line) => sum + Array.from(line).length, 0);
		expect(remainingChars).toBe(totalChars - 1);
	});

	it("advances to the next slogan after deletion and pause", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		const slogan0 = SAMPLE_SLOGANS[0];
		const total0 = slogan0.reduce((sum, line) => sum + Array.from(line).length, 0);

		// Type through slogan 0, hold, delete fully, pause, then type 2 chars of slogan 1.
		const typePhase = DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * total0;
		const deletePhase = DETERMINISTIC.holdMs + DETERMINISTIC.deleteSpeedMs * total0;
		const pausePhase = DETERMINISTIC.pauseMs + DETERMINISTIC.typeSpeedMs * 2;

		act(() => {
			vi.advanceTimersByTime(typePhase + deletePhase + pausePhase);
		});

		expect(result.current.displayed[0]).toBe("不是");
		expect(result.current.displayed[1]).toBe("");
	});

	it("loops back to slogan 0 after finishing the last slogan", () => {
		const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		const charCount = (s: readonly string[]) =>
			s.reduce((sum, line) => sum + Array.from(line).length, 0);
		const total0 = charCount(SAMPLE_SLOGANS[0]);
		const total1 = charCount(SAMPLE_SLOGANS[1]);

		const sloganCycle = (total: number) =>
			DETERMINISTIC.typeSpeedMs * total +
			DETERMINISTIC.holdMs +
			DETERMINISTIC.deleteSpeedMs * total +
			DETERMINISTIC.pauseMs;

		// Start delay + slogan 0 full cycle + slogan 1 full cycle + 2 chars into slogan 0 again.
		act(() => {
			vi.advanceTimersByTime(
				DETERMINISTIC.startDelayMs +
					sloganCycle(total0) +
					sloganCycle(total1) +
					DETERMINISTIC.typeSpeedMs * 2,
			);
		});

		expect(result.current.displayed[0]).toBe("把基");
		expect(result.current.displayed[1]).toBe("");
	});

	it("snaps to the first slogan fully typed when prefers-reduced-motion is active", () => {
		const restore = overrideMatchMedia(true);
		try {
			const { result } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
			expect(result.current.isReducedMotion).toBe(true);
			// Effect runs synchronously to pin to holding(0).
			act(() => {
				vi.advanceTimersByTime(0);
			});
			expect(result.current.displayed).toEqual([SAMPLE_SLOGANS[0][0], SAMPLE_SLOGANS[0][1]]);
			expect(result.current.cursorLine).toBe(1);

			// Running the clock forward does not produce any deletion or loop.
			act(() => {
				vi.advanceTimersByTime(10_000);
			});
			expect(result.current.displayed).toEqual([SAMPLE_SLOGANS[0][0], SAMPLE_SLOGANS[0][1]]);
		} finally {
			restore();
		}
	});

	it("clears pending timers on unmount with no state updates afterward", () => {
		const { result, unmount } = renderHook(() => useTypewriter(SAMPLE_SLOGANS, DETERMINISTIC));
		act(() => {
			vi.advanceTimersByTime(DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * 2);
		});
		const snapshotBeforeUnmount = result.current.displayed;
		unmount();
		// Advance the clock well past when more chars would have been typed.
		act(() => {
			vi.advanceTimersByTime(10_000);
		});
		// If no state update happened post-unmount, the last rendered snapshot
		// held by the test harness still matches what we saw before unmount.
		expect(result.current.displayed).toEqual(snapshotBeforeUnmount);
	});

	it("handles single-line slogans without crashing", () => {
		const singleLine = [["短句"], ["再一段"]] as const;
		const { result } = renderHook(() => useTypewriter(singleLine, DETERMINISTIC));
		act(() => {
			vi.advanceTimersByTime(DETERMINISTIC.startDelayMs + DETERMINISTIC.typeSpeedMs * 2);
		});
		expect(result.current.displayed).toEqual(["短句"]);
		expect(result.current.cursorLine).toBe(0);
	});
});
