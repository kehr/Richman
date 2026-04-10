import { Button, Flex, Switch, TimePicker, Tooltip, Typography } from "@/ui-kit/eat";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
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

	const timeValue = time ? dayjs(`2000-01-01 ${time}`, "YYYY-MM-DD HH:mm") : null;

	const handleTimeChange = (value: Dayjs | null) => {
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
