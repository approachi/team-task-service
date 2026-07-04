-- +goose Up
CREATE TABLE task_history (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id    BIGINT UNSIGNED NOT NULL,
    changed_by BIGINT UNSIGNED NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    old_value  TEXT NULL,
    new_value  TEXT NULL,
    changed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_task_history_task    FOREIGN KEY (task_id)    REFERENCES tasks(id) ON DELETE CASCADE,
    CONSTRAINT fk_task_history_changer FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- GET /tasks/{id}/history ordered by time
CREATE INDEX idx_task_history_task_changed_at ON task_history(task_id, changed_at);
CREATE INDEX idx_task_history_changed_by ON task_history(changed_by);

-- +goose Down
DROP TABLE task_history;
