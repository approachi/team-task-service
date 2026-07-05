-- +goose Up
CREATE TABLE tasks (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id     BIGINT UNSIGNED NOT NULL,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    status      ENUM('todo','in_progress','done') NOT NULL DEFAULT 'todo',
    assignee_to BIGINT UNSIGNED NULL,
    created_by  BIGINT UNSIGNED NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_tasks_team     FOREIGN KEY (team_id)     REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_to) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_tasks_creator  FOREIGN KEY (created_by)  REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- base list query: team_id [+ created_at ordering] when no other filters given
CREATE INDEX idx_tasks_team_created_at ON tasks(team_id, created_at);
-- team_id + status filter combo
CREATE INDEX idx_tasks_team_status ON tasks(team_id, status);
-- team_id + assignee_to filter combo
CREATE INDEX idx_tasks_team_assignee ON tasks(team_id, assignee_to);
-- future: per-team "done in last 7 days" aggregate (WHERE team_id=? AND status='done' AND updated_at >= ?)
CREATE INDEX idx_tasks_team_status_updated ON tasks(team_id, status, updated_at);
-- future: per-team-per-month top-3 creators window function (GROUP BY team_id, created_by, MONTH(created_at))
CREATE INDEX idx_tasks_creator_team_created ON tasks(created_by, team_id, created_at);

-- +goose Down
DROP TABLE tasks;
