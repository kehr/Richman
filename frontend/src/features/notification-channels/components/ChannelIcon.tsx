import { Mail } from "lucide-react";
import type { ChannelType } from "../types";

interface ChannelIconProps {
	channelType: ChannelType;
	size?: number;
}

// FeishuIcon renders a simplified version of the Feishu/Lark brand mark.
// Brand color: #1456F0 (Feishu 2024 primary blue).
function FeishuIcon({ size }: { size: number }) {
	return (
		<svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
			{/* Two overlapping rounded shapes suggesting wings / the Lark bird motif */}
			<path
				d="M12 3C9.5 3 7.5 4.5 7.5 7c0 1.4.7 2.6 1.8 3.4L8 14l3.5-1.8c.16.02.32.04.5.04 2.5 0 4.5-1.5 4.5-4C16.5 5 14.5 3 12 3z"
				fill="#1456F0"
			/>
			<path
				d="M15.5 7.5c.33.48.5 1.04.5 1.64 0 2.48-2.24 4.36-5 4.36-.06 0-.13 0-.2-.01l-.3.16A5.5 5.5 0 0 0 13 14c.36 0 .71-.04 1.05-.12L17 15.5l-1-3.07A4.4 4.4 0 0 0 18 9c0-1.2-.6-2.23-1.5-2.93-.3-.23-.64-.42-1-.5z"
				fill="#1456F0"
				opacity="0.55"
			/>
		</svg>
	);
}

// WechatIcon renders WeChat's classic overlapping double speech-bubble mark.
// Brand color: #07C160 (WeChat official green).
function WechatIcon({ size }: { size: number }) {
	return (
		<svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
			{/* Primary bubble (lower-left, larger) */}
			<path
				d="M9.5 3C5.91 3 3 5.46 3 8.5c0 1.66.84 3.15 2.17 4.17L4.5 15l2.87-1.43c.69.19 1.41.43 2.13.43 3.59 0 6.5-2.46 6.5-5.5S13.09 3 9.5 3z"
				fill="#07C160"
			/>
			{/* Secondary bubble (upper-right, smaller) */}
			<path
				d="M14.5 9c3.04 0 5.5 2.02 5.5 4.5 0 1.3-.64 2.46-1.66 3.27L19 19l-2.55-1.28A5.9 5.9 0 0 1 14.5 18c-3.04 0-5.5-2.02-5.5-4.5 0-.18.01-.36.03-.53.47.06.96.03 1.47.03 2.45 0 4.65-.88 6.08-2.28A4.5 4.5 0 0 0 14.5 9z"
				fill="#07C160"
				opacity="0.6"
			/>
		</svg>
	);
}

// ChannelIcon resolves the brand icon for a given notification channel type.
export function ChannelIcon({ channelType, size = 20 }: ChannelIconProps) {
	switch (channelType) {
		case "email":
			return <Mail size={size} color="#1677ff" strokeWidth={1.5} />;
		case "feishu":
			return <FeishuIcon size={size} />;
		case "wechat":
			return <WechatIcon size={size} />;
		default:
			return <Mail size={size} strokeWidth={1.5} />;
	}
}
