import {
	ASSET_CATEGORIES,
	ASSET_CATEGORY_META,
	type AssetCategory,
	type AssetDto,
	useAssets,
} from "@/features/asset-catalog";
import { useCreateHolding, useHoldings } from "@/features/portfolio";
import { usePatchUserSettings, useUserSettings } from "@/features/user-settings";
import {
	Alert,
	Button,
	Collapse,
	Form,
	InputNumber,
	Radio,
	Select,
	Space,
	Tabs,
	Tooltip,
	Typography,
	message,
} from "@/ui-kit/eat";
import { type Variants, motion, useReducedMotion } from "framer-motion";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingState } from "./state";
import { useOnboardingNav } from "./use-onboarding-nav";

const { Text } = Typography;

// containerVariants / itemVariants / reducedItemVariants mirror the constants
// used on WelcomePage and CategoriesPage. Keep the duration + stagger in sync
// so the entrance rhythm reads consistently across the four-step wizard.
const containerVariants: Variants = {
	hidden: { opacity: 0 },
	visible: {
		opacity: 1,
		transition: { staggerChildren: 0.08 },
	},
};

const itemVariants: Variants = {
	hidden: { opacity: 0, y: 20 },
	visible: {
		opacity: 1,
		y: 0,
		transition: { duration: 0.4, ease: "easeOut" },
	},
};

const reducedItemVariants: Variants = {
	hidden: { opacity: 0 },
	visible: { opacity: 1, transition: { duration: 0.2 } },
};

// QuickModeValues mirrors the antd form shape. assetCode is selected via the
// Select search and costPrice/positionRatio are plain numbers in user-visible
// units (CNY for price, percent 0-100 for the position ratio).
interface QuickModeValues {
	assetCode?: string;
	costPrice?: number;
	positionRatio?: number;
}

// QuickModeForm renders the four animated form items for the quick capture
// path. It receives the onboarding nav + draft state from the parent so the
// parent can own the footer CTA (via OnboardingLayout.footer) and the
// canGoNext predicate registration stays close to the draft mutations.
interface QuickModeFormProps {
	itemsVariant: Variants;
}

function QuickModeForm({ itemsVariant }: QuickModeFormProps) {
	const { t } = useTranslation("auth");
	const [form] = Form.useForm<QuickModeValues>();
	const { state, updateHoldingDraft } = useOnboardingState();
	// The radio category is transient UI state: it only controls which asset
	// catalog subset is fetched. The committed assetType is stored on
	// holdingDraft when the user picks a concrete asset below.
	const [category, setCategory] = useState<AssetCategory>(
		(state.holdingDraft.assetType as AssetCategory | undefined) ?? "gold_etf",
	);
	const [keyword, setKeyword] = useState("");

	// Fetch the asset list for the active category plus the current search
	// keyword. The backend endpoint is cheap and the catalog is small, so we
	// intentionally re-fetch on every keystroke rather than paginating.
	const { data: assets, isLoading } = useAssets({
		type: category,
		keyword: keyword || undefined,
	});

	const assetOptions = useMemo(
		() =>
			(assets ?? []).map((a: AssetDto) => ({
				value: a.code,
				label: `${a.code} ${a.name}`,
				data: a,
			})),
		[assets],
	);

	// Seed the form from the draft once on mount. This preserves field values
	// when the user navigates back from a later step and reopens the page.
	// The empty dependency array is deliberate: re-seeding on every draft
	// mutation would fight the user's in-flight edits.
	// biome-ignore lint/correctness/useExhaustiveDependencies: intentional mount-only seed
	useEffect(() => {
		const { assetCode, costPrice, positionRatio } = state.holdingDraft;
		form.setFieldsValue({
			assetCode,
			costPrice,
			positionRatio: positionRatio ?? 10,
		});
	}, []);

	const handleCategoryChange = (next: AssetCategory) => {
		setCategory(next);
		setKeyword("");
		// Reset all asset-dependent fields so the form shows a clean slate and
		// canGoNext does not stay false with stale cost/ratio values visible.
		form.setFieldsValue({ assetCode: undefined, costPrice: undefined, positionRatio: undefined });
		updateHoldingDraft({
			assetCode: undefined,
			assetName: undefined,
			assetType: undefined,
			costPrice: undefined,
			positionRatio: undefined,
		});
	};

	const handleValuesChange = (changed: Partial<QuickModeValues>) => {
		const patch: Parameters<typeof updateHoldingDraft>[0] = {};
		if ("assetCode" in changed) {
			patch.assetCode = changed.assetCode;
			if (changed.assetCode) {
				const asset = assetOptions.find((opt) => opt.value === changed.assetCode)?.data;
				if (asset) {
					patch.assetName = asset.name;
					patch.assetType = asset.assetType;
				}
			} else {
				patch.assetName = undefined;
				patch.assetType = undefined;
			}
		}
		if ("costPrice" in changed) {
			patch.costPrice = changed.costPrice ?? undefined;
		}
		if ("positionRatio" in changed) {
			patch.positionRatio = changed.positionRatio ?? undefined;
		}
		updateHoldingDraft(patch);
	};

	const quickFormRules = useMemo(
		() => ({
			assetCode: [
				{ required: true, message: t("onboarding.firstHolding.validation.assetRequired") },
			],
			costPrice: [
				{ required: true, message: t("onboarding.firstHolding.validation.costPriceRequired") },
			],
			positionRatio: [
				{ required: true, message: t("onboarding.firstHolding.validation.positionRatioRequired") },
				{
					type: "number" as const,
					min: 0,
					max: 100,
					message: t("onboarding.firstHolding.validation.positionRatioRange"),
				},
			],
		}),
		[t],
	);

	return (
		<Form<QuickModeValues> form={form} layout="vertical" onValuesChange={handleValuesChange}>
			<motion.div variants={containerVariants} initial="hidden" animate="visible">
				<motion.div variants={itemsVariant}>
					<Form.Item label={t("onboarding.firstHolding.assetType")} required>
						<Radio.Group
							value={category}
							onChange={(e) => handleCategoryChange(e.target.value as AssetCategory)}
							optionType="button"
							buttonStyle="solid"
							options={ASSET_CATEGORIES.map((key) => ({
								label: ASSET_CATEGORY_META[key].label,
								value: key,
							}))}
						/>
					</Form.Item>
				</motion.div>

				<motion.div variants={itemsVariant}>
					<Form.Item
						label={t("onboarding.firstHolding.asset")}
						name="assetCode"
						rules={quickFormRules.assetCode}
					>
						<Select
							showSearch
							allowClear
							placeholder={t("onboarding.firstHolding.assetPlaceholder")}
							loading={isLoading}
							filterOption={false}
							onSearch={setKeyword}
							options={assetOptions.map(({ value, label }) => ({ value, label }))}
							notFoundContent={t("onboarding.firstHolding.error.noMatchFound")}
						/>
					</Form.Item>
				</motion.div>

				<motion.div variants={itemsVariant}>
					<Form.Item
						label={t("onboarding.firstHolding.costPrice")}
						name="costPrice"
						rules={quickFormRules.costPrice}
					>
						<InputNumber
							min={0}
							step={0.01}
							style={{ width: "100%" }}
							placeholder={t("onboarding.firstHolding.costPricePlaceholder")}
						/>
					</Form.Item>
				</motion.div>

				<motion.div variants={itemsVariant}>
					<Form.Item
						label={t("onboarding.firstHolding.positionRatio")}
						name="positionRatio"
						rules={quickFormRules.positionRatio}
						extra={t("onboarding.firstHolding.positionRatioExtra")}
					>
						<InputNumber min={0} max={100} step={1} style={{ width: "100%" }} />
					</Form.Item>
				</motion.div>
			</motion.div>
		</Form>
	);
}

function TotalCapitalCollapse() {
	const { t } = useTranslation("auth");
	const { data: settings } = useUserSettings();
	const patch = usePatchUserSettings();
	const [value, setValue] = useState<number | null>(settings?.totalCapitalCny ?? null);

	// Sync local state once the settings query resolves after mount. Without
	// this effect, users arriving on the page before the query completes see
	// an empty input even when the server already has a total capital value.
	useEffect(() => {
		if (settings?.totalCapitalCny != null) {
			setValue(settings.totalCapitalCny);
		}
	}, [settings?.totalCapitalCny]);

	const handleSave = async () => {
		if (value == null || value <= 0) {
			message.error(t("onboarding.firstHolding.totalCapital.invalidValue"));
			return;
		}
		try {
			await patch.mutateAsync({ totalCapitalCny: value });
			message.success(t("onboarding.firstHolding.totalCapital.saved"));
		} catch {
			message.error(t("onboarding.firstHolding.totalCapital.saveError"));
		}
	};

	return (
		<Collapse
			ghost
			items={[
				{
					key: "capital",
					label: (
						<Text type="secondary" style={{ fontSize: 13 }}>
							{t("onboarding.firstHolding.totalCapital.label")}
						</Text>
					),
					children: (
						<Space.Compact style={{ width: "100%" }}>
							<InputNumber
								min={0}
								step={1000}
								value={value}
								onChange={(v) => setValue(v)}
								style={{ flex: 1 }}
								placeholder={t("onboarding.firstHolding.totalCapital.placeholder")}
								data-testid="onboarding-total-capital-input"
							/>
							<Button
								type="primary"
								loading={patch.isPending}
								onClick={handleSave}
								data-testid="onboarding-total-capital-save"
							>
								{t("onboarding.firstHolding.totalCapital.save")}
							</Button>
						</Space.Compact>
					),
				},
			]}
		/>
	);
}

export default function FirstHoldingPage() {
	const { t } = useTranslation("auth");
	const nav = useOnboardingNav();
	const { state, updateHoldingDraft } = useOnboardingState();
	const { data: holdings } = useHoldings();
	const createHolding = useCreateHolding();
	const reducedMotion = useReducedMotion();
	const items = reducedMotion ? reducedItemVariants : itemVariants;
	const [activeTab, setActiveTab] = useState<string>(state.holdingDraft.mode ?? "quick");

	const existingCount = holdings?.length ?? 0;
	const disabledTooltip = t("onboarding.firstHolding.comingSoon");

	// Register a canGoNext predicate that validates the active mode's required
	// fields. Only the quick mode is enabled in this release; detail and
	// screenshot tabs are disabled in the Tabs items array below, so the
	// predicate does not need to handle them yet. The screenshot branch
	// returns false to keep the gate closed in case the mode field somehow
	// flips to a non-quick value via a stale draft.
	useEffect(() => {
		return nav.registerCanGoNext(() => {
			const draft = state.holdingDraft;
			if (draft.mode === "quick") {
				return Boolean(draft.assetCode && draft.costPrice != null && draft.positionRatio != null);
			}
			return false;
		});
	}, [nav, state.holdingDraft]);

	const handleTabChange = (key: string) => {
		setActiveTab(key);
		if (key === "quick" || key === "detail" || key === "screenshot") {
			updateHoldingDraft({ mode: key });
		}
	};

	const handleSubmit = async () => {
		if (!nav.canGoNext) return;
		const draft = state.holdingDraft;
		if (!draft.assetCode || !draft.assetName || !draft.assetType) {
			message.error(t("onboarding.firstHolding.validation.holdingAssetRequired"));
			return;
		}
		if (draft.costPrice == null || draft.positionRatio == null) {
			message.error(t("onboarding.firstHolding.validation.holdingCostAndRatioRequired"));
			return;
		}
		try {
			await createHolding.mutateAsync({
				assetCode: draft.assetCode,
				assetName: draft.assetName,
				assetType: draft.assetType,
				costPrice: draft.costPrice,
				positionRatio: draft.positionRatio,
				// Onboarding quick mode captures cost + percentage only; the user
				// enters real trade quantities later on the transactions sub-page.
				quantity: 0,
			});
			await nav.next();
		} catch {
			message.error(t("onboarding.firstHolding.error.saveFailed"));
		}
	};

	// The fast-forward button is distinct from the header "跳过引导" control.
	// Header skip aborts the whole wizard via POST /onboarding/skip; this
	// button advances the form into step 4 so FirstAnalysisPage owns the
	// markCompleted call (step 4 is the single source of truth for completion).
	const handleFastForward = async () => {
		await nav.forceNext();
	};

	return (
		<OnboardingLayout
			currentStep={3}
			title={t("onboarding.firstHolding.title")}
			description={t("onboarding.firstHolding.description")}
			footer={
				<Button
					type="primary"
					size="large"
					data-testid="onboarding-holding-submit"
					disabled={!nav.canGoNext}
					loading={createHolding.isPending}
					onClick={handleSubmit}
				>
					{t("onboarding.firstHolding.submitButton")}
				</Button>
			}
		>
			{existingCount > 0 ? (
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 16 }}
					message={t("onboarding.firstHolding.existingHoldings.detected", { count: existingCount })}
					description={t("onboarding.firstHolding.existingHoldings.description")}
					action={
						<Button
							type="primary"
							size="small"
							data-testid="onboarding-skip-to-analysis"
							onClick={handleFastForward}
						>
							{t("onboarding.firstHolding.existingHoldings.directAnalysis")}
						</Button>
					}
				/>
			) : null}

			<Tabs
				activeKey={activeTab}
				onChange={handleTabChange}
				items={[
					{
						key: "quick",
						label: t("onboarding.firstHolding.tab.quick"),
						children: <QuickModeForm itemsVariant={items} />,
					},
					{
						key: "detail",
						label: (
							<Tooltip title={disabledTooltip}>
								<span>{t("onboarding.firstHolding.tab.detail")}</span>
							</Tooltip>
						),
						disabled: true,
						children: null,
					},
					{
						key: "screenshot",
						label: (
							<Tooltip title={disabledTooltip}>
								<span>{t("onboarding.firstHolding.tab.screenshot")}</span>
							</Tooltip>
						),
						disabled: true,
						children: null,
					},
				]}
			/>

			<div style={{ marginTop: 16 }}>
				<TotalCapitalCollapse />
			</div>
		</OnboardingLayout>
	);
}
