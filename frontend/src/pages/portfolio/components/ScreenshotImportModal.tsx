import {
	type EditableRecognizedHolding,
	type RecognizedHolding,
	useCreateHolding,
	useScreenshotImport,
} from "@/features/portfolio";
import {
	Alert,
	Button,
	Col,
	InboxOutlined,
	Modal,
	Row,
	Space,
	Spin,
	Typography,
	Upload,
	type UploadProps,
	message,
} from "@/ui-kit/eat";
import { useEffect, useMemo, useState } from "react";
import { ImagePreview } from "./ImagePreview";
import { RecognizedHoldingTable } from "./RecognizedHoldingTable";

// ScreenshotImportModal implements the bulk screenshot import flow from
// PRD §4.3. It is opened from the PortfolioListPage top button and renders a
// full-screen Modal with four states:
//
//   initial-upload  -> antd Upload.Dragger zone
//   recognizing     -> centered spinner while the vision model runs
//   recognized      -> dual-pane preview/table for the user to confirm
//   failed          -> error banner with a retry button
//
// On confirm we walk the selected rows sequentially (for/await) and call
// the existing POST /holdings endpoint per row. We deliberately stop on the
// first failure and surface a "成功 X / 失败 Y" message; there is no rollback
// because the holdings table has no DELETE counterpart and the user can
// re-run the modal after fixing the offending row.

interface ScreenshotImportModalProps {
	open: boolean;
	onClose: () => void;
	currentHoldingCount: number;
	holdingLimit: number;
}

type Phase = "initial-upload" | "recognizing" | "recognized" | "failed";

let rowIdSeed = 0;
function nextRowId(): string {
	rowIdSeed += 1;
	return `recognized-${Date.now()}-${rowIdSeed}`;
}

function parseNumber(raw: string): number | null {
	if (!raw) return null;
	// Strip spaces, ¥/$ and trailing % so the LLM's free-form value lands as
	// a parseable number. We keep the value null if parsing fails so the row
	// renders the "请手动填写" placeholder.
	const cleaned = raw.replace(/[¥$,\s%]/g, "");
	const n = Number(cleaned);
	return Number.isFinite(n) ? n : null;
}

function toEditableRows(
	holdings: RecognizedHolding[],
	currentHoldingCount: number,
	holdingLimit: number,
): EditableRecognizedHolding[] {
	const remaining = Math.max(0, holdingLimit - currentHoldingCount);
	return holdings.map((h, idx) => ({
		rowId: nextRowId(),
		assetName: h.assetName.value,
		assetNameConfidence: h.assetName.confidence,
		assetCode: h.assetCode.value,
		assetCodeConfidence: h.assetCode.confidence,
		costPrice: parseNumber(h.costPrice.value),
		costPriceConfidence: h.costPrice.confidence,
		positionRatio: parseNumber(h.positionPct.value),
		positionRatioConfidence: h.positionPct.confidence,
		assetType: h.assetTypeGuess || "a_share_broad",
		// Pre-check rows up to the cap; rows beyond the cap default off.
		selected: idx < remaining,
	}));
}

export function ScreenshotImportModal({
	open,
	onClose,
	currentHoldingCount,
	holdingLimit,
}: ScreenshotImportModalProps) {
	const [phase, setPhase] = useState<Phase>("initial-upload");
	const [previewUrl, setPreviewUrl] = useState<string | null>(null);
	const [rows, setRows] = useState<EditableRecognizedHolding[]>([]);
	const [warning, setWarning] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	const screenshotImport = useScreenshotImport();
	const createHolding = useCreateHolding();

	useEffect(() => {
		if (open) return;
		// Always reset internal state when the modal closes so the next
		// open begins from the upload step.
		setPhase("initial-upload");
		setRows([]);
		setWarning(null);
		setSubmitting(false);
		setPreviewUrl((prev) => {
			if (prev) URL.revokeObjectURL(prev);
			return null;
		});
	}, [open]);

	const selectedRows = useMemo(() => rows.filter((r) => r.selected), [rows]);
	const remainingSlots = Math.max(0, holdingLimit - currentHoldingCount);

	const handleUpload = async (file: File) => {
		if (previewUrl) URL.revokeObjectURL(previewUrl);
		setPreviewUrl(URL.createObjectURL(file));
		setPhase("recognizing");
		setWarning(null);
		try {
			const result = await screenshotImport.mutateAsync(file);
			if (result.overallStatus === "failed" || result.holdings.length === 0) {
				setWarning(result.warning || "识别失败，请换一张截图重试");
				setPhase("failed");
				return;
			}
			setRows(toEditableRows(result.holdings, currentHoldingCount, holdingLimit));
			setWarning(result.warning || null);
			setPhase("recognized");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "识别失败";
			setWarning(msg);
			setPhase("failed");
		}
	};

	const draggerProps: UploadProps = {
		accept: "image/png,image/jpeg,image/webp",
		showUploadList: false,
		multiple: false,
		// Returning false tells antd to skip its built-in XHR upload so we can
		// drive the request through our own mutation hook.
		beforeUpload: (file) => {
			void handleUpload(file as File);
			return false;
		},
	};

	const handleRowChange = (rowId: string, patch: Partial<EditableRecognizedHolding>) => {
		setRows((prev) => prev.map((r) => (r.rowId === rowId ? { ...r, ...patch } : r)));
	};

	const handleRowDelete = (rowId: string) => {
		setRows((prev) => prev.filter((r) => r.rowId !== rowId));
	};

	const validateBeforeConfirm = (): string | null => {
		if (selectedRows.length === 0) return "请至少选择一个识别结果";
		if (selectedRows.length > remainingSlots) {
			return `最多再添加 ${remainingSlots} 个标的`;
		}
		for (const row of selectedRows) {
			if (!row.assetName.trim() || !row.assetCode.trim()) {
				return "请补全名称和代码";
			}
			if (row.costPrice == null || row.positionRatio == null) {
				return "请补全成本和仓位";
			}
		}
		return null;
	};

	const handleConfirm = async () => {
		const validationError = validateBeforeConfirm();
		if (validationError) {
			message.warning(validationError);
			return;
		}
		setSubmitting(true);
		let success = 0;
		let failure = 0;
		// Sequential bulk POST: stop on first failure so the user can review
		// the offending row before re-running. There is no rollback because
		// holdings has no DELETE counterpart in the bulk path.
		for (const row of selectedRows) {
			try {
				await createHolding.mutateAsync({
					assetCode: row.assetCode.trim(),
					assetName: row.assetName.trim(),
					assetType: row.assetType,
					// biome-ignore lint/style/noNonNullAssertion: validated above
					costPrice: row.costPrice!,
					// biome-ignore lint/style/noNonNullAssertion: validated above
					positionRatio: row.positionRatio!,
				});
				success += 1;
			} catch {
				failure += 1;
				break;
			}
		}
		setSubmitting(false);
		if (failure > 0) {
			message.error(`成功 ${success} / 失败 ${failure}`);
		} else {
			message.success(`成功导入 ${success} 个持仓`);
			onClose();
		}
	};

	const renderBody = () => {
		if (phase === "initial-upload") {
			return (
				<Upload.Dragger {...draggerProps} data-testid="screenshot-upload-dragger">
					<p className="ant-upload-drag-icon">
						<InboxOutlined />
					</p>
					<p className="ant-upload-text">点击或拖拽截图到此区域上传</p>
					<p className="ant-upload-hint">支持单张 PNG / JPEG / WebP，最大 5 MB</p>
				</Upload.Dragger>
			);
		}
		if (phase === "recognizing") {
			return (
				<div
					data-testid="screenshot-recognizing"
					style={{ display: "flex", justifyContent: "center", padding: 80 }}
				>
					<Spin tip="正在识别截图..." size="large" />
				</div>
			);
		}
		if (phase === "failed") {
			return (
				<Space direction="vertical" size="middle" style={{ width: "100%" }}>
					<Alert
						type="error"
						showIcon
						message={warning || "识别失败"}
						data-testid="screenshot-failed-alert"
					/>
					<Button
						type="primary"
						onClick={() => {
							setPhase("initial-upload");
							setWarning(null);
						}}
					>
						重新上传
					</Button>
				</Space>
			);
		}
		return (
			<Space direction="vertical" size="middle" style={{ width: "100%" }}>
				{warning && (
					<Alert type="warning" showIcon message={warning} data-testid="screenshot-warning" />
				)}
				<Row gutter={16}>
					<Col span={9}>{previewUrl && <ImagePreview src={previewUrl} />}</Col>
					<Col span={15}>
						<RecognizedHoldingTable
							rows={rows}
							currentHoldingCount={currentHoldingCount}
							holdingLimit={holdingLimit}
							onChange={handleRowChange}
							onDelete={handleRowDelete}
						/>
					</Col>
				</Row>
			</Space>
		);
	};

	return (
		<Modal
			open={open}
			onCancel={onClose}
			width={1100}
			title={
				<div
					data-testid="screenshot-modal-header"
					style={{
						background: "#1f1f1f",
						color: "#fff",
						padding: "12px 16px",
						margin: "-20px -24px 0",
						borderRadius: "8px 8px 0 0",
					}}
				>
					<Typography.Text strong style={{ color: "#fff", fontSize: 16 }}>
						截图识别结果 — 校对
					</Typography.Text>
					{phase === "recognized" && (
						<Typography.Text style={{ color: "#fff", marginLeft: 12 }}>
							识别出 {rows.length} 个标的 · 请检查高亮字段后确认导入
						</Typography.Text>
					)}
				</div>
			}
			footer={
				phase === "recognized" ? (
					<Space>
						<Button onClick={onClose}>取消</Button>
						<Button
							type="primary"
							loading={submitting}
							onClick={handleConfirm}
							data-testid="screenshot-confirm-button"
						>
							确认导入
						</Button>
					</Space>
				) : (
					<Button onClick={onClose}>取消</Button>
				)
			}
			destroyOnClose
			data-testid="screenshot-import-modal"
		>
			{renderBody()}
		</Modal>
	);
}
