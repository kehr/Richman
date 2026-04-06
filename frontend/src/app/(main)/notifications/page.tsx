"use client";

import {
	ChannelConfigForm,
	useChannels,
	useDeleteChannel,
	useUpdateChannel,
} from "@/features/notification";
import type { NotificationChannelDto } from "@/features/notification";
import {
	Button,
	Card,
	Modal,
	PageContainer,
	Popconfirm,
	ProTable,
	Space,
	Switch,
	Tag,
	message,
} from "@/ui-kit/eat";
import { DeleteOutlined, EditOutlined, PlusOutlined } from "@/ui-kit/eat";
import { useState } from "react";

export default function NotificationsPage() {
	const { data: channels, isLoading } = useChannels();
	const updateMutation = useUpdateChannel();
	const deleteMutation = useDeleteChannel();
	const [formOpen, setFormOpen] = useState(false);
	const [editingChannel, setEditingChannel] = useState<NotificationChannelDto | undefined>();

	const handleToggle = async (id: number, enabled: boolean) => {
		try {
			await updateMutation.mutateAsync({ id, data: { enabled } });
		} catch {
			message.error("Failed to update channel");
		}
	};

	const handleDelete = async (id: number) => {
		try {
			await deleteMutation.mutateAsync(id);
			message.success("Channel deleted");
		} catch {
			message.error("Failed to delete channel");
		}
	};

	const openCreate = () => {
		setEditingChannel(undefined);
		setFormOpen(true);
	};

	const openEdit = (channel: NotificationChannelDto) => {
		setEditingChannel(channel);
		setFormOpen(true);
	};

	const columns = [
		{
			title: "Type",
			dataIndex: "channelType",
			key: "channelType",
			width: 120,
			render: (_: unknown, record: NotificationChannelDto) => <Tag>{record.channelType}</Tag>,
		},
		{
			title: "Enabled",
			dataIndex: "enabled",
			key: "enabled",
			width: 100,
			render: (_: unknown, record: NotificationChannelDto) => (
				<Switch
					checked={record.enabled}
					onChange={(checked) => handleToggle(record.channelId, checked)}
				/>
			),
		},
		{
			title: "Config",
			dataIndex: "config",
			key: "config",
			render: (_: unknown, record: NotificationChannelDto) => (
				<code>{JSON.stringify(record.config, null, 0)}</code>
			),
		},
		{
			title: "Actions",
			key: "actions",
			width: 160,
			render: (_: unknown, record: NotificationChannelDto) => (
				<Space>
					<Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
						Edit
					</Button>
					<Popconfirm title="Delete this channel?" onConfirm={() => handleDelete(record.channelId)}>
						<Button type="link" size="small" danger icon={<DeleteOutlined />}>
							Delete
						</Button>
					</Popconfirm>
				</Space>
			),
		},
	];

	return (
		<PageContainer title="Notification Channels">
			<ProTable<NotificationChannelDto>
				columns={columns}
				dataSource={channels}
				rowKey="channelId"
				loading={isLoading}
				search={false}
				toolBarRender={() => [
					<Button key="add" type="primary" icon={<PlusOutlined />} onClick={openCreate}>
						Add Channel
					</Button>,
				]}
				pagination={false}
			/>

			<Modal
				title={editingChannel ? "Edit Channel" : "Add Channel"}
				open={formOpen}
				onCancel={() => setFormOpen(false)}
				footer={null}
				destroyOnClose
			>
				<ChannelConfigForm initialValues={editingChannel} onSuccess={() => setFormOpen(false)} />
			</Modal>
		</PageContainer>
	);
}
