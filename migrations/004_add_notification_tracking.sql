CREATE TABLE IF NOT EXISTS task_notifications (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL, -- 'due_soon', 'overdue', 'task_assigned'
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'sent', -- 'sent', 'failed', 'pending'
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes   
CREATE INDEX idx_task_notifications_task ON task_notifications(task_id);
CREATE INDEX idx_task_notifications_user ON task_notifications(user_id);
CREATE INDEX idx_task_notifications_type_sent ON task_notifications(notification_type, sent_at);
CREATE INDEX idx_task_notifications_status ON task_notifications(status) WHERE status != 'sent';
CREATE INDEX idx_task_notifications_pending ON task_notifications(status, retry_count) WHERE status = 'failed';

CREATE UNIQUE INDEX idx_unique_daily_notification 
ON task_notifications(task_id, user_id, notification_type, DATE(sent_at));
