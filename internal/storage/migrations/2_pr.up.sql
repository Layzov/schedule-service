CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id TEXT PRIMARY KEY UNIQUE NOT NULL,
    pull_request_name TEXT NOT NULL,
    author_id TEXT NOT NULL,
    pr_status TEXT NOT NULL,
    merged_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(user_id) ON DELETE RESTRICT
)

CREATE INDEX IF NOT EXISTS idx_author_id ON pull_requests (author_id);
CREATE INDEX IF NOT EXISTS idx_status ON pull_requests (pr_status);

CREATE TRIGGER update_pr_updated_at
    BEFORE UPDATE ON pull_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION set_merged_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.pr_status = 'MERGED' AND OLD.pr_status <> 'MERGED' THEN
        NEW.merged_at := NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_set_merged_at
BEFORE UPDATE OF pr_status ON pull_requests
FOR EACH ROW
EXECUTE FUNCTION set_merged_at();


CREATE TABLE IF NOT EXISTS pr_reviewers (
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    reviewer_id TEXT DEFAULT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pull_request_id, reviewer_id)
)

CREATE INDEX IF NOT EXISTS idx_pr_reviewers ON pr_reviewers (reviewer_id);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_pr ON pr_reviewers (pull_request_id);