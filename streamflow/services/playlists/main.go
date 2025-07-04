package main

// Playlists microservice gRPC implementation.
// Implements the PlaylistsService defined in media_services.proto.
// Rules (summarised):
//   • Todas las llamadas requieren usuario autenticado (metadata user_id, role).
//   • Sólo el creador de la lista puede añadir/eliminar vídeos y borrar la lista.
//   • ListPlaylists devuelve sólo las listas del usuario autenticado.
//   • ListVideos requiere ser el dueño de la lista.

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	// Update the import path below to match your actual Go module path for the generated protobuf files.
	// For example, if your module is "streamflow" and the generated files are in "protos/playlist.streamflow/services/playlists/pb", use:
	pb "playlists-service/pb" // => generated files (playlists_pb2.go etc.)
	// "playlists-service/pb" // => generated files (playlists_pb2.go etc.)
	//	"playlists/pb" // => generated files (playlists_pb2.go
)

//======================== CONFIG & DB =========================

type dbCfg struct{ Host, Port, Name, User, Pass string }

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func cfg() dbCfg {
	return dbCfg{
		Host: getenv("DB_HOST", "localhost"),
		Port: getenv("DB_PORT", "5432"),
		Name: getenv("DB_NAME", "playlists_db"),
		User: getenv("DB_USER", "postgres"),
		Pass: getenv("DB_PASSWORD", "password"),
	}
}
func (c dbCfg) dsn() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.Pass, c.Name)
}

var db *sql.DB

func initSchema() error {
	const q1 = `CREATE TABLE IF NOT EXISTS playlists (
        id SERIAL PRIMARY KEY,
        owner_id INTEGER NOT NULL,
        name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP NULL
    )`
	const q2 = `CREATE TABLE IF NOT EXISTS playlist_videos (
        id SERIAL PRIMARY KEY,
        playlist_id INTEGER NOT NULL,
        video_id INTEGER NOT NULL,
        added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        deleted_at TIMESTAMP NULL,
        UNIQUE (playlist_id, video_id)
    )`
	const q3 = `CREATE INDEX IF NOT EXISTS idx_playlists_owner ON playlists (owner_id)`
	const q4 = `CREATE INDEX IF NOT EXISTS idx_playlists_deleted ON playlists (deleted_at)`
	const q5 = `CREATE INDEX IF NOT EXISTS idx_playlist_videos_playlist ON playlist_videos (playlist_id)`
	const q6 = `CREATE INDEX IF NOT EXISTS idx_playlist_videos_deleted ON playlist_videos (deleted_at)`
	if _, err := db.Exec(q1); err != nil {
		return err
	}
	if _, err := db.Exec(q2); err != nil {
		return err
	}
	if _, err := db.Exec(q3); err != nil {
		return err
	}
	if _, err := db.Exec(q4); err != nil {
		return err
	}
	if _, err := db.Exec(q5); err != nil {
		return err
	}
	if _, err := db.Exec(q6); err != nil {
		return err
	}
	return nil
}

//======================== AUTH =========================

const mdUserID = "user_id"

type auth struct{ userID int64 }

func getAuth(ctx context.Context) (auth, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return auth{}, status.Error(codes.Unauthenticated, "metadata missing")
	}
	vals := md.Get(mdUserID)
	if len(vals) == 0 {
		return auth{}, status.Error(codes.Unauthenticated, "user not logged in")
	}
	uid, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {
		return auth{}, status.Error(codes.Unauthenticated, "invalid user_id")
	}
	return auth{userID: uid}, nil
}

//======================== SERVICE =========================

type srv struct {
	pb.UnimplementedPlaylistsServiceServer
}

// CreatePlaylist
func (s *srv) CreatePlaylist(ctx context.Context, req *pb.CreatePlaylistRequest) (*pb.CreatePlaylistResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name required")
	}
	var id int64
	err = db.QueryRowContext(ctx, `INSERT INTO playlists (owner_id, name) VALUES ($1, $2) RETURNING id`, auth.userID, req.Name).Scan(&id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert: %v", err)
	}
	p := &pb.Playlist{Id: id, OwnerId: auth.userID, Name: req.Name, CreatedAt: timestamppb.Now()}
	return &pb.CreatePlaylistResponse{Playlist: p}, nil
}

// AddVideo
func (s *srv) AddVideo(ctx context.Context, req *pb.AddVideoRequest) (*pb.AddVideoResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	var ownerID int64
	err = db.QueryRowContext(ctx, `SELECT owner_id FROM playlists WHERE id = $1 AND deleted_at IS NULL`, req.PlaylistId).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "playlist not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	if ownerID != auth.userID {
		return nil, status.Error(codes.PermissionDenied, "not owner")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO playlist_videos (playlist_id, video_id) VALUES ($1, $2)`, req.PlaylistId, req.VideoId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert: %v", err)
	}
	playlistResp, err := s.buildPlaylistResponse(ctx, req.PlaylistId)
	if err != nil {
		return nil, err
	}
	return &pb.AddVideoResponse{Playlist: playlistResp.Playlist}, nil
}

// RemoveVideo
func (s *srv) RemoveVideo(ctx context.Context, req *pb.RemoveVideoRequest) (*pb.RemoveVideoResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	var ownerID int64
	err = db.QueryRowContext(ctx, `SELECT owner_id FROM playlists WHERE id = $1 AND deleted_at IS NULL`, req.PlaylistId).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "playlist not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	if ownerID != auth.userID {
		return nil, status.Error(codes.PermissionDenied, "not owner")
	}
	_, err = db.ExecContext(ctx, `UPDATE playlist_videos SET deleted_at = NOW() WHERE playlist_id = $1 AND video_id = $2 AND deleted_at IS NULL`, req.PlaylistId, req.VideoId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete: %v", err)
	}
	playlistResp, err := s.buildPlaylistResponse(ctx, req.PlaylistId)
	if err != nil {
		return nil, err
	}
	return &pb.RemoveVideoResponse{Playlist: playlistResp.Playlist}, nil
}

// ListPlaylists
func (s *srv) ListPlaylists(ctx context.Context, _ *emptypb.Empty) (*pb.ListPlaylistsResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, name, created_at FROM playlists WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, auth.userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	defer rows.Close()
	var resp pb.ListPlaylistsResponse
	for rows.Next() {
		var id int64
		var name string
		var ts time.Time
		if err := rows.Scan(&id, &name, &ts); err != nil {
			return nil, status.Errorf(codes.Internal, "scan: %v", err)
		}
		resp.Playlists = append(resp.Playlists, &pb.Playlist{Id: id, OwnerId: auth.userID, Name: name, CreatedAt: timestamppb.New(ts)})
	}
	return &resp, nil
}

// ListVideos
func (s *srv) ListVideos(ctx context.Context, req *pb.ListVideosRequest) (*pb.ListVideosResponse, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	// verify owner
	var ownerID int64
	err = db.QueryRowContext(ctx, `SELECT owner_id FROM playlists WHERE id = $1 AND deleted_at IS NULL`, req.PlaylistId).Scan(&ownerID)
	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "playlist not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	if ownerID != auth.userID {
		return nil, status.Error(codes.PermissionDenied, "not owner")
	}

	rows, err := db.QueryContext(ctx, `SELECT video_id FROM playlist_videos WHERE playlist_id = $1 AND deleted_at IS NULL`, req.PlaylistId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query: %v", err)
	}
	defer rows.Close()
	var resp pb.ListVideosResponse
	for rows.Next() {
		var vid int64
		if err := rows.Scan(&vid); err != nil {
			return nil, status.Errorf(codes.Internal, "scan: %v", err)
		}
		resp.Videos = append(resp.Videos, &pb.VideoInPlaylist{VideoId: vid})
	}
	return &resp, nil
}

// DeletePlaylist
func (s *srv) DeletePlaylist(ctx context.Context, req *pb.DeletePlaylistRequest) (*emptypb.Empty, error) {
	auth, err := getAuth(ctx)
	if err != nil {
		return nil, err
	}
	// verify owner
	res, err := db.ExecContext(ctx, `UPDATE playlists SET deleted_at = NOW() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`, req.PlaylistId, auth.userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "delete: %v", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, status.Error(codes.PermissionDenied, "not owner or playlist not found")
	}
	return &emptypb.Empty{}, nil
}

// buildPlaylistResponse fetches minimal playlist row after modification
func (s *srv) buildPlaylistResponse(ctx context.Context, pid int64) (*pb.AddVideoResponse, error) {
	var name string
	var ts time.Time
	var owner int64
	err := db.QueryRowContext(ctx, `SELECT owner_id, name, created_at FROM playlists WHERE id = $1`, pid).Scan(&owner, &name, &ts)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "select: %v", err)
	}
	return &pb.AddVideoResponse{Playlist: &pb.Playlist{Id: pid, OwnerId: owner, Name: name, CreatedAt: timestamppb.New(ts)}}, nil
}

//======================== MAIN =========================

func main() {
	c := cfg()
	var err error
	db, err = sql.Open("postgres", c.dsn())
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	if err = initSchema(); err != nil {
		log.Fatalf("schema: %v", err)
	}
	log.Println("Playlists DB ready")

	lis, err := net.Listen("tcp", ":"+getenv("PORT", "50055"))
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPlaylistsServiceServer(grpcServer, &srv{})

	log.Printf("PlaylistsService listening on %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
