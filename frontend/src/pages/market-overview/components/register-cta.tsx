import { Button, Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

const { Text } = Typography;
const { useToken } = theme;

// RegisterCTA is the bottom fixed strip shown only to unauthenticated visitors.
// It prompts them to register to unlock personalized investment plans.
export function RegisterCTA() {
	const { t } = useTranslation("market");
	const { token } = useToken();
	const navigate = useNavigate();

	return (
		<div
			style={{
				position: "fixed",
				bottom: 0,
				left: 0,
				right: 0,
				zIndex: 100,
				background: token.colorBgContainer,
				borderTop: `1px solid ${token.colorBorder}`,
				padding: "12px 24px",
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
				gap: 16,
				boxShadow: "0 -2px 8px rgba(0,0,0,0.06)",
			}}
		>
			<Text style={{ fontSize: 13, color: token.colorText }}>{t("overview.registerCta.text")}</Text>
			<Button type="primary" size="small" onClick={() => navigate("/register")}>
				{t("overview.registerCta.button")}
			</Button>
		</div>
	);
}
