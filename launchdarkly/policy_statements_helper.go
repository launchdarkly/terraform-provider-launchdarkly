package launchdarkly

import (
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func statementsToStatementReps(policies []ldapi.Statement) []ldapi.Statement {
	statements := make([]ldapi.Statement, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.Statement(p)
		statements = append(statements, rep)
	}
	return statements
}

// The relay proxy config api requires a statementRep in the POST body
func statementPostsToStatementReps(policies []ldapi.StatementPost) []ldapi.Statement {
	statements := make([]ldapi.Statement, 0, len(policies))
	for _, p := range policies {
		rep := ldapi.Statement(p)
		statements = append(statements, rep)
	}
	return statements
}
