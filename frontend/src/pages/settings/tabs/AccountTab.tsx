import { gravatarUrl } from "@/domain/auth/gravatar";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import { useLogout } from "@/features/auth";
import {
	type RiskPreference,
	usePatchUserSettings,
	useResetOnboarding,
	useUserSettings,
} from "@/features/user-settings";
import {
	Avatar,
	Button,
	Divider,
	Flex,
	Form,
	InputNumber,
	Popconfirm,
	Radio,
	Space,
	Tooltip,
	Typography,
	UserOutlined,
	message,
} from "@/ui-kit/eat";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

interface CapitalFormValues {
	totalCapitalCny?: number | null;
}

// AccountTab renders the PRD §6.2 fields: read-only email, password reset
// (disabled placeholder), total capital with privacy hint, risk preference
// dropdown, logout, and a "redo onboarding" CTA that clears both the server
// onboarding flags and the local nudge dismissal state before navigating to
// the wizard. The CTA ships to production (no dev-only gate) because users
// who dismissed the Dashboard nudge would otherwise have no regret path.
export function AccountTab() {
	const { t } = useTranslation("settings");
	const settingsQuery = useUserSettings();
	const patchMutation = usePatchUserSettings();
	const resetOnboarding = useResetOnboarding();
	const logout = useLogout();
	const currentUser = useCurrentUser();
	const navigate = useNavigate();

	const [capitalForm] = Form.useForm<CapitalFormValues>();

	const settings = settingsQuery.data;
	const email = currentUser.data?.email ?? "";
	const displayName = email ? email.split("@")[0] : "—";

	const riskOptions = useMemo(
		() => [
			{ label: t("account.riskOptions.conservative"), value: "conservative" as RiskPreference },
			{ label: t("account.riskOptions.neutral"), value: "neutral" as RiskPreference },
			{ label: t("account.riskOptions.aggressive"), value: "aggressive" as RiskPreference },
		],
		[t],
	);

	// Sync form initial value with the loaded settings snapshot. We rely on
	// setFieldsValue rather than initialValues so the form picks up the
	// current capital after the query resolves.
	useEffect(() => {
		if (settings) {
			capitalForm.setFieldsValue({ totalCapitalCny: settings.totalCapitalCny ?? undefined });
		}
	}, [capitalForm, settings]);

	const handleSaveCapital = async () => {
		try {
			const values = await capitalForm.validateFields();
			const raw = values.totalCapitalCny;
			if (raw == null) {
				await patchMutation.mutateAsync({ clearTotalCapitalCny: true });
			} else {
				await patchMutation.mutateAsync({ totalCapitalCny: raw });
			}
			message.success(t("account.message.capitalSaved"));
		} catch (err) {
			// antd throws an object with errorFields when validation fails; we
			// ignore that case because the form already renders per-field errors.
			if (err && typeof err === "object" && "errorFields" in err) return;
			message.error(t("account.message.capitalSaveError"));
		}
	};

	const handleRiskChange = async (value: RiskPreference) => {
		try {
			await patchMutation.mutateAsync({ riskPreference: value });
			message.success(t("account.message.riskUpdated"));
		} catch {
			message.error(t("account.message.riskUpdateError"));
		}
	};

	const handleResetOnboarding = async () => {
		try {
			await resetOnboarding.mutateAsync();
			// useResetOnboarding's onSuccess already clears the sessionStorage
			// draft, the localStorage nudge-dismissed flag, and refetches
			// onboarding status — we just need to navigate once the mutation
			// has settled so the guard sees the fresh not-completed status.
			navigate("/onboarding/welcome");
		} catch {
			message.error(t("account.message.resetError"));
		}
	};

	return (
		<Flex vertical gap={24} data-testid="account-tab">
			<Flex align="center" gap={16} data-testid="account-avatar-section">
				<Avatar src={gravatarUrl(email, 64)} icon={<UserOutlined />} size={64} />
				<Flex vertical gap={4}>
					<Typography.Text strong>{displayName}</Typography.Text>
					<Typography.Text type="secondary">{email}</Typography.Text>
					<Typography.Link
						href="https://gravatar.com"
						target="_blank"
						rel="noopener noreferrer"
						style={{ fontSize: 12 }}
					>
						{t("account.avatar.changeLink")}
					</Typography.Link>
				</Flex>
			</Flex>

			<Space>
				<Tooltip title={t("account.changePasswordTooltip")}>
					<Button disabled data-testid="account-change-password">
						{t("account.changePassword")}
					</Button>
				</Tooltip>
			</Space>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">{t("account.totalCapital")}</Typography.Text>
				<Form<CapitalFormValues> form={capitalForm} layout="inline">
					<Form.Item name="totalCapitalCny" style={{ marginBottom: 0 }}>
						<InputNumber
							min={0}
							step={1000}
							style={{ width: 240 }}
							addonAfter="CNY"
							placeholder={t("account.totalCapitalPlaceholder")}
							data-testid="account-total-capital-input"
						/>
					</Form.Item>
					<Button
						type="primary"
						loading={patchMutation.isPending}
						onClick={handleSaveCapital}
						data-testid="account-total-capital-save"
					>
						{t("action.save", { ns: "common" })}
					</Button>
				</Form>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("account.totalCapitalHint")}
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">{t("account.riskPreference")}</Typography.Text>
				<Radio.Group
					value={settings?.riskPreference}
					onChange={(e) => handleRiskChange(e.target.value as RiskPreference)}
					options={riskOptions}
					disabled={settingsQuery.isLoading}
					data-testid="account-risk-preference"
				/>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("account.riskPreferenceHint")}
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex gap={12} align="center">
				<Button danger onClick={logout} data-testid="account-logout">
					{t("account.logout")}
				</Button>
				<Popconfirm
					title={t("account.resetOnboardingConfirm.title")}
					description={t("account.resetOnboardingConfirm.description")}
					okText={t("account.resetOnboardingConfirm.ok")}
					cancelText={t("account.resetOnboardingConfirm.cancel")}
					onConfirm={handleResetOnboarding}
				>
					<Button loading={resetOnboarding.isPending} data-testid="account-reset-onboarding">
						{t("account.resetOnboarding")}
					</Button>
				</Popconfirm>
			</Flex>
		</Flex>
	);
}
