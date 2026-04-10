import prettyMilliseconds from "pretty-ms";

// formatDuration converts a millisecond count to a tiered human-readable
// string using separate milliseconds so the unit is always explicit:
//   < 1s  → "483ms"
//   < 1m  → "3s 810ms"
//   >= 1m → "1m 5s"  (ms dropped at minute scale — too granular)
export function formatDuration(ms: number): string {
	if (ms < 1000) {
		return prettyMilliseconds(ms);
	}
	if (ms < 60_000) {
		return prettyMilliseconds(ms, { separateMilliseconds: true });
	}
	return prettyMilliseconds(ms, { unitCount: 2 });
}
