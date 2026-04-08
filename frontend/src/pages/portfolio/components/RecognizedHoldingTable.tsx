import {
	CONFIDENCE_HIGH,
	CONFIDENCE_LOW,
	type EditableRecognizedHolding,
} from "@/features/portfolio";
import {
	Alert,
	Button,
	Checkbox,
	DeleteOutlined,
	Input,
	InputNumber,
	Space,
	Table,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import type { CSSProperties } from "react";

// RecognizedHoldingTable renders the right pane of ScreenshotImportModal
// (PRD §4.3). It owns no state itself; all row mutations bubble up to the
// modal so the modal can drive the bulk-create flow on confirm.
//
// The 5-holding cap from §4.1 is enforced visually here: rows that would
// push the user above the cap have their checkbox disabled, and a warning
// banner explains the limit.

interface RecognizedHoldingTableProps {
	rows: EditableRecognizedHolding[];
	currentHoldingCount: number;
	holdingLimit: number;
	onChange: (rowId: string, patch: Partial<EditableRecognizedHolding>) => void;
	onDelete: (rowId: string) => void;
}

function fieldStyle(confidence: number): CSSProperties {
	if (confidence >= CONFIDENCE_HIGH) {
		return { background: "#fff" };
	}
	if (confidence >= CONFIDENCE_LOW) {
		return {
			background: "#fffbe6",
			borderColor: "#faad14",
		};
	}
	return {
		background: "#fff",
		borderColor: "#ff4d4f",
	};
}

function confidenceTooltip(confidence: number): string | null {
	if (confidence >= CONFIDENCE_HIGH) return null;
	if (confidence >= CONFIDENCE_LOW) return "识别置信度中等，请检查";
	return "请手动填写";
}

export function RecognizedHoldingTable({
	rows,
	currentHoldingCount,
	holdingLimit,
	onChange,
	onDelete,
}: RecognizedHoldingTableProps) {
	const remainingSlots = Math.max(0, holdingLimit - currentHoldingCount);
	const selectedCount = rows.filter((r) => r.selected).length;
	const capReached = selectedCount >= remainingSlots;

	const handleToggle = (row: EditableRecognizedHolding, next: boolean) => {
		// Refuse to enable a row that would push us beyond the cap. The checkbox
		// for unselected rows is also rendered as disabled when capReached, so
		// this is just a defensive guard.
		if (next && capReached && !row.selected) return;
		onChange(row.rowId, { selected: next });
	};

	const renderConfidenceWrapper = (confidence: number, node: React.ReactNode) => {
		const tip = confidenceTooltip(confidence);
		if (!tip) return node;
		return <Tooltip title={tip}>{node}</Tooltip>;
	};

	const columns = [
		{
			title: "",
			key: "select",
			width: 48,
			render: (_: unknown, row: EditableRecognizedHolding) => {
				const disabled = !row.selected && capReached;
				const node = (
					<Checkbox
						checked={row.selected}
						disabled={disabled}
						onChange={(e) => handleToggle(row, e.target.checked)}
						data-testid={`recognized-row-checkbox-${row.rowId}`}
					/>
				);
				return disabled ? <Tooltip title="已达 5 个标的上限">{node}</Tooltip> : node;
			},
		},
		{
			title: "名称",
			key: "assetName",
			render: (_: unknown, row: EditableRecognizedHolding) =>
				renderConfidenceWrapper(
					row.assetNameConfidence,
					<Input
						value={row.assetName}
						placeholder="请手动填写"
						style={fieldStyle(row.assetNameConfidence)}
						onChange={(e) => onChange(row.rowId, { assetName: e.target.value })}
						data-testid={`recognized-row-name-${row.rowId}`}
					/>,
				),
		},
		{
			title: "代码",
			key: "assetCode",
			width: 140,
			render: (_: unknown, row: EditableRecognizedHolding) =>
				renderConfidenceWrapper(
					row.assetCodeConfidence,
					<Input
						value={row.assetCode}
						placeholder="请手动填写"
						style={fieldStyle(row.assetCodeConfidence)}
						onChange={(e) => onChange(row.rowId, { assetCode: e.target.value })}
						data-testid={`recognized-row-code-${row.rowId}`}
					/>,
				),
		},
		{
			title: "成本",
			key: "costPrice",
			width: 140,
			render: (_: unknown, row: EditableRecognizedHolding) => {
				return renderConfidenceWrapper(
					row.costPriceConfidence,
					<InputNumber
						value={row.costPrice ?? undefined}
						min={0}
						step={0.01}
						style={{ width: "100%", ...fieldStyle(row.costPriceConfidence) }}
						placeholder={row.costPriceConfidence < CONFIDENCE_LOW ? "请手动填写" : undefined}
						onChange={(value) =>
							onChange(row.rowId, { costPrice: typeof value === "number" ? value : null })
						}
						data-testid={`recognized-row-cost-${row.rowId}`}
					/>,
				);
			},
		},
		{
			title: "仓位",
			key: "positionRatio",
			width: 140,
			render: (_: unknown, row: EditableRecognizedHolding) => {
				return renderConfidenceWrapper(
					row.positionRatioConfidence,
					<InputNumber
						value={row.positionRatio ?? undefined}
						min={0}
						max={100}
						step={1}
						addonAfter="%"
						style={{ width: "100%", ...fieldStyle(row.positionRatioConfidence) }}
						placeholder={row.positionRatioConfidence < CONFIDENCE_LOW ? "请手动填写" : undefined}
						onChange={(value) =>
							onChange(row.rowId, {
								positionRatio: typeof value === "number" ? value : null,
							})
						}
						data-testid={`recognized-row-pct-${row.rowId}`}
					/>,
				);
			},
		},
		{
			title: "",
			key: "actions",
			width: 48,
			render: (_: unknown, row: EditableRecognizedHolding) => (
				<Button
					type="text"
					danger
					size="small"
					icon={<DeleteOutlined />}
					onClick={() => onDelete(row.rowId)}
					data-testid={`recognized-row-delete-${row.rowId}`}
					aria-label="移除该行"
				/>
			),
		},
	];

	return (
		<Space direction="vertical" size="small" style={{ width: "100%" }}>
			{capReached && (
				<Alert
					type="warning"
					showIcon
					message={`MVP 最多 ${holdingLimit} 个标的，当前已有 ${currentHoldingCount} 个，已勾选 ${selectedCount} 个，已达上限`}
					data-testid="recognized-cap-warning"
				/>
			)}
			<Table<EditableRecognizedHolding>
				size="small"
				columns={columns}
				dataSource={rows}
				rowKey="rowId"
				pagination={false}
				data-testid="recognized-holding-table"
			/>
			<Typography.Text type="secondary" data-testid="recognized-summary">
				将新增 {selectedCount} 个持仓
			</Typography.Text>
		</Space>
	);
}
