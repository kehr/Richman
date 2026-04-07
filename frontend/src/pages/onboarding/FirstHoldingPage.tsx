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
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";

const { Text } = Typography;

// QuickModeValues mirrors the antd form shape. assetCode is selected via the
// Select search and costPrice/positionRatio are plain numbers in user-visible
// units (CNY for price, percent 0-100 for the position ratio).
interface QuickModeValues {
	assetCode: string;
	costPrice: number;
	positionRatio: number;
}

function QuickModeForm({ onSuccess }: { onSuccess: () => void }) {
	const [form] = Form.useForm<QuickModeValues>();
	const [category, setCategory] = useState<AssetCategory>("gold_etf");
	const [keyword, setKeyword] = useState("");
	const createHolding = useCreateHolding();

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

	const handleCategoryChange = (next: AssetCategory) => {
		setCategory(next);
		setKeyword("");
		form.setFieldValue("assetCode", undefined);
	};

	const handleSubmit = async (values: QuickModeValues) => {
		const asset = assetOptions.find((opt) => opt.value === values.assetCode)?.data;
		if (!asset) {
			message.error("请选择一个标的");
			return;
		}
		try {
			await createHolding.mutateAsync({
				assetCode: asset.code,
				assetName: asset.name,
				assetType: asset.assetType,
				costPrice: values.costPrice,
				positionRatio: values.positionRatio,
			});
			message.success("持仓已保存");
			onSuccess();
		} catch {
			message.error("保存失败，请重试");
		}
	};

	return (
		<Form<QuickModeValues>
			form={form}
			layout="vertical"
			onFinish={handleSubmit}
			initialValues={{ positionRatio: 10 }}
		>
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

			<Form.Item
				label="均价成本（单价）"
				name="costPrice"
				rules={[{ required: true, message: "请输入均价成本" }]}
			>
				<InputNumber min={0} step={0.01} style={{ width: "100%" }} placeholder="例如 3.25" />
			</Form.Item>

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

			<Form.Item>
				<Button
					type="primary"
					htmlType="submit"
					size="large"
					block
					data-testid="onboarding-holding-submit"
					loading={createHolding.isPending}
				>
					保存并开始分析 →
				</Button>
			</Form.Item>
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
	const navigate = useNavigate();
	const { data: holdings } = useHoldings();
	const [activeTab, setActiveTab] = useState("quick");

	const existingCount = holdings?.length ?? 0;

	const disabledTooltip = "即将推出（Step 16/17）";

	return (
		<OnboardingLayout
			currentStep={3}
			title="先录一个持仓"
			description="先录一个就行，后面随时可以加。我们需要至少一个持仓才能生成决策卡。"
		>
			{existingCount > 0 ? (
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 16 }}
					message={`检测到你已有 ${existingCount} 个持仓`}
					description="可以直接跳到分析步骤，或继续在下面添加一个新的。"
					action={
						<Button
							type="primary"
							size="small"
							data-testid="onboarding-skip-to-analysis"
							onClick={() => navigate("/onboarding/first-analysis")}
						>
							跳过，直接开始分析
						</Button>
					}
				/>
			) : null}

			<Tabs
				activeKey={activeTab}
				onChange={setActiveTab}
				items={[
					{
						key: "quick",
						label: "快速模式",
						children: <QuickModeForm onSuccess={() => navigate("/onboarding/first-analysis")} />,
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
