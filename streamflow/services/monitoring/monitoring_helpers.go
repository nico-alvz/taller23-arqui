// monitoring_helpers.go
package main

import (
	"context"
	"os"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	mdUserID = "user_id"
	mdRole   = "role"
)

type authCtx struct {
	userID int64
	role   string // "admin" | "client"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getAuth(ctx context.Context) (authCtx, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return authCtx{}, status.Error(codes.Unauthenticated, "metadata missing: user not logged in")
	}
	uidVals := md.Get(mdUserID)
	roleVals := md.Get(mdRole)
	if len(uidVals) == 0 || len(roleVals) == 0 {
		return authCtx{}, status.Error(codes.Unauthenticated, "user not logged in")
	}
	uid, err := strconv.ParseInt(uidVals[0], 10, 64)
	if err != nil {
		return authCtx{}, status.Error(codes.Unauthenticated, "invalid user_id metadata")
	}
	role := roleVals[0]
	if role != "admin" && role != "client" {
		return authCtx{}, status.Error(codes.PermissionDenied, "unknown role")
	}
	return authCtx{userID: uid, role: role}, nil
}

func requireAdmin(a authCtx) error {
	if a.role != "admin" {
		return status.Error(codes.PermissionDenied, "admin privileges required")
	}
	return nil
}
