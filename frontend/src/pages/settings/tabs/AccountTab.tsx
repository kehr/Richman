import { gravatarUrl } from "@/domain/auth/gravatar";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import { useExchangeRates } from "@/domain/money/useExchangeRates";
import { useLogout } from "@/features/auth";
import {
	type DisplayCurrency,
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
	Select,
	Space,
	Tooltip,
	Typography,
	UserOutlined,
	message,
} from "@/ui-kit/eat";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

interface CapitalFormValues {
	amount?: number | null;
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
	const { rates } = useExchangeRates();

	const [capitalForm] = Form.useForm<CapitalFormValues>();
	// inputCurrency controls the unit shown next to the capital input.
	// Defaults to the user's displayCurrency preference; falls back to CNY when
	// the required exchange rate is unavailable.
	const [inputCurrency, setInputCurrency] = useState<DisplayCurrency>("CNY");

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

	// Sync inputCurrency to the user's displayCurrency preference on first load.
	const displayCurrencyPref = settings?.displayCurrency;
	useEffect(() => {
		if (displayCurrencyPref) {
			setInputCurrency(displayCurrencyPref);
		}
	}, [displayCurrencyPref]);

	// Re-derive form value whenever inputCurrency, rates, or stored capital changes.
	// Converts the stored CNY amount to the currently selected input unit.
	useEffect(() => {
		const cny = settings?.totalCapitalCny;
		let displayValue: number | undefined;
		if (cny != null) {
			if (inputCurrency === "CNY") {
				displayValue = Math.round(cny);
			} else {
				const rate = rates[inputCurrency];
				displayValue = rate ? Math.round(cny * rate) : Math.round(cny);
			}
		}
		capitalForm.setFieldsValue({ amount: displayValue });
	}, [capitalForm, settings?.totalCapitalCny, inputCurrency, rates]);

	const handleCurrencyChange = (currency: DisplayCurrency) => {
		setInputCurrency(currency);
		// The effect above will update the form value from the stored CNY amount.
	};

	const handleSaveCapital = async () => {
		try {
			const values = await capitalForm.validateFields();
			const raw = values.amount;
			if (raw == null) {
				await patchMutation.mutateAsync({ clearTotalCapitalCny: true });
			} else {
				let cnyValue: number;
				if (inputCurrency === "CNY") {
					cnyValue = raw;
				} else {
					const rate = rates[inputCurrency];
					if (!rate) {
						message.error(t("account.message.rateUnavailable"));
						return;
					}
					// rate = "1 CNY = X foreign", so foreign → CNY = amount / rate
					cnyValue = Math.round(raw / rate);
				}
				await patchMutation.mutateAsync({ totalCapitalCny: cnyValue });
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

	const currencySelector = (
		<Select
			value={inputCurrency}
			onChange={handleCurrencyChange}
			options={[
				{ label: "CNY", value: "CNY" },
				{ label: "USD", value: "USD", disabled: !rates.USD },
				{ label: "HKD", value: "HKD", disabled: !rates.HKD },
			]}
			style={{ width: 70 }}
		/>
	);

	return (
		<Flex vertical gap={24} data-testid="account-tab">
			<Flex align="center" gap={16} data-testid="account-avatar-section">
				<Avatar src={gravatarUrl(email, 64)} icon={<UserOutlined />} size={64} draggable={false} />
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
					<Form.Item name="amount" style={{ marginBottom: 0 }}>
						<InputNumber
							min={0}
							step={inputCurrency === "CNY" ? 1000 : 100}
							style={{ width: 240 }}
							addonAfter={currencySelector}
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
