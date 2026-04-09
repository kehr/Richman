import { Skeleton, Space, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LLMConfigForm } from "./LLMConfigForm";
import { LLMEmptyState } from "./LLMEmptyState";
import { LLMFailingCard } from "./LLMFailingCard";
import { LLMHealthyCard } from "./LLMHealthyCard";
import { useLLMSettings } from "./hooks";

const { Title, Paragraph } = Typography;

interface LLMSectionProps {
	// systemDefaultAvailable is supplied by the composing page from the
	// dashboard-summary feature. The settings-llm feature does not import
	// from dashboard-summary directly because the dependency-cruiser forbids
	// cross-feature imports; instead the page wires the two together.
	systemDefaultAvailable: boolean;
}

// LLMSection is the top-level container for the AI interpretation settings
// tab. It owns the mode-switching between Empty / Healthy / Failing cards and
// the add/edit modal lifecycle. All API interaction is delegated to the
// hooks exported by the feature.
export function LLMSection({ systemDefaultAvailable }: LLMSectionProps) {
	const { t } = useTranslation("settings");
	const query = useLLMSettings();
	const [isFormOpen, setIsFormOpen] = useState(false);

	const openAdd = () => setIsFormOpen(true);
	const openEdit = () => setIsFormOpen(true);
	const closeForm = () => setIsFormOpen(false);

	if (query.isLoading) {
		return (
			<Space direction="vertical" size={16} style={{ width: "100%" }} data-testid="llm-section">
				<Title level={4} style={{ margin: 0 }}>
					{t("llm.title")}
				</Title>
				<Skeleton active paragraph={{ rows: 3 }} />
			</Space>
		);
	}

	const data = query.data;
	const configured = Boolean(data?.configured);
	const health = data?.healthStatus ?? "unknown";
	const mode = configured ? "edit" : "create";

	return (
		<Space direction="vertical" size={16} style={{ width: "100%" }} data-testid="llm-section">
			<div>
				<Title level={4} style={{ margin: 0 }}>
					{t("llm.title")}
				</Title>
				<Paragraph type="secondary" style={{ marginBottom: 0, marginTop: 4 }}>
					{t("llm.description")}
				</Paragraph>
			</div>

			{!configured && data && (
				<LLMEmptyState
					systemDefaultAvailable={systemDefaultAvailable}
					useSystemDefaultConsent={data.useSystemDefaultWhenUnconfigured}
					onAddProvider={openAdd}
				/>
			)}

			{configured && data && health !== "failing" && (
				<LLMHealthyCard config={data} onEdit={openEdit} />
			)}

			{configured && data && health === "failing" && (
				<LLMFailingCard
					config={data}
					systemDefaultAvailable={systemDefaultAvailable}
					onEdit={openEdit}
				/>
			)}

			<LLMConfigForm open={isFormOpen} mode={mode} initialValue={data} onClose={closeForm} />
		</Space>
	);
}
