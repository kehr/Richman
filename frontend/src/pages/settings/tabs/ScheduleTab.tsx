import type { MarketScheduleDTO, ScheduleSettingsDTO } from "@/features/schedule";
import { useScheduleSettings, useUpdateScheduleSettings } from "@/features/schedule";
import { Button, Divider, Flex, Skeleton, message } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { GlobalFrequencySelector } from "./schedule/GlobalFrequencySelector";
import { HKPlaceholderCard } from "./schedule/HKPlaceholderCard";
import { MarketWindowCard } from "./schedule/MarketWindowCard";

// ScheduleTab is the composition root for the schedule settings view. It loads
// the global schedule config, keeps a local draft, and flushes it to the server
// when the user presses Save. All nested components receive draft slices and
// emit partial updates via onUpdate callbacks.
export function ScheduleTab() {
	const { t } = useTranslation("settings");
	const query = useScheduleSettings();
	const mutation = useUpdateScheduleSettings();

	// draft mirrors the server data while the user is editing. It is
	// initialised from server data on first load and reset to server data when
	// the query refetches after a successful mutation.
	const [draft, setDraft] = useState<ScheduleSettingsDTO | null>(null);

	// Merge server data into draft on first load or when the server data
	// changes and the user has not yet made edits.
	const serverData = query.data;
	const effective: ScheduleSettingsDTO | null = draft ?? serverData ?? null;

	const handleGlobalFrequencyChange = (
		freq: ScheduleSettingsDTO["globalFrequency"],
		days?: number,
	) => {
		if (!effective) return;
		setDraft({
			...effective,
			globalFrequency: freq,
			globalFrequencyDays: freq === "custom" ? (days ?? effective.globalFrequencyDays ?? 7) : null,
		});
	};

	const handleMarketUpdate = (
		market: keyof ScheduleSettingsDTO["markets"],
		patch: Partial<MarketScheduleDTO>,
	) => {
		if (!effective) return;
		setDraft({
			...effective,
			markets: {
				...effective.markets,
				[market]: { ...effective.markets[market], ...patch },
			},
		});
	};

	const handleSave = async () => {
		if (!effective) return;
		try {
			await mutation.mutateAsync(effective);
			// Reset draft after successful save so the UI reflects server state.
			setDraft(null);
			message.success(t("schedule.saveSuccess"));
		} catch {
			message.error(t("schedule.saveError"));
		}
	};

	if (query.isLoading) {
		return <Skeleton active paragraph={{ rows: 8 }} />;
	}

	if (!effective) {
		return null;
	}

	return (
		<Flex vertical gap={24} data-testid="schedule-tab">
			<GlobalFrequencySelector
				value={effective.globalFrequency}
				customDays={effective.globalFrequencyDays}
				onChange={handleGlobalFrequencyChange}
			/>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={0}>
				<MarketWindowCard
					market="a_share"
					settings={effective.markets.a_share}
					onUpdate={(patch) => handleMarketUpdate("a_share", patch)}
				/>
				<MarketWindowCard
					market="us_stock"
					settings={effective.markets.us_stock}
					onUpdate={(patch) => handleMarketUpdate("us_stock", patch)}
				/>
				<HKPlaceholderCard />
			</Flex>

			<Flex justify="flex-end">
				<Button type="primary" loading={mutation.isPending} onClick={handleSave}>
					{t("action.save", { ns: "common" })}
				</Button>
			</Flex>
		</Flex>
	);
}
