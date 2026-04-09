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

	return (
		<Form<QuickModeValues> form={form} layout="vertical" onValuesChange={handleValuesChange}>
			<motion.div variants={containerVariants} initial="hidden" animate="visible">
				<motion.div variants={itemsVariant}>
					<Form.Item label="标的类型" required>
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
						label="标的"
						name="assetCode"
						rules={[{ required: true, message: "请选择一个标的" }]}
					>
						<Select
							showSearch
							allowClear
							placeholder="搜索代码或名称"
							loading={isLoading}
							filterOption={false}
							onSearch={setKeyword}
							options={assetOptions.map(({ value, label }) => ({ value, label }))}
							notFoundContent={isLoading ? "加载中..." : "暂无匹配"}
						/>
					</Form.Item>
				</motion.div>

				<motion.div variants={itemsVariant}>
					<Form.Item
						label="均价成本（单价）"
						name="costPrice"
						rules={[{ required: true, message: "请输入均价成本" }]}
					>
						<InputNumber min={0} step={0.01} style={{ width: "100%" }} placeholder="例如 3.25" />
					</Form.Item>
				</motion.div>

				<motion.div variants={itemsVariant}>
					<Form.Item
						label="仓位比例（%）"
						name="positionRatio"
						rules={[
							{ required: true, message: "请输入仓位比例" },
							{ type: "number", min: 0, max: 100, message: "0 - 100 之间" },
						]}
						extra="该标的占你整体资金的比例，无需总资金也能计算"
					>
						<InputNumber min={0} max={100} step={1} style={{ width: "100%" }} />
					</Form.Item>
				</motion.div>
			</motion.div>
		</Form>
	);
}

function TotalCapitalCollapse() {
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
			message.error("请输入有效的总资金");
			return;
		}
		try {
			await patch.mutateAsync({ totalCapitalCny: value });
			message.success("总资金已保存");
		} catch {
			message.error("保存失败");
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
							想看具体金额？设置总资金（可选）
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
								placeholder="例如 100000"
								data-testid="onboarding-total-capital-input"
							/>
							<Button
								type="primary"
								loading={patch.isPending}
								onClick={handleSave}
								data-testid="onboarding-total-capital-save"
							>
								保存
							</Button>
						</Space.Compact>
					),
				},
			]}
		/>
	);
}

export default function FirstHoldingPage() {
	const nav = useOnboardingNav();
	const { state, updateHoldingDraft } = useOnboardingState();
	const { data: holdings } = useHoldings();
	const createHolding = useCreateHolding();
	const reducedMotion = useReducedMotion();
	const items = reducedMotion ? reducedItemVariants : itemVariants;
	const [activeTab, setActiveTab] = useState<string>(state.holdingDraft.mode ?? "quick");

	const existingCount = holdings?.length ?? 0;
	const disabledTooltip = "即将推出（Step 16/17）";

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
			message.error("请选择一个标的");
			return;
		}
		if (draft.costPrice == null || draft.positionRatio == null) {
			message.error("请填写成本价和仓位比例");
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
			message.success("持仓已保存");
			await nav.next();
		} catch {
			message.error("保存持仓失败，请稍后重试");
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
			title="先录一个持仓"
			description="先录一个就行，后面随时可以加。我们需要至少一个持仓才能生成决策卡。"
			footer={
				<Button
					type="primary"
					size="large"
					data-testid="onboarding-holding-submit"
					disabled={!nav.canGoNext}
					loading={createHolding.isPending}
					onClick={handleSubmit}
				>
					保存并开始分析 →
				</Button>
			}
		>
			{existingCount > 0 ? (
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 16 }}
					message={`检测到你已有 ${existingCount} 个持仓`}
					description="可以直接用已有持仓进入分析，或继续在下面添加一个新的。"
					action={
						<Button
							type="primary"
							size="small"
							data-testid="onboarding-skip-to-analysis"
							onClick={handleFastForward}
						>
							用已有持仓直接分析 →
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
						label: "快速模式",
						children: <QuickModeForm itemsVariant={items} />,
					},
					{
						key: "detail",
						label: (
							<Tooltip title={disabledTooltip}>
								<span>明细模式</span>
							</Tooltip>
						),
						disabled: true,
						children: null,
					},
					{
						key: "screenshot",
						label: (
							<Tooltip title={disabledTooltip}>
								<span>截图识别</span>
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
