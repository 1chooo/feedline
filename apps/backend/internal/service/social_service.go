package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/1chooo/ad-service/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type UserStore interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	CreateSession(ctx context.Context, session *model.Session) error
	GetUserByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*model.User, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
}

type PostStore interface {
	CreatePost(ctx context.Context, post *model.Post) error
	ListPosts(ctx context.Context) ([]model.Post, error)
	ListPostsByUsername(ctx context.Context, username string) ([]model.Post, error)
}

type SocialService struct {
	users UserStore
	posts PostStore
	now   func() time.Time
}

func NewSocialService(users UserStore, posts PostStore) *SocialService {
	return &SocialService{
		users: users,
		posts: posts,
		now:   time.Now,
	}
}

func (s *SocialService) WithClock(now func() time.Time) *SocialService {
	s.now = now
	return s
}

func (s *SocialService) Register(ctx context.Context, req model.RegisterRequest) (*model.AuthResponse, error) {
	username, email, password, displayName, bio, age, gender, country, err := model.ValidateRegisterRequest(req)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Bio:          bio,
		Age:          age,
		Gender:       gender,
		Country:      country,
	}
	if err := s.users.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, User: user.Public()}, nil
}

func (s *SocialService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, error) {
	email, username, password, err := model.ValidateLoginRequest(req)
	if err != nil {
		return nil, err
	}

	var user *model.User
	if email != "" {
		user, err = s.users.GetByEmail(ctx, email)
	} else {
		user, err = s.users.GetByUsername(ctx, username)
	}
	if err != nil {
		return nil, err
	}
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, model.Unauthorized("invalid email or password")
	}

	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, User: user.Public()}, nil
}

func (s *SocialService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.users.DeleteSessionByTokenHash(ctx, hashToken(token))
}

func (s *SocialService) Me(ctx context.Context, token string) (*model.User, error) {
	user, err := s.UserByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, model.Unauthorized("sign in required")
	}
	return user, nil
}

func (s *SocialService) UserByToken(ctx context.Context, token string) (*model.User, error) {
	if token == "" {
		return nil, nil
	}
	return s.users.GetUserByTokenHash(ctx, hashToken(token), s.now().UTC())
}

func (s *SocialService) GetUser(ctx context.Context, username string) (*model.PublicUser, error) {
	normalized, err := normalizeLookupUsername(username)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByUsername(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, model.NotFound("user not found")
	}
	public := user.Public()
	return &public, nil
}

func (s *SocialService) ListPosts(ctx context.Context) (*model.ListPostsResponse, error) {
	posts, err := s.posts.ListPosts(ctx)
	if err != nil {
		return nil, err
	}
	return postsResponse(posts), nil
}

func (s *SocialService) ListPostsByUsername(ctx context.Context, username string) (*model.ListPostsResponse, error) {
	normalized, err := normalizeLookupUsername(username)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByUsername(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, model.NotFound("user not found")
	}
	posts, err := s.posts.ListPostsByUsername(ctx, user.Username)
	if err != nil {
		return nil, err
	}
	return postsResponse(posts), nil
}

func (s *SocialService) CreatePost(ctx context.Context, token string, req model.CreatePostRequest) (*model.PostResponse, error) {
	user, err := s.Me(ctx, token)
	if err != nil {
		return nil, err
	}

	title, description, imageUrl, landingPageUrl, err := model.ValidateCreatePostRequest(req)
	if err != nil {
		return nil, err
	}

	post := &model.Post{
		UserID:         user.ID,
		Title:          title,
		Description:    description,
		ImageUrl:       imageUrl,
		LandingPageUrl: landingPageUrl,
		Author:         user.Public(),
	}
	if err := s.posts.CreatePost(ctx, post); err != nil {
		return nil, err
	}

	resp := model.NewPostResponse(*post)
	return &resp, nil
}

func (s *SocialService) createSession(ctx context.Context, userID int64) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	session := &model.Session{
		UserID:    userID,
		TokenHash: hashToken(token),
		ExpiresAt: s.now().UTC().Add(model.SessionTTL),
	}
	if err := s.users.CreateSession(ctx, session); err != nil {
		return "", err
	}
	return token, nil
}

func postsResponse(posts []model.Post) *model.ListPostsResponse {
	items := make([]model.PostResponse, len(posts))
	for i, post := range posts {
		items[i] = model.NewPostResponse(post)
	}
	return &model.ListPostsResponse{Items: items}
}

func normalizeLookupUsername(username string) (string, error) {
	normalized, err := model.NormalizeUsername(username)
	if err != nil {
		return "", model.NotFound("user not found")
	}
	return normalized, nil
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
