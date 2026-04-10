import { type Language, usePatchUserSettings, useUserSettings } from "@/features/user-settings";
import { Collapse, Divider, Flex, Radio, Typography, message } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// PreferencesTab covers PRD §6.4: language radio, timezone select (frozen at
// Asia/Shanghai for MVP), theme placeholder, and a collapsible "advanced
// number formatting" panel. The language setting persists to the backend so
// the LLM synthesis layer generates content in the user's preferred language.
export function PreferencesTab() {
	const { t, i18n } = useTranslation("settings");
	const settingsQuery = useUserSettings();
	const patchMutation = usePatchUserSettings();

	const handleLanguageChange = async (lang: Language) => {
		await i18n.changeLanguage(lang);
		try {
			await patchMutation.mutateAsync({ language: lang });
		} catch {
			message.error(t("preferences.languageSaveError"));
		}
	};

	return (
		<Flex vertical gap={24} data-testid="preferences-tab">
			<Flex vertical gap={8}>
				<Typography.Text type="secondary">{t("preferences.language")}</Typography.Text>
				<Radio.Group
					value={settingsQuery.data?.language ?? i18n.language}
					onChange={(e) => handleLanguageChange(e.target.value as Language)}
					disabled={settingsQuery.isLoading}
					data-testid="preferences-language"
				>
					<Radio value="zh">中文</Radio>
					<Radio value="en">English</Radio>
				</Radio.Group>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("preferences.languageHint")}
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">{t("preferences.timezone")}</Typography.Text>
				<Radio.Group
					defaultValue="Asia/Shanghai"
					options={[
						{ label: "Asia/Shanghai (UTC+8)", value: "Asia/Shanghai" },
						{ label: "Asia/Hong_Kong (UTC+8)", value: "Asia/Hong_Kong" },
						{ label: "America/New_York (UTC-5)", value: "America/New_York" },
					]}
					data-testid="preferences-timezone"
				/>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("preferences.timezoneHint")}
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">{t("preferences.theme")}</Typography.Text>
				<Radio.Group value="light" disabled data-testid="preferences-theme">
					<Radio value="light">{t("preferences.themeLight")}</Radio>
				</Radio.Group>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					{t("preferences.themeHint")}
				</Typography.Text>
			</Flex>

			<Collapse
				ghost
				items={[
					{
						key: "number-format",
						label: t("preferences.numberFormat"),
						children: (
							<Flex vertical gap={8}>
								<Typography.Text type="secondary">
									{t("preferences.numberFormatHint")}
								</Typography.Text>
							</Flex>
						),
					},
				]}
			/>
		</Flex>
	);
}
