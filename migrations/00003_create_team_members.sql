-- +goose Up
CREATE TABLE team_members (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    team_id    BIGINT UNSIGNED NOT NULL,
    user_id    BIGINT UNSIGNED NOT NULL,
    role       ENUM('owner','admin','member') NOT NULL DEFAULT 'member',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY uq_team_members_team_user (team_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- membership lookup "which teams does user X belong to"
CREATE INDEX idx_team_members_user_id ON team_members(user_id);

-- +goose Down
DROP TABLE team_members;
