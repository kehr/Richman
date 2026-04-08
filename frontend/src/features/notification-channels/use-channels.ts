import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createChannel, deleteChannel, listChannels, updateChannel } from "./api";
import type { ChannelDto, CreateChannelInput, UpdateChannelInput } from "./types";

export const NOTIFICATION_CHANNELS_QUERY_KEY = ["notification-channels"] as const;

// useChannels loads the user's full notification channel list. The list is
// small (3-5 rows) so we keep the data fresh and refetch on focus.
export function useChannels() {
	return useQuery<ChannelDto[]>({
		queryKey: NOTIFICATION_CHANNELS_QUERY_KEY,
		queryFn: async () => {
			const res = await listChannels();
			return res.data;
		},
		staleTime: 10_000,
	});
}

function useInvalidateChannels() {
	const queryClient = useQueryClient();
	return () => queryClient.invalidateQueries({ queryKey: NOTIFICATION_CHANNELS_QUERY_KEY });
}

export function useCreateChannel() {
	const invalidate = useInvalidateChannels();
	return useMutation({
		mutationFn: (input: CreateChannelInput) => createChannel(input),
		onSuccess: invalidate,
	});
}

export function useUpdateChannel() {
	const invalidate = useInvalidateChannels();
	return useMutation({
		mutationFn: ({ channelId, input }: { channelId: number; input: UpdateChannelInput }) =>
			updateChannel(channelId, input),
		onSuccess: invalidate,
	});
}

export function useDeleteChannel() {
	const invalidate = useInvalidateChannels();
	return useMutation({
		mutationFn: (channelId: number) => deleteChannel(channelId),
		onSuccess: invalidate,
	});
}
