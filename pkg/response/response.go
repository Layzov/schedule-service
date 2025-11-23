package response

import "errors"

// import (
// 	"fmt"
// 	"strings"
// 	"github.com/go-playground/validator/v10"
// )

type Response struct {
	ResponseError `json:"error,omitzero"`
}

type ResponseError struct {
	Code string `json:"code"`
	Message  string `json:"message"`
}

//Error Codes
type ErrCode string 
var (
	FAILED_REQUEST ErrCode = "REQUEST_FAILED"
	BAD_REQUEST ErrCode = "FAILED_TO_DECODE"
	TEAM_EXISTS ErrCode = "TEAM_EXISTS"
	NOT_FOUND ErrCode = "NOT_FOUND"
	PR_EXISTS ErrCode = "PR_EXISTS"
	NO_CANDIDATE ErrCode = "NO_CANDIDATE"
	NOT_ASSIGNED ErrCode = "NOT_ASSIGNED"
	PR_MERGED ErrCode = "PR_MERGED"
)

var (
	ErrBadRequest = errors.New("bad request")
	ErrInvalidId = errors.New("invalid user_id")
	ErrTeamExists = errors.New("team_name already exists")
	ErrNotFound = errors.New("resource not found")
	ErrPRExists = errors.New("PR id already exists")
	ErrNoCandidate = errors.New("no active replacement candidate in team")
	ErrNotAssigned = errors.New("reviewer is not assigned to this PR")
	ErrPRMerged = errors.New("cannot reassign on merged PR")
)

func Error(code, msg string) Response {
	return Response{
		ResponseError: ResponseError{
			Code:    code,
			Message: msg,
		},
	}
}	

// func ValidationError(errs validator.ValidationErrors) Response {
// 	var errMsg []string

// 	for _, err := range errs {
// 		switch err.ActualTag() {
// 		case "required":
// 			errMsg = append(errMsg, fmt.Sprintf("Field '%s' is required", err.Field()))
// 		case "min":
// 			errMsg = append(errMsg, fmt.Sprintf("Field '%s' must be at least %s characters long", err.Field(), err.Param()))
// 		case "max":
// 			errMsg = append(errMsg, fmt.Sprintf("Field '%s' must be at most %s characters long", err.Field(), err.Param()))
// 		default:
// 			errMsg = append(errMsg, fmt.Sprintf("Field '%s' is invalid", err.Field()))
// 		}
// 	}

// 	return Response{
// 		Status: StatusError,
// 		Error:  strings.Join(errMsg, ", "),
// 	}
// }
