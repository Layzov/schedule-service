package api

import "time"

type PullRequest struct {
	PullRequestID 	string `json:"pull_request_id"`
	PullRequestName	string `json:"pull_request_name"`
	AuthorID 		string `json:"author_id"`
	Status 			string `json:"status"`
	Reviewers		[]string `json:"assigned_reviewers"`
	MergedAt 		*time.Time `json:"mergedAt,omitempty"`
}

type PRCreateRequest struct {
	PullRequestID  	string `json:"pull_request_id"`
	PullRequestName	string `json:"pull_request_name"`
	AuthorID 		string `json:"author_id"`
}

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName    string       `json:"team_name"`
	Members     []TeamMember `json:"members"`
}