import { Segmented } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// HoldingEntryMode represents the three holding entry modes (TRD SS7.1).
// "tag"    — asset only; system fills price/ratio defaults
// "quick"  — asset + position (existing quick form)
// "detail" — all fields (existing detail form)
export type HoldingEntryMode = "tag" | "quick" | "detail";

interface ModeSelectorProps {
	value: HoldingEntryMode;
	onChange: (mode: HoldingEntryMode) => void;
}

// ModeSelector lets the user pick an entry mode for the add-holding drawer.
// Rendered inside AddHoldingDrawer after an asset has been selected.
export function ModeSelector({ value, onChange }: ModeSelectorProps) {
	const { t } = useTranslation("app");

	return (
		<Segmented
			value={value}
			onChange={(v) => onChange(v as HoldingEntryMode)}
			options={[
				{ label: t("portfolio.modeSelector.tag"), value: "tag" },
				{ label: t("portfolio.modeSelector.quick"), value: "quick" },
				{ label: t("portfolio.modeSelector.detail"), value: "detail" },
			]}
			block
		/>
	);
}
