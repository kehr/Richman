import { Badge, Button, Card, Dropdown, EllipsisOutlined, Popconfirm, theme } from "@/ui-kit/eat";
import type { MenuProps } from "@/ui-kit/eat";
import { BrainCircuit } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { LLMSettingsDTO } from "./types";
import { formatRelativeTime } from "./utils/formatRelativeTime";

// ProviderCardLayout is the shared three-section card skeleton used by
// LLMHealthyCard and LLMFailingCard. It is intentionally NOT exported from
// the feature barrel (index.ts) — it is an internal implementation detail.

interface ProviderCardLayoutProps {
	providerType: LLMSettingsDTO["providerType"];
	lastProbeAt: string | null | undefined;
	healthStatus: "healthy" | "failing" | "unknown";
	onDelete: () => Promise<void>;
	isDeleting: boolean;
	bodyContent: ReactNode;
	footerContent: ReactNode;
	"data-testid"?: string;
}

const PROVIDER_BADGE_STYLE: Record<
	NonNullable<LLMSettingsDTO["providerType"]>,
	{ bg: string; border: string; color: string }
> = {
	claude: { bg: "#f0f5ff", border: "#d6e4ff", color: "#2f54eb" },
	openai: { bg: "#f6ffed", border: "#d9f7be", color: "#389e0d" },
	openai_compatible: { bg: "#f9f0ff", border: "#efdbff", color: "#531dab" },
};

const FAILING_BADGE_OVERRIDE = { bg: "#fff2f0", border: "#ffccc7", color: "#cf1322" };

const FALLBACK_BADGE_STYLE = { bg: "#f5f5f5", border: "#d9d9d9", color: "#595959" };

// Badge status values mirror antd PresetStatusColorType
type BadgeStatus = "success" | "processing" | "error" | "default" | "warning";

const BADGE_STATUS: Record<string, BadgeStatus> = {
	healthy: "success",
	failing: "error",
	unknown: "default",
};

// providerLabel maps the provider enum to a human-readable brand label.
// These are proper nouns / brand names and intentionally not translated.
function providerLabel(providerType: LLMSettingsDTO["providerType"]): string {
	switch (providerType) {
		case "claude":
			return "Claude (Anthropic)";
		case "openai":
			return "OpenAI";
		case "openai_compatible":
			return "OpenAI Compatible";
		default:
			return "Unknown";
	}
}

export function ProviderCardLayout({
	providerType,
	lastProbeAt,
	healthStatus,
	onDelete,
	isDeleting,
	bodyContent,
	footerContent,
	"data-testid": dataTestId,
}: ProviderCardLayoutProps) {
	const { t, i18n } = useTranslation("settings");
	const { token } = theme.useToken();

	// Compute badge style, applying the failing override when needed.
	const baseBadge =
		providerType != null ? PROVIDER_BADGE_STYLE[providerType] : FALLBACK_BADGE_STYLE;
	const badge =
		healthStatus === "failing"
			? {
					...baseBadge,
					bg: FAILING_BADGE_OVERRIDE.bg,
					border: FAILING_BADGE_OVERRIDE.border,
					color: FAILING_BADGE_OVERRIDE.color,
				}
			: baseBadge;

	const badgeText = (() => {
		if (healthStatus === "healthy") return t("llm.healthyCard.healthy");
		if (healthStatus === "failing") return t("llm.failingCard.failing");
		return t("llm.failingCard.unknown");
	})();

	const dropdownItems: MenuProps["items"] = [
		{
			key: "delete",
			danger: true,
			label: (
				<Popconfirm
					title={t("llm.healthyCard.deleteConfirm.title")}
					description={t("llm.healthyCard.deleteConfirm.description")}
					okText={t("llm.healthyCard.deleteConfirm.ok")}
					cancelText={t("llm.healthyCard.deleteConfirm.cancel")}
					okButtonProps={{ danger: true, loading: isDeleting }}
					onConfirm={onDelete}
				>
					<span>{t("llm.healthyCard.deleteMenuLabel")}</span>
				</Popconfirm>
			),
		},
	];

	return (
		<Card styles={{ body: { padding: 0 } }} data-testid={dataTestId}>
			{/* Header */}
			<div
				style={{
					padding: "16px 20px",
					borderBottom: `1px solid ${token.colorBorderSecondary}`,
				}}
			>
				<div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
					{/* Left side: provider badge + name + time */}
					<div style={{ display: "flex", alignItems: "center", gap: 10 }}>
						<div
							style={{
								width: 32,
								height: 32,
								borderRadius: 6,
								background: badge.bg,
								border: `1px solid ${badge.border}`,
								display: "flex",
								alignItems: "center",
								justifyContent: "center",
								color: badge.color,
								flexShrink: 0,
							}}
						>
							<BrainCircuit size={16} />
						</div>
						<div>
							<div style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
								{providerLabel(providerType)}
							</div>
							<div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 1 }}>
								{t("llm.healthyCard.lastProbedAt", {
									time: formatRelativeTime(lastProbeAt, i18n.language),
								})}
							</div>
						</div>
					</div>
					{/* Right side: health badge + dropdown */}
					<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
						<Badge status={BADGE_STATUS[healthStatus] ?? "default"} text={badgeText} />
						<Dropdown menu={{ items: dropdownItems }} trigger={["click"]}>
							<Button type="text" icon={<EllipsisOutlined />} size="small" />
						</Dropdown>
					</div>
				</div>
			</div>
			{/* Body */}
			<div style={{ padding: "16px 20px" }}>{bodyContent}</div>
			{/* Footer */}
			<div
				style={{
					padding: "12px 20px",
					borderTop: `1px solid ${token.colorBorderSecondary}`,
				}}
			>
				{footerContent}
			</div>
		</Card>
	);
}
