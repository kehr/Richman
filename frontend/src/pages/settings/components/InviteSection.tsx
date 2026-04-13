import { useMyCodesQuery, useMyInvitesQuery } from "@/features/invite";
import {
	App,
	Badge,
	Button,
	CalendarOutlined,
	CopyOutlined,
	Divider,
	FireOutlined,
	Flex,
	List,
	Skeleton,
	Tag,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// InviteSection shows the user's personal invite codes, copy-to-clipboard
// functionality, unlock progress bar, and the list of invited users.
export function InviteSection() {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const codesQuery = useMyCodesQuery();
	const invitesQuery = useMyInvitesQuery();

	const handleCopy = async (code: string) => {
		try {
			await navigator.clipboard.writeText(code);
			message.success(t("invite.copySuccess"));
		} catch {
			message.error(t("invite.copyError"));
		}
	};

	const codes = codesQuery.data?.codes ?? [];
	const nextUnlockIn = codesQuery.data?.nextUnlockIn ?? 0;
	const totalCodes = codesQuery.data?.totalCodes ?? 0;
	const usedCount = codesQuery.data?.usedCount ?? 0;

	const invites = invitesQuery.data?.invites ?? [];
	const totalInvited = invitesQuery.data?.totalInvited ?? 0;

	return (
		<Flex vertical gap={24} data-testid="invite-section">
			<Flex vertical gap={8}>
				<Typography.Text strong>{t("invite.myCodesTitle")}</Typography.Text>
				<Typography.Text type="secondary" style={{ fontSize: 13 }}>
					{t("invite.myCodesDesc", { total: totalCodes, used: usedCount })}
				</Typography.Text>
			</Flex>

			{codesQuery.isLoading ? (
				<Skeleton active paragraph={{ rows: 3 }} />
			) : (
				<Flex vertical gap={8}>
					{codes.map((item) => (
						<Flex
							key={item.code}
							align="center"
							gap={12}
							style={{
								padding: "10px 14px",
								borderRadius: 8,
								background: item.isUsed ? "#fafafa" : "#f6ffed",
								border: `1px solid ${item.isUsed ? "#f0f0f0" : "#b7eb8f"}`,
							}}
						>
							<Typography.Text
								style={{
									fontFamily: "monospace",
									fontSize: 15,
									fontWeight: 600,
									letterSpacing: "0.08em",
									color: item.isUsed ? "#999" : "#333",
									flex: 1,
								}}
							>
								{item.code}
							</Typography.Text>

							{item.isUsed ? (
								<Tag color="default">{t("invite.codeUsed")}</Tag>
							) : (
								<Tag color="success">{t("invite.codeAvailable")}</Tag>
							)}

							<Tooltip title={item.isUsed ? t("invite.codeAlreadyUsed") : t("invite.copy")}>
								<Button
									type="text"
									size="small"
									icon={<CopyOutlined />}
									disabled={item.isUsed}
									onClick={() => handleCopy(item.code)}
									data-testid={`invite-copy-${item.code}`}
								/>
							</Tooltip>
						</Flex>
					))}
				</Flex>
			)}

			{/* Unlock progress */}
			<Flex
				align="center"
				gap={8}
				style={{
					padding: "10px 14px",
					borderRadius: 8,
					background: "#fffbe6",
					border: "1px solid #ffe58f",
				}}
			>
				<FireOutlined style={{ color: "#fa8c16", fontSize: 16 }} />
				<Typography.Text style={{ fontSize: 13 }}>
					{nextUnlockIn > 0
						? t("invite.unlockProgress", { days: nextUnlockIn })
						: t("invite.unlockReady")}
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			{/* Invited users list */}
			<Flex vertical gap={8}>
				<Typography.Text strong>
					{t("invite.invitedUsersTitle", { count: totalInvited })}
				</Typography.Text>

				{invitesQuery.isLoading ? (
					<Skeleton active paragraph={{ rows: 2 }} />
				) : invites.length === 0 ? (
					<Typography.Text type="secondary" style={{ fontSize: 13 }}>
						{t("invite.noInvitesYet")}
					</Typography.Text>
				) : (
					<List
						size="small"
						dataSource={invites}
						renderItem={(invite) => (
							<List.Item style={{ padding: "8px 0" }}>
								<Flex align="center" gap={8} style={{ width: "100%" }}>
									<Typography.Text>{invite.invitedUserName}</Typography.Text>
									<Flex align="center" gap={4} style={{ marginLeft: "auto" }}>
										<CalendarOutlined style={{ color: "#8c8c8c", fontSize: 12 }} />
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											{new Date(invite.invitedAt).toLocaleDateString()}
										</Typography.Text>
									</Flex>
								</Flex>
							</List.Item>
						)}
					/>
				)}
			</Flex>
		</Flex>
	);
}
