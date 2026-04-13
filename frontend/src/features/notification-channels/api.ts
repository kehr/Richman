import { requestV1 as request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { ChannelDto, CreateChannelInput, UpdateChannelInput } from "./types";

// listChannels returns every notification channel owned by the current user.
export function listChannels() {
	return request<ApiResponse<ChannelDto[]>>("/notification/channels");
}

// createChannel registers a new channel. The backend validates the config
// shape against the adapter for the given channelType.
export function createChannel(input: CreateChannelInput) {
	return request<ApiResponse<ChannelDto>>("/notification/channels", {
		method: "POST",
		body: JSON.stringify(input),
	});
}

// updateChannel sends a sparse PUT (config and/or enabled). Fields omitted
// from the payload are left unchanged on the server.
export function updateChannel(channelId: number, input: UpdateChannelInput) {
	return request<ApiResponse<ChannelDto>>(`/notification/channels/${channelId}`, {
		method: "PUT",
		body: JSON.stringify(input),
	});
}

// deleteChannel permanently removes a channel from the user's account.
export function deleteChannel(channelId: number) {
	return request<ApiResponse<{ message: string }>>(`/notification/channels/${channelId}`, {
		method: "DELETE",
	});
}
