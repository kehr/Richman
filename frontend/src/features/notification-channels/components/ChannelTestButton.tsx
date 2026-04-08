import { Button, Tooltip } from "@/ui-kit/eat";

// ChannelTestButton is a disabled placeholder for the "test send" action.
// The backend has no test-send endpoint yet (see Step 17 trade-delete pattern),
// so we render an always-disabled button with a tooltip explaining the
// situation rather than wiring up a frontend-only mock path.
export function ChannelTestButton() {
	return (
		<Tooltip title="测试发送接口待后端补齐">
			<Button size="small" disabled data-testid="channel-test-button">
				测试发送
			</Button>
		</Tooltip>
	);
}
