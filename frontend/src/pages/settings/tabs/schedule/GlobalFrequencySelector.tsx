import type { Frequency } from "@/features/schedule";
import { Flex, InputNumber, Radio, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

type NonNullFrequency = Exclude<Frequency, null>;

// FREQUENCY_LIST is the ordered set of non-null frequency values shown as
// Radio options. "custom" always appears last.
const FREQUENCY_LIST: NonNullFrequency[] = [
	"every_window",
	"daily",
	"every_2_days",
	"every_3_days",
	"weekly",
	"custom",
];

interface GlobalFrequencySelectorProps {
	value: NonNullFrequency;
	customDays: number | null | undefined;
	onChange: (freq: NonNullFrequency, days?: number) => void;
}

// GlobalFrequencySelector renders the six global-frequency choices as a plain
// Radio.Group (no button style per project convention). When "custom" is
// selected an InputNumber appears so the user can specify the exact interval.
export function GlobalFrequencySelector({
	value,
	customDays,
	onChange,
}: GlobalFrequencySelectorProps) {
	const { t } = useTranslation("settings");

	const handleRadioChange = (freq: NonNullFrequency) => {
		if (freq === "custom") {
			onChange(freq, customDays ?? 7);
		} else {
			onChange(freq);
		}
	};

	const handleDaysChange = (days: number | null) => {
		if (days != null) {
			onChange("custom", days);
		}
	};

	return (
		<Flex align="center" gap={12} wrap>
			<Typography.Text type="secondary" style={{ fontSize: 13, whiteSpace: "nowrap" }}>
				{t("schedule.globalFrequency.label")}
			</Typography.Text>
			<Radio.Group
				value={value}
				onChange={(e) => handleRadioChange(e.target.value as NonNullFrequency)}
			>
				<Flex gap={16} wrap>
					{FREQUENCY_LIST.map((freq) => (
						<Radio key={freq} value={freq}>
							{t(`schedule.globalFrequency.${freq}`)}
						</Radio>
					))}
				</Flex>
			</Radio.Group>
			{value === "custom" && (
				<InputNumber
					min={1}
					max={30}
					value={customDays ?? 7}
					onChange={handleDaysChange}
					size="small"
					style={{ width: 100 }}
					addonAfter={t("schedule.globalFrequency.customPlaceholder")}
				/>
			)}
		</Flex>
	);
}
