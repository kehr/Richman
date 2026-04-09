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
import { useTranslation } from "react-i18next";

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

export function RecognizedHoldingTable({
	rows,
	currentHoldingCount,
	holdingLimit,
	onChange,
	onDelete,
}: RecognizedHoldingTableProps) {
	const { t } = useTranslation("app");
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
		const tip =
			confidence >= CONFIDENCE_HIGH
				? null
				: confidence >= CONFIDENCE_LOW
					? t("portfolio.screenshotModal.confidence.medium")
					: t("portfolio.screenshotModal.confidence.low");
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
				return disabled ? (
					<Tooltip title={t("portfolio.screenshotModal.rowLimitReached", { limit: holdingLimit })}>
						{node}
					</Tooltip>
				) : (
					node
				);
			},
		},
		{
			title: t("portfolio.recognizedTable.name"),
			key: "assetName",
			render: (_: unknown, row: EditableRecognizedHolding) =>
				renderConfidenceWrapper(
					row.assetNameConfidence,
					<Input
						value={row.assetName}
						placeholder={t("portfolio.screenshotModal.placeholder")}
						style={fieldStyle(row.assetNameConfidence)}
						onChange={(e) => onChange(row.rowId, { assetName: e.target.value })}
						data-testid={`recognized-row-name-${row.rowId}`}
					/>,
				),
		},
		{
			title: t("portfolio.recognizedTable.code"),
			key: "assetCode",
			width: 140,
			render: (_: unknown, row: EditableRecognizedHolding) =>
				renderConfidenceWrapper(
					row.assetCodeConfidence,
					<Input
						value={row.assetCode}
						placeholder={t("portfolio.screenshotModal.placeholder")}
						style={fieldStyle(row.assetCodeConfidence)}
						onChange={(e) => onChange(row.rowId, { assetCode: e.target.value })}
						data-testid={`recognized-row-code-${row.rowId}`}
					/>,
				),
		},
		{
			title: t("portfolio.recognizedTable.cost"),
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
						placeholder={
							row.costPriceConfidence < CONFIDENCE_LOW
								? t("portfolio.screenshotModal.placeholder")
								: undefined
						}
						onChange={(value) =>
							onChange(row.rowId, { costPrice: typeof value === "number" ? value : null })
						}
						data-testid={`recognized-row-cost-${row.rowId}`}
					/>,
				);
			},
		},
		{
			title: t("portfolio.recognizedTable.position"),
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
						placeholder={
							row.positionRatioConfidence < CONFIDENCE_LOW
								? t("portfolio.screenshotModal.placeholder")
								: undefined
						}
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
					aria-label={t("portfolio.recognizedTable.removeRow")}
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
					message={t("portfolio.screenshotModal.capWarning", {
						limit: holdingLimit,
						current: currentHoldingCount,
						selected: selectedCount,
					})}
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
				{t("portfolio.screenshotModal.rowCount", { count: selectedCount })}
			</Typography.Text>
		</Space>
	);
}
