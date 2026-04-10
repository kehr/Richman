import type { Frequency, MarketScheduleDTO, WindowDTO } from "@/features/schedule";
import { Card, Flex, Select, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { WindowToggleRow } from "./WindowToggleRow";

// DEFAULT_TIMES holds the server-side defaults for each market window so the
// UI can reset a customised time back to its original value.
// All times are in Asia/Shanghai timezone, matching backend defaults.go.
const DEFAULT_TIMES: Record<"a_share" | "us_stock", { pre: string; post: string }> = {
	a_share: { pre: "08:30", post: "15:05" },
	// US pre = NYSE 09:30 EDT - 1h buffer = 08:30 EDT = 20:30 CST
	// US post = NYSE 16:00 EDT + 5min = 16:05 EDT = 04:05 CST (next day)
	us_stock: { pre: "20:30", post: "04:05" },
};

// TIME_RANGES constrains the TimePicker to a sensible hour band per window.
// All ranges are in Asia/Shanghai timezone (TRD spec).
// US post ["04:00", "08:00"] spans midnight — buildDisabledHours handles this case.
const TIME_RANGES: Record<
	"a_share" | "us_stock",
	{ pre: [string, string]; post: [string, string] }
> = {
	a_share: { pre: ["07:00", "09:29"], post: ["15:00", "20:00"] },
	us_stock: { pre: ["20:00", "23:00"], post: ["04:00", "08:00"] },
};

// FREQUENCY_OPTIONS lists all per-market frequency override choices. null means
// "follow the global setting".
const FREQUENCY_OPTIONS: Array<{ value: Frequency; labelKey: string }> = [
	{ value: null, labelKey: "schedule.globalFrequency.followGlobal" },
	{ value: "every_window", labelKey: "schedule.globalFrequency.every_window" },
	{ value: "daily", labelKey: "schedule.globalFrequency.daily" },
	{ value: "every_2_days", labelKey: "schedule.globalFrequency.every_2_days" },
	{ value: "every_3_days", labelKey: "schedule.globalFrequency.every_3_days" },
	{ value: "weekly", labelKey: "schedule.globalFrequency.weekly" },
	{ value: "custom", labelKey: "schedule.globalFrequency.custom" },
];

interface MarketWindowCardProps {
	market: "a_share" | "us_stock";
	settings: MarketScheduleDTO;
	onUpdate: (updated: Partial<MarketScheduleDTO>) => void;
}

// MarketWindowCard shows a single market's frequency override and two window
// toggle rows (pre-market, post-market). Changes are propagated to the parent
// via onUpdate so the parent can manage a single draft ScheduleSettingsDTO.
export function MarketWindowCard({ market, settings, onUpdate }: MarketWindowCardProps) {
	const { t } = useTranslation("settings");
	const defaults = DEFAULT_TIMES[market];
	const ranges = TIME_RANGES[market];

	const handleFrequencyChange = (value: Frequency) => {
		onUpdate({ frequency: value });
	};

	const handleWindowUpdate = (window: "preWindow" | "postWindow", patch: Partial<WindowDTO>) => {
		const current = settings[window];
		onUpdate({ [window]: { ...current, ...patch, isCustom: true } });
	};

	const handleWindowReset = (window: "preWindow" | "postWindow", defaultTime: string) => {
		const current = settings[window];
		onUpdate({
			[window]: { ...current, time: defaultTime, isCustom: false },
		});
	};

	const frequencySelectOptions = FREQUENCY_OPTIONS.map(({ value, labelKey }) => ({
		// labelKey is a runtime-computed string; bypass TFunction's literal-union
		// key constraint with a double cast since the keys are validated at build
		// time by the i18n:check script rather than tsc.
		// biome-ignore lint/suspicious/noExplicitAny: dynamic i18n key bypass
		label: t(labelKey as any),
		// Select requires a string or number value; use empty string for null.
		value: value === null ? "__follow__" : value,
	}));

	const frequencySelectValue = settings.frequency === null ? "__follow__" : settings.frequency;

	const handleSelectChange = (val: string) => {
		handleFrequencyChange(val === "__follow__" ? null : (val as Frequency));
	};

	return (
		<Card size="small" style={{ marginBottom: 12 }}>
			<Flex vertical gap={16}>
				{/* Market title */}
				<Flex justify="space-between" align="center">
					<Typography.Text strong>
						{/* biome-ignore lint/suspicious/noExplicitAny: dynamic i18n key bypass */}
						{t(`schedule.markets.${market}` as any)}
					</Typography.Text>
				</Flex>

				{/* Per-market frequency override */}
				<Flex align="center" gap={12}>
					<Typography.Text type="secondary" style={{ fontSize: 13, whiteSpace: "nowrap" }}>
						{t("schedule.markets.frequency")}
					</Typography.Text>
					<Select
						value={frequencySelectValue}
						onChange={handleSelectChange}
						options={frequencySelectOptions}
						size="small"
						style={{ width: 160 }}
					/>
				</Flex>

				{/* Window rows */}
				<Flex vertical gap={12}>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{t("schedule.markets.windows")}
					</Typography.Text>
					<WindowToggleRow
						enabled={settings.preWindow.enabled}
						time={settings.preWindow.time}
						isCustom={settings.preWindow.isCustom}
						label={t("schedule.window.pre")}
						hint={t("schedule.window.preHint")}
						onToggle={(v) => handleWindowUpdate("preWindow", { enabled: v })}
						onTimeChange={(time) => handleWindowUpdate("preWindow", { time })}
						onReset={() => handleWindowReset("preWindow", defaults.pre)}
						timeRange={ranges.pre}
					/>
					<WindowToggleRow
						enabled={settings.postWindow.enabled}
						time={settings.postWindow.time}
						isCustom={settings.postWindow.isCustom}
						label={t("schedule.window.post")}
						hint={t("schedule.window.postHint")}
						onToggle={(v) => handleWindowUpdate("postWindow", { enabled: v })}
						onTimeChange={(time) => handleWindowUpdate("postWindow", { time })}
						onReset={() => handleWindowReset("postWindow", defaults.post)}
						timeRange={ranges.post}
					/>
				</Flex>
			</Flex>
		</Card>
	);
}
