import type { DayjsLike } from "@/domain/datetime/dayjs-like";
import { Button, Flex, Switch, TimePicker, Tooltip, Typography } from "@/ui-kit/eat";
import { RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";

interface WindowToggleRowProps {
	enabled: boolean;
	time: string;
	isCustom: boolean;
	label: string;
	hint: string;
	onToggle: (v: boolean) => void;
	onTimeChange: (t: string) => void;
	onReset: () => void;
	disabled?: boolean;
	timeRange: [string, string];
}

// parseHhmm creates a DayjsLike-compatible object from a "HH:mm" string so
// the TimePicker can be controlled without importing dayjs directly. antd
// bundles dayjs transitively but it is not a declared frontend dependency;
// using a structural alias avoids the tsc "cannot find module" error.
function parseHhmm(hhmm: string): DayjsLike {
	const [h, m] = hhmm.split(":").map(Number);
	const d = new Date(2000, 0, 1, h, m, 0, 0);
	return {
		toDate: () => d,
		format(fmt: string) {
			const hh = String(d.getHours()).padStart(2, "0");
			const mm = String(d.getMinutes()).padStart(2, "0");
			return fmt.replace("HH", hh).replace("mm", mm);
		},
		hour: () => d.getHours(),
		minute: () => d.getMinutes(),
	};
}

// WindowToggleRow renders a single market window row: an enable/disable toggle,
// a label with its hint, an optional "modified" badge, and a TimePicker button.
// When the row is toggled off, the time picker is inert. When isCustom is true,
// a blue "modified" indicator and a reset button appear.
export function WindowToggleRow({
	enabled,
	time,
	isCustom,
	label,
	hint,
	onToggle,
	onTimeChange,
	onReset,
	disabled = false,
	timeRange,
}: WindowToggleRowProps) {
	const { t } = useTranslation("settings");

	// antd TimePicker accepts any Dayjs-shaped value; our structural DayjsLike
	// satisfies that contract at runtime without importing the dayjs package.
	// biome-ignore lint/suspicious/noExplicitAny: structural compatibility cast
	const timeValue = time ? (parseHhmm(time) as any) : null;

	const handleTimeChange = (value: DayjsLike | null) => {
		if (value) {
			onTimeChange(value.format("HH:mm"));
		}
	};

	// Derive disabled hours from timeRange to constrain the picker boundaries.
	const [rangeStart, rangeEnd] = timeRange;
	const startHour = rangeStart ? Number.parseInt(rangeStart.split(":")[0], 10) : 0;
	const endHour = rangeEnd ? Number.parseInt(rangeEnd.split(":")[0], 10) : 23;

	const buildDisabledHours = () => {
		const hours: number[] = [];
		if (startHour <= endHour) {
			// Normal range (e.g. 07:00-09:29): disable hours outside [start, end].
			for (let h = 0; h < startHour; h++) hours.push(h);
			for (let h = endHour + 1; h < 24; h++) hours.push(h);
		} else {
			// Cross-midnight range (e.g. 04:00-08:00 where startHour=4, endHour=8
			// but the window actually means 04:xx of the next calendar day, stored
			// as a clock time). Disable hours NOT in [startHour, 23] ∪ [0, endHour].
			// In practice this means only hours in (endHour, startHour) are disabled.
			for (let h = endHour + 1; h < startHour; h++) hours.push(h);
		}
		return hours;
	};

	return (
		<Flex
			align="center"
			gap={12}
			style={{ opacity: !enabled || disabled ? 0.45 : 1, transition: "opacity 0.2s" }}
		>
			<Switch checked={enabled} onChange={onToggle} size="small" disabled={disabled} />

			<Flex vertical gap={2} style={{ flex: 1, minWidth: 0 }}>
				<Flex align="center" gap={6}>
					<Typography.Text strong style={{ fontSize: 13 }}>
						{label}
					</Typography.Text>
					{isCustom && (
						<Typography.Text
							style={{
								fontSize: 11,
								color: "#1677ff",
								background: "#e6f4ff",
								borderRadius: 3,
								padding: "0 5px",
								lineHeight: "18px",
							}}
						>
							{t("schedule.window.customLabel")}
						</Typography.Text>
					)}
				</Flex>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{hint}
				</Typography.Text>
			</Flex>

			<Flex align="center" gap={4}>
				<TimePicker
					value={timeValue}
					// biome-ignore lint/suspicious/noExplicitAny: DayjsLike cast above
					onChange={handleTimeChange as any}
					format="HH:mm"
					minuteStep={5}
					disabled={!enabled}
					allowClear={false}
					disabledTime={() => ({ disabledHours: buildDisabledHours })}
					size="small"
					style={{ width: 90 }}
					inputReadOnly
				/>
				{isCustom && (
					<Tooltip title={t("schedule.window.resetTooltip")}>
						<Button
							type="text"
							size="small"
							icon={<RotateCcw size={14} />}
							onClick={onReset}
							disabled={!enabled}
						/>
					</Tooltip>
				)}
			</Flex>
		</Flex>
	);
}
