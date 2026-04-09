import { useCreateHolding } from "@/features/portfolio";
import {
	Button,
	Drawer,
	Flex,
	Form,
	Space,
	Steps,
	Tabs,
	Tag,
	Tooltip,
	Typography,
	message,
} from "@/ui-kit/eat";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { AssetTypeStep, type SelectedAssetValue } from "./AssetTypeStep";
import { QuickHoldingForm, type QuickHoldingFormValues } from "./QuickHoldingForm";

// AddHoldingDrawer implements the two-step add-holding flow from PRD §4.2:
//   step 1 — pick asset type + search the catalog
//   step 2 — three tabs (quick / detail / screenshot); only quick is functional
//            in this step. The other tabs are disabled with tooltips because
//            the detail editor and screenshot import ship in later steps.

interface AddHoldingDrawerProps {
	open: boolean;
	onClose: () => void;
	onCreated?: () => void;
}

type TabKey = "quick" | "detail" | "screenshot";

export function AddHoldingDrawer({ open, onClose, onCreated }: AddHoldingDrawerProps) {
	const { t } = useTranslation("app");
	const [selectedAsset, setSelectedAsset] = useState<SelectedAssetValue | null>(null);
	const [activeTab, setActiveTab] = useState<TabKey>("quick");
	const [form] = Form.useForm<QuickHoldingFormValues>();
	const createMutation = useCreateHolding();

	// Reset drawer state whenever the drawer is closed so the next open starts
	// fresh at step 1. Running this inside onClose would be cleaner but the
	// drawer may also be dismissed via the mask / escape key which bypass our
	// cancel button.
	useEffect(() => {
		if (!open) {
			setSelectedAsset(null);
			setActiveTab("quick");
			form.resetFields();
		}
	}, [open, form]);

	const handleAssetSelect = (asset: SelectedAssetValue) => {
		setSelectedAsset(asset);
	};

	const handleChangeAsset = () => {
		setSelectedAsset(null);
		form.resetFields();
	};

	const handleSubmit = async (values: QuickHoldingFormValues) => {
		if (!selectedAsset) return;
		try {
			await createMutation.mutateAsync({
				assetCode: selectedAsset.code,
				assetName: selectedAsset.name,
				assetType: selectedAsset.assetType,
				costPrice: values.costPrice,
				positionRatio: values.positionRatio,
				// Quick mode captures cost + percentage only; share quantity is
				// recorded separately on the transactions sub-page.
				quantity: 0,
			});
			message.success(t("portfolio.addHoldingDrawer.saveSuccess"));
			onCreated?.();
			onClose();
		} catch {
			message.error(t("portfolio.addHoldingDrawer.saveError"));
		}
	};

	const currentStep = selectedAsset ? 1 : 0;

	const tabItems = [
		{
			key: "quick",
			label: t("portfolio.addHoldingDrawer.tabQuick"),
			children: <QuickHoldingForm form={form} onFinish={handleSubmit} />,
		},
		{
			key: "detail",
			label: (
				// antd disables pointer events on disabled tab labels, so the
				// Tooltip will not actually open on hover. Wrap the label span
				// in pointerEvents:auto so the hint is still reachable.
				<Tooltip title={t("portfolio.addHoldingDrawer.comingSoon")}>
					<span
						data-testid="tab-detail-disabled"
						style={{ display: "inline-block", pointerEvents: "auto" }}
					>
						{t("portfolio.addHoldingDrawer.tabDetail")}
					</span>
				</Tooltip>
			),
			disabled: true,
			children: null,
		},
		{
			key: "screenshot",
			label: (
				<Tooltip title={t("portfolio.addHoldingDrawer.comingSoon")}>
					<span
						data-testid="tab-screenshot-disabled"
						style={{ display: "inline-block", pointerEvents: "auto" }}
					>
						{t("portfolio.addHoldingDrawer.tabScreenshot")}
					</span>
				</Tooltip>
			),
			disabled: true,
			children: null,
		},
	];

	return (
		<Drawer
			title={t("portfolio.addHoldingDrawer.title")}
			placement="right"
			width={720}
			open={open}
			onClose={onClose}
			data-testid="add-holding-drawer"
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={onClose}>{t("action.cancel", { ns: "common" })}</Button>
					<Button
						type="primary"
						disabled={!selectedAsset}
						loading={createMutation.isPending}
						onClick={() => form.submit()}
						data-testid="add-holding-save"
					>
						{t("action.save", { ns: "common" })}
					</Button>
				</Flex>
			}
		>
			<Steps
				current={currentStep}
				size="small"
				items={[
					{ title: t("portfolio.addHoldingDrawer.step1") },
					{ title: t("portfolio.addHoldingDrawer.step2") },
				]}
				style={{ marginBottom: 24 }}
				data-testid="add-holding-steps"
			/>

			{selectedAsset ? (
				<Space direction="vertical" size="middle" style={{ width: "100%" }}>
					<Flex align="center" gap={8} data-testid="selected-asset-chip">
						<Tag color="blue">{selectedAsset.code}</Tag>
						<Typography.Text strong>{selectedAsset.name}</Typography.Text>
						<Button type="link" size="small" onClick={handleChangeAsset}>
							{t("portfolio.addHoldingDrawer.changeAsset")}
						</Button>
					</Flex>
					<Tabs
						activeKey={activeTab}
						onChange={(k) => setActiveTab(k as TabKey)}
						items={tabItems}
					/>
				</Space>
			) : (
				<AssetTypeStep onSelect={handleAssetSelect} />
			)}
		</Drawer>
	);
}
