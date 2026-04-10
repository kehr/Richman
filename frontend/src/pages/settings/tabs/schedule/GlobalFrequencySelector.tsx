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
		<Flex vertical gap={12}>
			<Typography.Text type="secondary" style={{ fontSize: 13 }}>
				{t("schedule.globalFrequency.label")}
			</Typography.Text>
			<Radio.Group
				value={value}
				onChange={(e) => handleRadioChange(e.target.value as NonNullFrequency)}
			>
				<Flex vertical gap={10}>
					{FREQUENCY_LIST.map((freq) => (
						<Flex key={freq} align="center" gap={12}>
							<Radio value={freq}>
								<Flex vertical gap={0}>
									<Typography.Text style={{ fontSize: 13 }}>
										{t(`schedule.globalFrequency.${freq}`)}
									</Typography.Text>
									<Typography.Text type="secondary" style={{ fontSize: 12 }}>
										{t(`schedule.globalFrequency.${freq}Hint`)}
									</Typography.Text>
								</Flex>
							</Radio>
							{freq === "custom" && value === "custom" && (
								<InputNumber
									min={1}
									max={30}
									value={customDays ?? 7}
									onChange={handleDaysChange}
									size="small"
									style={{ width: 80 }}
									addonAfter={t("schedule.globalFrequency.customPlaceholder")}
								/>
							)}
						</Flex>
					))}
				</Flex>
			</Radio.Group>
		</Flex>
	);
}
