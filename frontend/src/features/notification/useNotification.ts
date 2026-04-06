import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createChannel, deleteChannel, fetchChannels, updateChannel } from "./api";
import type { CreateChannelInput } from "./api";

export function useChannels() {
	return useQuery({
		queryKey: ["notification-channels"],
		queryFn: fetchChannels,
		select: (res) => res.data,
	});
}

export function useCreateChannel() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: CreateChannelInput) => createChannel(data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["notification-channels"] });
		},
	});
}

export function useUpdateChannel() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({
			id,
			data,
		}: {
			id: number;
			data: { config?: Record<string, unknown>; enabled?: boolean };
		}) => updateChannel(id, data),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["notification-channels"] });
		},
	});
}

export function useDeleteChannel() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: number) => deleteChannel(id),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["notification-channels"] });
		},
	});
}
