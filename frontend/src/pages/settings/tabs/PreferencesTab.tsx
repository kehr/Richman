import { useLocale } from "@/domain/i18n/provider";
import { Collapse, Divider, Flex, Radio, Select, Typography } from "@/ui-kit/eat";

// PreferencesTab covers PRD §6.4: language radio, timezone select (frozen at
// Asia/Shanghai for MVP), theme placeholder, and a collapsible "advanced
// number formatting" panel. None of these settings round-trip to the backend
// today; language is the only one that takes effect immediately, via the
// existing i18n provider.
export function PreferencesTab() {
	const { locale, setLocale } = useLocale();

	return (
		<Flex vertical gap={24} data-testid="preferences-tab">
			<Flex vertical gap={8}>
				<Typography.Text type="secondary">语言</Typography.Text>
				<Radio.Group
					value={locale}
					onChange={(e) => setLocale(e.target.value)}
					data-testid="preferences-language"
				>
					<Radio value="zh">中文</Radio>
					<Radio value="en">English</Radio>
				</Radio.Group>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">时区</Typography.Text>
				<Select
					defaultValue="Asia/Shanghai"
					style={{ width: 240 }}
					options={[
						{ label: "Asia/Shanghai (UTC+8)", value: "Asia/Shanghai" },
						{ label: "Asia/Hong_Kong (UTC+8)", value: "Asia/Hong_Kong" },
						{ label: "America/New_York (UTC-5)", value: "America/New_York" },
					]}
					data-testid="preferences-timezone"
				/>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					MVP 默认 Asia/Shanghai，时区切换暂不影响推送窗口。
				</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Flex vertical gap={8}>
				<Typography.Text type="secondary">主题</Typography.Text>
				<Radio.Group value="light" disabled data-testid="preferences-theme">
					<Radio value="light">亮色</Radio>
				</Radio.Group>
				<Typography.Text type="secondary" style={{ fontSize: 12 }}>
					MVP 暂不支持暗色。
				</Typography.Text>
			</Flex>

			<Collapse
				ghost
				items={[
					{
						key: "number-format",
						label: "数字格式（高级选项）",
						children: (
							<Flex vertical gap={8}>
								<Typography.Text type="secondary">
									千分位分隔符 / 货币符号位置 / 小数位数等高级选项预留位，MVP 暂未启用。
								</Typography.Text>
							</Flex>
						),
					},
				]}
			/>
		</Flex>
	);
}
