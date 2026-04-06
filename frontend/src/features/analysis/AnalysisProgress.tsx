import { Alert, Button, Card, Progress, Space, Typography } from "@/ui-kit/eat";
import { ReloadOutlined } from "@/ui-kit/eat";
import { useState } from "react";
import { useTaskStatus, useTriggerAnalysis } from "./useAnalysis";

const { Text, Title } = Typography;

export function AnalysisProgress() {
	const [taskId, setTaskId] = useState<string | null>(null);
	const trigger = useTriggerAnalysis();
	const { data: task } = useTaskStatus(taskId);

	const handleTrigger = async () => {
		try {
			const result = await trigger.mutateAsync();
			setTaskId(result.data.taskId);
		} catch {
			// error handled by mutation
		}
	};

	const isRunning = task && task.status !== "completed" && task.status !== "failed";
	const isCompleted = task?.status === "completed";
	const isFailed = task?.status === "failed";
	const statusCopy = (() => {
		if (!task) return "";
		if (isFailed) return task.error || "Analysis failed";
		if (isCompleted) return "Analysis completed";
		return "Analysis running...";
	})();

	return (
		<Card>
			<Space direction="vertical" style={{ width: "100%" }} size="large">
				<div
					style={{
						display: "flex",
						justifyContent: "space-between",
						alignItems: "center",
					}}
				>
					<Title level={4} style={{ margin: 0 }}>
						Analysis
					</Title>
					<Button
						type="primary"
						icon={<ReloadOutlined />}
						onClick={handleTrigger}
						loading={trigger.isPending}
						disabled={!!isRunning}
					>
						{isRunning ? "Running..." : "Start Analysis"}
					</Button>
				</div>

				{task && (
					<>
					<Progress
						percent={Math.round((task.progress ?? 0) * 100)}
						status={isFailed ? "exception" : isCompleted ? "success" : "active"}
					/>
					<Text>{statusCopy}</Text>
					{isFailed && (
						<Alert
							type="error"
							message="Analysis failed"
							description={task.error || "Unknown error"}
						/>
					)}
						{isCompleted && (
							<Alert
								type="success"
								message="Analysis completed"
								description="Decision cards have been updated."
							/>
						)}
					</>
				)}
			</Space>
		</Card>
	);
}
