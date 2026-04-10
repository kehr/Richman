import {
	type Frequency,
	type UpdateHoldingScheduleInput,
	type WindowPreference,
	useHoldingSchedule,
	useUpdateHoldingSchedule,
} from "@/features/schedule";
import { Select, Space, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

// Frequency option value used to represent "inherit from market/global".
// The backend accepts null for this case; "__follow__" is only a UI-level
// sentinel that we translate back to null before sending the mutation.
const FOLLOW_SENTINEL = "__follow__";

interface HoldingScheduleSectionProps {
	holdingId: number;
}

// frequencyToSelectValue converts a nullable Frequency (backend) to the
// Select value string. null is represented as the FOLLOW_SENTINEL UI sentinel.
function frequencyToSelectValue(f: Frequency): string {
	return f ?? FOLLOW_SENTINEL;
}

// windowToSelectValue converts a nullable WindowPreference (backend) to the
// Select value string. null is represented as the FOLLOW_SENTINEL UI sentinel.
function windowToSelectValue(w: WindowPreference): string {
	return w ?? FOLLOW_SENTINEL;
}

// selectValueToFrequency converts a Select value string back to the nullable
// Frequency type accepted by the backend.
function selectValueToFrequency(v: string): Frequency {
	return v === FOLLOW_SENTINEL ? null : (v as Exclude<Frequency, null>);
}

// selectValueToWindow converts a Select value string back to the nullable
// WindowPreference type accepted by the backend.
function selectValueToWindow(v: string): WindowPreference {
	return v === FOLLOW_SENTINEL ? null : (v as Exclude<WindowPreference, null>);
}

// HoldingScheduleSection renders per-holding schedule override controls
// (frequency + window selects) and exposes nextAnalysisAt from the backend
// so the parent sidebar can display it without a duplicate network call.
//
// The component is self-contained: it owns the query and the mutation so
// it can be embedded anywhere a holdingId is available.
export function HoldingScheduleSection({ holdingId }: HoldingScheduleSectionProps) {
	const { t } = useTranslation("settings");
	const { data, isLoading } = useHoldingSchedule(holdingId);
	const updateMutation = useUpdateHoldingSchedule();

	// Options are built inside the component so t() is called directly,
	// avoiding a (key: string) => string type mismatch with TFunction.
	const freqOptions = [
		{ value: FOLLOW_SENTINEL, label: t("schedule.globalFrequency.followMarket") },
		{ value: "every_window", label: t("schedule.globalFrequency.every_window") },
		{ value: "daily", label: t("schedule.globalFrequency.daily") },
		{ value: "every_2_days", label: t("schedule.globalFrequency.every_2_days") },
		{ value: "every_3_days", label: t("schedule.globalFrequency.every_3_days") },
		{ value: "weekly", label: t("schedule.globalFrequency.weekly") },
		{ value: "custom", label: t("schedule.globalFrequency.custom") },
	];

	const winOptions = [
		{ value: FOLLOW_SENTINEL, label: t("schedule.holdingOverride.windowOptions.follow") },
		{ value: "pre", label: t("schedule.holdingOverride.windowOptions.pre") },
		{ value: "post", label: t("schedule.holdingOverride.windowOptions.post") },
		{ value: "both", label: t("schedule.holdingOverride.windowOptions.both") },
	];

	function handleFrequencyChange(value: string) {
		const patch: UpdateHoldingScheduleInput = {
			frequency: selectValueToFrequency(value),
			frequencyDays: data?.frequencyDays ?? null,
			window: data?.window ?? null,
		};
		updateMutation.mutate({ holdingId, data: patch });
	}

	function handleWindowChange(value: string) {
		const patch: UpdateHoldingScheduleInput = {
			frequency: data?.frequency ?? null,
			frequencyDays: data?.frequencyDays ?? null,
			window: selectValueToWindow(value),
		};
		updateMutation.mutate({ holdingId, data: patch });
	}

	return (
		<Space direction="vertical" size={8} style={{ width: "100%" }}>
			<div>
				<Text type="secondary">{t("schedule.holdingOverride.frequency")}</Text>
				<Select
					style={{ width: "100%", marginTop: 4 }}
					loading={isLoading}
					value={frequencyToSelectValue(data?.frequency ?? null)}
					options={freqOptions}
					onChange={handleFrequencyChange}
					disabled={isLoading || updateMutation.isPending}
					size="small"
				/>
			</div>
			<div>
				<Text type="secondary">{t("schedule.holdingOverride.window")}</Text>
				<Select
					style={{ width: "100%", marginTop: 4 }}
					loading={isLoading}
					value={windowToSelectValue(data?.window ?? null)}
					options={winOptions}
					onChange={handleWindowChange}
					disabled={isLoading || updateMutation.isPending}
					size="small"
				/>
			</div>
		</Space>
	);
}

// useHoldingNextAnalysisAt is a thin hook that re-uses the same cached query
// so MetaSidebar can read nextAnalysisAt without a duplicate network call.
export function useHoldingNextAnalysisAt(holdingId: number): string | null {
	const { data } = useHoldingSchedule(holdingId);
	return data?.nextAnalysisAt ?? null;
}
