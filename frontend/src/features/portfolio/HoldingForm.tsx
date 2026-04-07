import { Button, Form, InputNumber, message } from "@/ui-kit/eat";
import { type ReactNode, useState } from "react";
import type { HoldingDto } from "./api";
import { useCreateHolding, useUpdateHolding } from "./usePortfolio";

export interface SelectedAsset {
	code: string;
	name: string;
	assetType: string;
}

interface HoldingFormProps {
	initialValues?: HoldingDto;
	onSuccess?: () => void;
	renderAssetPicker?: (props: {
		open: boolean;
		onClose: () => void;
		onSelect: (asset: SelectedAsset) => void;
	}) => ReactNode;
}

export function HoldingForm({ initialValues, onSuccess, renderAssetPicker }: HoldingFormProps) {
	const [form] = Form.useForm();
	const [pickerOpen, setPickerOpen] = useState(false);
	const [selectedAsset, setSelectedAsset] = useState<SelectedAsset | null>(null);
	const createMutation = useCreateHolding();
	const updateMutation = useUpdateHolding();

	const isEdit = !!initialValues;

	const handleAssetSelect = (asset: SelectedAsset) => {
		setSelectedAsset(asset);
		form.setFieldsValue({
			assetCode: asset.code,
			assetName: asset.name,
			assetType: asset.assetType,
		});
	};

	const handleSubmit = async (values: Record<string, unknown>) => {
		try {
			if (isEdit) {
				await updateMutation.mutateAsync({
					id: initialValues.holdingId,
					data: {
						costPrice: values.costPrice as number,
						positionRatio: values.positionRatio as number,
					},
				});
				message.success("Holding updated");
			} else {
				await createMutation.mutateAsync({
					assetCode: values.assetCode as string,
					assetName: values.assetName as string,
					assetType: values.assetType as string,
					costPrice: values.costPrice as number,
					positionRatio: values.positionRatio as number,
				});
				message.success("Holding created");
			}
			onSuccess?.();
		} catch {
			message.error("Operation failed");
		}
	};

	return (
		<>
			<Form
				form={form}
				layout="vertical"
				initialValues={
					initialValues
						? {
								assetCode: initialValues.assetCode,
								assetName: initialValues.assetName,
								assetType: initialValues.assetType,
								costPrice: initialValues.costPrice,
								positionRatio: initialValues.positionRatio,
							}
						: undefined
				}
				onFinish={handleSubmit}
			>
				{!isEdit && (
					<Form.Item label="Asset" required>
						<Button onClick={() => setPickerOpen(true)}>
							{selectedAsset ? `${selectedAsset.code} - ${selectedAsset.name}` : "Select Asset"}
						</Button>
						<Form.Item name="assetCode" noStyle rules={[{ required: true }]}>
							<input type="hidden" />
						</Form.Item>
						<Form.Item name="assetName" noStyle>
							<input type="hidden" />
						</Form.Item>
						<Form.Item name="assetType" noStyle>
							<input type="hidden" />
						</Form.Item>
					</Form.Item>
				)}

				<Form.Item
					label="Cost Price"
					name="costPrice"
					rules={[{ required: true, message: "Please enter cost price" }]}
				>
					<InputNumber min={0} step={0.01} style={{ width: "100%" }} />
				</Form.Item>

				<Form.Item
					label="Position Ratio (%)"
					name="positionRatio"
					rules={[{ required: true, message: "Please enter position ratio" }]}
					tooltip="0-100 percent. Must agree with HoldingTable / DashboardPage / decision card consumers (Step 16 unified to percent)."
				>
					<InputNumber min={0} max={100} step={1} style={{ width: "100%" }} />
				</Form.Item>

				<Form.Item>
					<Button
						type="primary"
						htmlType="submit"
						loading={createMutation.isPending || updateMutation.isPending}
					>
						{isEdit ? "Update" : "Create"}
					</Button>
				</Form.Item>
			</Form>

			{renderAssetPicker?.({
				open: pickerOpen,
				onClose: () => setPickerOpen(false),
				onSelect: handleAssetSelect,
			})}
		</>
	);
}
