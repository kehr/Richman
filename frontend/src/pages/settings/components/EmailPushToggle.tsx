import { requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import { App, Flex, MailOutlined, Switch, Typography } from "@/ui-kit/eat";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

// EmailPushPrefs mirrors the backend response for GET/PATCH /api/v2/user/email-push.
interface EmailPushPrefs {
	emailPushEnabled: boolean;
}

const EMAIL_PUSH_QUERY_KEY = ["user", "email-push"] as const;

function getEmailPush() {
	return requestV2<ApiResponse<EmailPushPrefs>>("/user/email-push");
}

function patchEmailPush(enabled: boolean) {
	return requestV2<ApiResponse<EmailPushPrefs>>("/user/email-push", {
		method: "PATCH",
		body: JSON.stringify({ emailPushEnabled: enabled }),
	});
}

// EmailPushToggle renders a labeled switch that controls whether the platform
// sends email digest notifications to the user. Calls PATCH /api/v2/user/email-push.
export function EmailPushToggle() {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const queryClient = useQueryClient();

	const query = useQuery<EmailPushPrefs>({
		queryKey: EMAIL_PUSH_QUERY_KEY,
		queryFn: async () => {
			const res = await getEmailPush();
			return res.data;
		},
		staleTime: 30_000,
	});

	const mutation = useMutation({
		mutationFn: (enabled: boolean) => patchEmailPush(enabled),
		onSuccess: (res) => {
			queryClient.setQueryData<EmailPushPrefs>(EMAIL_PUSH_QUERY_KEY, res.data);
			message.success(t("emailPush.updateSuccess"));
		},
		onError: () => {
			message.error(t("emailPush.updateError"));
		},
	});

	const handleChange = (checked: boolean) => {
		mutation.mutate(checked);
	};

	return (
		<Flex align="center" gap={12} data-testid="email-push-toggle">
			<MailOutlined style={{ fontSize: 16, color: "#595959" }} />
			<Flex vertical gap={2} style={{ flex: 1 }}>
				<Typography.Text>{t("emailPush.label")}</Typography.Text>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("emailPush.hint")}
				</Typography.Text>
			</Flex>
			<Switch
				checked={query.data?.emailPushEnabled ?? false}
				loading={query.isLoading || mutation.isPending}
				onChange={handleChange}
				data-testid="email-push-switch"
			/>
		</Flex>
	);
}
