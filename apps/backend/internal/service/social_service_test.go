package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type mockUserStore struct {
	users    []*model.User
	sessions []*model.Session
}

func (m *mockUserStore) CreateUser(_ context.Context, user *model.User) error {
	for _, existing := range m.users {
		if existing.Username == user.Username {
			return model.Conflict("username is already taken")
		}
		if existing.Email == user.Email {
			return model.Conflict("email is already in use")
		}
	}
	user.ID = int64(len(m.users) + 1)
	user.CreatedAt = time.Now().UTC()
	clone := *user
	m.users = append(m.users, &clone)
	return nil
}

func (m *mockUserStore) GetByEmail(_ context.Context, email string) (*model.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			clone := *user
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) GetByUsername(_ context.Context, username string) (*model.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			clone := *user
			return &clone, nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) CreateSession(_ context.Context, session *model.Session) error {
	session.ID = int64(len(m.sessions) + 1)
	clone := *session
	m.sessions = append(m.sessions, &clone)
	return nil
}

func (m *mockUserStore) GetUserByTokenHash(_ context.Context, tokenHash string, now time.Time) (*model.User, error) {
	for _, session := range m.sessions {
		if session.TokenHash == tokenHash && session.ExpiresAt.After(now) {
			for _, user := range m.users {
				if user.ID == session.UserID {
					clone := *user
					return &clone, nil
				}
			}
		}
	}
	return nil, nil
}

func (m *mockUserStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	kept := m.sessions[:0]
	for _, session := range m.sessions {
		if session.TokenHash != tokenHash {
			kept = append(kept, session)
		}
	}
	m.sessions = kept
	return nil
}

func (m *mockUserStore) UpdateRole(_ context.Context, userID int64, role string) error {
	for _, user := range m.users {
		if user.ID == userID {
			user.Role = role
			return nil
		}
	}
	return nil
}

func (m *mockUserStore) RecordActivity(_ context.Context, _ int64, _ string, _ time.Time) error {
	return nil
}

type mockPostStore struct {
	posts []model.Post
}

func (m *mockPostStore) CreatePost(_ context.Context, post *model.Post) error {
	post.ID = int64(len(m.posts) + 1)
	post.CreatedAt = time.Now().UTC()
	m.posts = append(m.posts, *post)
	return nil
}

func (m *mockPostStore) ListPosts(_ context.Context) ([]model.Post, error) {
	return append([]model.Post(nil), m.posts...), nil
}

func (m *mockPostStore) ListPostsByUsername(_ context.Context, username string) ([]model.Post, error) {
	var out []model.Post
	for _, post := range m.posts {
		if post.Author.Username == username {
			out = append(out, post)
		}
	}
	return out, nil
}

func TestRegisterAndLogin(t *testing.T) {
	t.Parallel()

	svc := NewSocialService(&mockUserStore{}, &mockPostStore{})
	resp, err := svc.Register(context.Background(), model.RegisterRequest{
		Username:    "Jane",
		Email:       "jane@stream.local",
		Password:    "password123",
		DisplayName: "Jane Park",
		Bio:         "Northline Outdoor.",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp.User.Username != "jane" {
		t.Fatalf("username = %q, want jane", resp.User.Username)
	}
	if resp.Token == "" {
		t.Fatal("expected session token")
	}
	if resp.User.Role != model.RoleMember {
		t.Fatalf("new user role = %q, want member", resp.User.Role)
	}

	me, err := svc.Me(context.Background(), resp.Token)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	if me.DisplayName != "Jane Park" {
		t.Fatalf("display name = %q", me.DisplayName)
	}

	login, err := svc.Login(context.Background(), model.LoginRequest{
		Email:    "jane@stream.local",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.User.Username != "jane" {
		t.Fatalf("login username = %q", login.User.Username)
	}

	_, err = svc.Login(context.Background(), model.LoginRequest{
		Email:    "jane@stream.local",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected invalid password to fail")
	}
}

func TestActivateAdvertiserRequiresAuthenticatedUser(t *testing.T) {
	t.Parallel()

	svc := NewSocialService(&mockUserStore{}, &mockPostStore{})
	auth, err := svc.Register(context.Background(), model.RegisterRequest{
		Username:    "brand",
		Email:       "brand@stream.local",
		Password:    "password123",
		DisplayName: "Brand Team",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	user, err := svc.ActivateAdvertiser(context.Background(), auth.Token)
	if err != nil {
		t.Fatalf("activate advertiser: %v", err)
	}
	if user.Role != model.RoleAdvertiser {
		t.Fatalf("role = %q, want advertiser", user.Role)
	}
	if _, err := svc.RequireAdvertiser(context.Background(), auth.Token); err != nil {
		t.Fatalf("require advertiser: %v", err)
	}
	if _, err := svc.RequireAdvertiser(context.Background(), ""); err == nil {
		t.Fatal("expected unauthenticated advertiser check to fail")
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()

	svc := NewSocialService(&mockUserStore{}, &mockPostStore{})
	_, err := svc.Register(context.Background(), model.RegisterRequest{
		Username:    "ab",
		Email:       "not-an-email",
		Password:    "short",
		DisplayName: "",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCreateAndListPosts(t *testing.T) {
	t.Parallel()

	users := &mockUserStore{}
	posts := &mockPostStore{}
	svc := NewSocialService(users, posts)

	auth, err := svc.Register(context.Background(), model.RegisterRequest{
		Username:    "kai",
		Email:       "kai@stream.local",
		Password:    "password123",
		DisplayName: "Kai Rivera",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	created, err := svc.CreatePost(context.Background(), auth.Token, model.CreatePostRequest{
		Title:       "Harbor blend",
		Description: "Roasted this morning",
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if created.Title != "Harbor blend" || created.Author.Username != "kai" {
		t.Fatalf("unexpected post: %+v", created)
	}

	list, err := svc.ListPosts(context.Background())
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d posts, want 1", len(list.Items))
	}

	profile, err := svc.ListPostsByUsername(context.Background(), "@Kai")
	if err != nil {
		t.Fatalf("list by username: %v", err)
	}
	if len(profile.Items) != 1 {
		t.Fatalf("profile posts = %d", len(profile.Items))
	}

	_, err = svc.CreatePost(context.Background(), "", model.CreatePostRequest{Title: "Nope"})
	if err == nil {
		t.Fatal("expected create without session to fail")
	}
}

func TestGetUserNotFound(t *testing.T) {
	t.Parallel()

	svc := NewSocialService(&mockUserStore{}, &mockPostStore{})
	_, err := svc.GetUser(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected missing user to fail")
	}
}

func TestSeedPasswordHashes(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte("password123")) != nil {
		t.Fatal("hash should match seed password")
	}
	if !strings.HasPrefix(string(hash), "$2") {
		t.Fatalf("unexpected hash prefix %q", hash)
	}
}
