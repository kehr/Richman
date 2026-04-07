// Analysis schedule helpers shared by Dashboard and Decision Card Detail.
//
// Mirrors backend/internal/service/analysis/scheduler.go:
//   AM brief  : 08:30 Asia/Shanghai, Mon-Fri
//   PM digest : 15:30 Asia/Shanghai, Mon-Fri
//   US digest : 06:00 Asia/Shanghai, Tue-Sat
//
// When the backend grows an explicit "next analysis" endpoint these helpers
// should switch to that response. Until then we compute the next slot on
// the client so the UI always has a value to show. Single source of truth
// here prevents Dashboard and Detail page schedules drifting apart.

const SHANGHAI_TZ = "Asia/Shanghai";

interface ScheduleSlot {
	hour: number;
	minute: number;
	weekdays: number[]; // 0 = Sunday
}

const ANALYSIS_SCHEDULE: ScheduleSlot[] = [
	{ hour: 6, minute: 0, weekdays: [2, 3, 4, 5, 6] }, // US digest
	{ hour: 8, minute: 30, weekdays: [1, 2, 3, 4, 5] }, // A-share AM
	{ hour: 15, minute: 30, weekdays: [1, 2, 3, 4, 5] }, // A-share PM
];

// shanghaiPartsFromInstant projects a UTC instant into Shanghai wall-clock
// parts (year/month/day/hour/minute/weekday). This avoids the trap where
// Date#getDay/getHours read the host machine's local clock.
function shanghaiPartsFromInstant(instant: Date): {
	year: number;
	month: number;
	day: number;
	hour: number;
	minute: number;
	weekday: number;
} {
	const fmt = new Intl.DateTimeFormat("en-US", {
		timeZone: SHANGHAI_TZ,
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		weekday: "short",
		hour12: false,
	});
	const parts = fmt.formatToParts(instant);
	const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "0";
	const weekdayMap: Record<string, number> = {
		Sun: 0,
		Mon: 1,
		Tue: 2,
		Wed: 3,
		Thu: 4,
		Fri: 5,
		Sat: 6,
	};
	return {
		year: Number.parseInt(get("year"), 10),
		month: Number.parseInt(get("month"), 10),
		day: Number.parseInt(get("day"), 10),
		// Intl can return "24" for midnight in the en-US 24h locale; normalize to 0.
		hour: Number.parseInt(get("hour"), 10) % 24,
		minute: Number.parseInt(get("minute"), 10),
		weekday: weekdayMap[get("weekday")] ?? 0,
	};
}

// computeNextAnalysisTime returns the next scheduled analysis slot strictly
// after `now`, computed against Shanghai wall-clock time so the result is
// correct regardless of where the user's browser is running. The return
// value is a UTC Date instance representing that exact instant.
export function computeNextAnalysisTime(now: Date): Date | null {
	const start = shanghaiPartsFromInstant(now);
	const startMinutes = start.hour * 60 + start.minute;
	for (let offset = 0; offset < 7; offset += 1) {
		const weekday = (start.weekday + offset) % 7;
		for (const slot of ANALYSIS_SCHEDULE) {
			if (!slot.weekdays.includes(weekday)) continue;
			const slotMinutes = slot.hour * 60 + slot.minute;
			if (offset === 0 && slotMinutes <= startMinutes) continue;
			// Build the candidate by adding `offset` days to today's Shanghai
			// date and then setting the slot's hour/minute. We construct the
			// instant via Date.UTC + the Shanghai offset of 8 hours so the
			// returned Date is the correct UTC moment.
			const baseUtcMs = Date.UTC(start.year, start.month - 1, start.day) + offset * 86400000;
			const candidateUtcMs = baseUtcMs + (slot.hour - 8) * 3600000 + slot.minute * 60000;
			return new Date(candidateUtcMs);
		}
	}
	return null;
}

// formatHm renders a Date as "HH:MM" in Shanghai wall-clock 24h. We use
// Intl rather than getHours/getMinutes so users outside Asia/Shanghai still
// see the schedule's local time, matching the rest of the product.
export function formatHm(date: Date | null | undefined): string {
	if (!date) return "--:--";
	const fmt = new Intl.DateTimeFormat("en-GB", {
		timeZone: SHANGHAI_TZ,
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
	});
	return fmt.format(date);
}
