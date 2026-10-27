package ctxkey

import "context"

type contextKey string

const UserID contextKey = "userID"

func UserIDFrom(ctx context.Context) string {
	userID, _ := ctx.Value(UserID).(string)
	return userID
}
