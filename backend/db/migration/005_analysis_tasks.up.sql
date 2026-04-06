CREATE TABLE IF NOT EXISTS analysis_tasks (
	task_id UUID PRIMARY KEY,
	user_id BIGINT NOT NULL,
	status VARCHAR(32) NOT NULL,
	progress DOUBLE PRECISION NOT NULL DEFAULT 0,
	error TEXT,
	started_at TIMESTAMPTZ NOT NULL,
	done_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analysis_tasks_user ON analysis_tasks (user_id);
