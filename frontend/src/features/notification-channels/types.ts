// Notification channel DTOs mirror backend/internal/model/notification_channel.go.
// The backend stores config as a free-form json.RawMessage, but on the
// frontend we narrow channelType to the three adapter subtypes that exist
// today (see backend/internal/notification/adapter/{email,feishu,wechat}).

export type ChannelType = "email" | "feishu" | "wechat";

// EmailChannelConfig matches backend/internal/notification/adapter/email/email.go
// emailConfig: { to string }. The recipient overrides the user's email; when
// empty the backend falls back to msg.UserEmail.
export interface EmailChannelConfig {
	to: string;
}

// FeishuChannelConfig matches backend feishuConfig: { webhookUrl string }.
export interface FeishuChannelConfig {
	webhookUrl: string;
}

// WechatChannelConfig matches backend wechatConfig: { openId, templateId }.
// Note: this is the WeChat official-account template message shape, not a
// webhook. The PRD copy says "微信公众号" so this matches.
export interface WechatChannelConfig {
	openId: string;
	templateId: string;
}

export type ChannelConfig = EmailChannelConfig | FeishuChannelConfig | WechatChannelConfig;

export interface ChannelDto {
	channelId: number;
	userId: number;
	channelType: ChannelType;
	// Config is whatever JSON the backend round-trips. We treat it as
	// unknown at the boundary and narrow inside the per-type renderers.
	config: unknown;
	enabled: boolean;
	createdAt: string;
	updatedAt: string;
}

export interface CreateChannelInput {
	channelType: ChannelType;
	config: ChannelConfig;
}

export interface UpdateChannelInput {
	config?: ChannelConfig;
	enabled?: boolean;
}
