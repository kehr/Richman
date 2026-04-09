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
	Popconfirm,
	Select,
	Space,
	Tooltip,
	Typography,
	message,
} from "@/ui-kit/eat";
import { useEffect } from "react";
import { useNavigate } from "react-router";

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
// dropdown, logout, and a "重新走一遍引导" CTA that clears both the server
// onboarding flags and the local nudge dismissal state before navigating to
// the wizard. The CTA ships to production (no dev-only gate) because users
// who dismissed the Dashboard nudge would otherwise have no regret path.
export function AccountTab() {
	const settingsQuery = useUserSettings();
	const patchMutation = usePatchUserSettings();
	const resetOnboarding = useResetOnboarding();
	const logout = useLogout();
	const currentUser = useCurrentUser();
	const navigate = useNavigate();

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
			// useResetOnboarding's onSuccess already clears the sessionStorage
			// draft, the localStorage nudge-dismissed flag, and refetches
			// onboarding status — we just need to navigate once the mutation
			// has settled so the guard sees the fresh not-completed status.
			navigate("/onboarding/welcome");
		} catch {
			message.error("重置失败，请稍后重试");
		}
	};

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
				<Popconfirm
					title="重新走一遍引导？"
					description="将清空本地引导草稿并让 Dashboard 的引导提示重新出现，当前持仓和决策卡不受影响。"
					okText="开始引导"
					cancelText="取消"
					onConfirm={handleResetOnboarding}
				>
					<Button loading={resetOnboarding.isPending} data-testid="account-reset-onboarding">
						重新走一遍引导
					</Button>
				</Popconfirm>
			</Flex>
		</Flex>
	);
}
