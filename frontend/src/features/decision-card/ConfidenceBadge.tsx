"use client";

import { Tag } from "@/ui-kit/eat";

interface ConfidenceBadgeProps {
	value: number;
}

function getColor(value: number): string {
	if (value >= 0.7) return "green";
	if (value >= 0.4) return "orange";
	return "red";
}

export function ConfidenceBadge({ value }: ConfidenceBadgeProps) {
	const percent = Math.round(value * 100);
	return <Tag color={getColor(value)}>{percent}%</Tag>;
}
