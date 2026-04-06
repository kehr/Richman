"use client";

import { Button, Empty, Input, Modal, Table, Tabs } from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import type { AssetDto } from "./api";
import { useAssets } from "./useAssetCatalog";

const ASSET_TYPES = [
	{ key: "all", label: "All" },
	{ key: "a_share", label: "A-Share" },
	{ key: "us_stock", label: "US Stock" },
	{ key: "gold", label: "Gold" },
	{ key: "event", label: "Event" },
];

interface AssetPickerProps {
	open: boolean;
	onClose: () => void;
	onSelect: (asset: AssetDto) => void;
}

export function AssetPicker({ open, onClose, onSelect }: AssetPickerProps) {
	const [keyword, setKeyword] = useState("");
	const [activeType, setActiveType] = useState("all");

	const queryType = activeType === "all" ? undefined : activeType;
	const { data: assets, isLoading } = useAssets({
		type: queryType,
		keyword: keyword || undefined,
	});

	const columns = useMemo(
		() => [
			{ title: "Code", dataIndex: "code", key: "code", width: 120 },
			{ title: "Name", dataIndex: "name", key: "name" },
			{ title: "Type", dataIndex: "assetType", key: "assetType", width: 100 },
			{ title: "Exchange", dataIndex: "exchange", key: "exchange", width: 100 },
			{
				title: "Action",
				key: "action",
				width: 80,
				render: (_: unknown, record: AssetDto) => (
					<Button
						type="link"
						size="small"
						onClick={() => {
							onSelect(record);
							onClose();
						}}
					>
						Select
					</Button>
				),
			},
		],
		[onSelect, onClose],
	);

	return (
		<Modal title="Select Asset" open={open} onCancel={onClose} footer={null} width={640}>
			<Input.Search
				placeholder="Search by code or name"
				value={keyword}
				onChange={(e) => setKeyword(e.target.value)}
				style={{ marginBottom: 12 }}
				allowClear
			/>
			<Tabs
				activeKey={activeType}
				onChange={setActiveType}
				items={ASSET_TYPES.map((t) => ({ key: t.key, label: t.label }))}
			/>
			{assets?.length === 0 && !isLoading ? (
				<Empty description="No assets found" />
			) : (
				<Table
					dataSource={assets}
					columns={columns}
					rowKey="code"
					loading={isLoading}
					size="small"
					pagination={{ pageSize: 10 }}
					scroll={{ y: 300 }}
				/>
			)}
		</Modal>
	);
}
