package models

type PRStatus string

const (
	PR_OPEN PRStatus = "OPEN"
	PR_MERGED PRStatus = "MERGED"
)

type PullRequestShort struct {
	PullRequestID  	string `db:"pull_request_id"`
	PullRequestName string `db:"pull_request_name"`
	AuthorID       	string `db:"author_id"`
	Status 			PRStatus `db:"status"`
}

type User struct {
	UserID string `db:"user_id"`
	Username string `db:"username"`
	TeamName string `db:"team_name"`
	IsActive bool `db:"is_active"`
}

type Team struct {
	TeamName string `db:"team_name"`
	Members []User 
}
