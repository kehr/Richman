import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface NotificationChannelDto {
	channelId: number;
	channelType: string;
	config: Record<string, unknown>;
	enabled: boolean;
}

export interface CreateChannelInput {
	channelType: string;
	config: Record<string, unknown>;
}

export function fetchChannels(): Promise<ApiResponse<NotificationChannelDto[]>> {
	return request<ApiResponse<NotificationChannelDto[]>>("/notification-channels");
}

export function createChannel(
	data: CreateChannelInput,
): Promise<ApiResponse<NotificationChannelDto>> {
	return request<ApiResponse<NotificationChannelDto>>("/notification-channels", {
		method: "POST",
		body: JSON.stringify(data),
	});
}

export function updateChannel(
	id: number,
	data: { config?: Record<string, unknown>; enabled?: boolean },
): Promise<ApiResponse<NotificationChannelDto>> {
	return request<ApiResponse<NotificationChannelDto>>(`/notification-channels/${id}`, {
		method: "PATCH",
		body: JSON.stringify(data),
	});
}

export function deleteChannel(id: number): Promise<ApiResponse<null>> {
	return request<ApiResponse<null>>(`/notification-channels/${id}`, {
		method: "DELETE",
	});
}
