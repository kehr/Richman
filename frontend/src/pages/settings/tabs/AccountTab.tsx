import { useCurrentUser } from "@/domain/auth/use-current-user";
import { useLogout } from "@/features/auth";
import {
	type RiskPreference,
	usePatchUserSettings,
	useResetOnboarding,
	useUserSettings,
} from "@/features/user-settings";
import {
	Button,
	Divider,
	Flex,
	Form,
	InputNumber,
	Select,
	Space,
	Tooltip,
	Typography,
	message,
} from "@/ui-kit/eat";
import { useEffect } from "react";

const RISK_OPTIONS: { label: string; value: RiskPreference }[] = [
	{ label: "稳健", value: "conservative" },
	{ label: "中性", value: "neutral" },
	{ label: "激进", value: "aggressive" },
];

interface CapitalFormValues {
	totalCapitalCny?: number | null;
}

// AccountTab renders the PRD §6.2 fields: read-only email, password reset
// (disabled placeholder), total capital with privacy hint, risk preference
// dropdown, logout, and a dev-only "reset onboarding" affordance.
export function AccountTab() {
	const settingsQuery = useUserSettings();
	const patchMutation = usePatchUserSettings();
	const resetOnboarding = useResetOnboarding();
	const logout = useLogout();
	const currentUser = useCurrentUser();

	const [capitalForm] = Form.useForm<CapitalFormValues>();

	const settings = settingsQuery.data;
	const email = currentUser.data?.email ?? "—";

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
			message.success("总资金已保存");
		} catch (err) {
			// antd throws an object with errorFields when validation fails; we
			// ignore that case because the form already renders per-field errors.
			if (err && typeof err === "object" && "errorFields" in err) return;
			message.error("保存总资金失败");
		}
	};

	const handleRiskChange = async (value: RiskPreference) => {
		try {
			await patchMutation.mutateAsync({ riskPreference: value });
			message.success("风险偏好已更新");
		} catch {
			message.error("更新风险偏好失败");
		}
	};

	const handleResetOnboarding = async () => {
		try {
			await resetOnboarding.mutateAsync();
			message.success("Onboarding 已重置，下次进入将重新引导");
		} catch {
			message.error("重置 Onboarding 失败");
		}
	};

	const isDev = import.meta.env.DEV === true;

	return (
		<Flex vertical gap={24} data-testid="account-tab">
			<Flex vertical gap={4}>
				<Typography.Text type="secondary">邮箱</Typography.Text>
				<Typography.Text strong data-testid="account-email">
					{email}
				</Typography.Text>
				<Space style={{ marginTop: 8 }}>
					<Tooltip title="修改密码接口待后端补齐">
						<Button disabled data-testid="account-change-password">
							发送修改链接到邮箱
						</Button>
					</Tooltip>
				</Space>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">总资金</Typography.Text>
				<Form<CapitalFormValues> form={capitalForm} layout="inline">
					<Form.Item name="totalCapitalCny" style={{ marginBottom: 0 }}>
						<InputNumber
							min={0}
							step={1000}
							style={{ width: 240 }}
							addonAfter="CNY"
							placeholder="例如 100000"
							data-testid="account-total-capital-input"
						/>
					</Form.Item>
					<Button
						type="primary"
						loading={patchMutation.isPending}
						onClick={handleSaveCapital}
						data-testid="account-total-capital-save"
					>
						保存
					</Button>
				</Form>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					总资金仅本地保存用于金额换算，不会进入 LLM 分析上下文。
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">风险偏好</Typography.Text>
				<Select<RiskPreference>
					value={settings?.riskPreference}
					onChange={handleRiskChange}
					options={RISK_OPTIONS}
					style={{ width: 240 }}
					loading={settingsQuery.isLoading}
					data-testid="account-risk-preference"
				/>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					影响 LLM 在权重微调范围内的倾向。
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex gap={12} align="center">
				<Button danger onClick={logout} data-testid="account-logout">
					退出登录
				</Button>
				{isDev && (
					<Button
						onClick={handleResetOnboarding}
						loading={resetOnboarding.isPending}
						data-testid="account-reset-onboarding"
					>
						重置 Onboarding（dev）
					</Button>
				)}
			</Flex>
		</Flex>
	);
}
