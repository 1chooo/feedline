package model

import (
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	MinUsernameLength = 3
	MaxUsernameLength = 20
	MinPasswordLength = 8
	MaxDisplayName    = 80
	MaxBioLength      = 280
	MaxTitleLength    = 200
	MaxDescription    = 2000
	SessionTTL        = 30 * 24 * time.Hour
	RoleMember        = "member"
	RoleAdvertiser    = "advertiser"
	RoleAdmin         = "admin"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"displayName"`
	Bio          string    `json:"bio"`
	Age          *int      `json:"age,omitempty"`
	Gender       *string   `json:"gender,omitempty"`
	Country      *string   `json:"country,omitempty"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

type PublicUser struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	Bio         string  `json:"bio"`
	Age         *int    `json:"age,omitempty"`
	Gender      *string `json:"gender,omitempty"`
	Country     *string `json:"country,omitempty"`
	Role        string  `json:"role"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		Age:         u.Age,
		Gender:      u.Gender,
		Country:     u.Country,
		Role:        u.Role,
	}
}

func (u User) IsAdvertiser() bool {
	return u.Role == RoleAdvertiser || u.Role == RoleAdmin
}

func (u User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

type Session struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
}

type Post struct {
	ID             int64
	UserID         int64
	Title          string
	Description    string
	ImageUrl       string
	LandingPageUrl string
	CreatedAt      time.Time
	Author         PublicUser
}

type RegisterRequest struct {
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	DisplayName string  `json:"displayName"`
	Bio         string  `json:"bio,omitempty"`
	Age         *int    `json:"age,omitempty"`
	Gender      *string `json:"gender,omitempty"`
	Country     *string `json:"country,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type CreatePostRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description,omitempty"`
	ImageUrl       string `json:"imageUrl,omitempty"`
	LandingPageUrl string `json:"landingPageUrl,omitempty"`
}

type PostResponse struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	ImageUrl       string     `json:"imageUrl,omitempty"`
	LandingPageUrl string     `json:"landingPageUrl,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Author         PublicUser `json:"author"`
}

type ListPostsResponse struct {
	Items []PostResponse `json:"items"`
}

func NewPostResponse(post Post) PostResponse {
	resp := PostResponse{
		ID:        strconv.FormatInt(post.ID, 10),
		Username:  post.Author.Username,
		Title:     post.Title,
		CreatedAt: post.CreatedAt,
		Author:    post.Author,
	}
	if post.Description != "" {
		resp.Description = post.Description
	}
	if post.ImageUrl != "" {
		resp.ImageUrl = post.ImageUrl
	}
	if post.LandingPageUrl != "" {
		resp.LandingPageUrl = post.LandingPageUrl
	}
	return resp
}

func ValidateRegisterRequest(req RegisterRequest) (username, email, password, displayName, bio string, age *int, gender, country *string, err error) {
	username, err = normalizeUsername(req.Username)
	if err != nil {
		return "", "", "", "", "", nil, nil, nil, err
	}

	email = strings.ToLower(strings.TrimSpace(req.Email))
	if _, parseErr := mail.ParseAddress(email); parseErr != nil || !strings.Contains(email, ".") {
		return "", "", "", "", "", nil, nil, nil, invalid("email must be a valid email address")
	}

	password = req.Password
	if len(password) < MinPasswordLength {
		return "", "", "", "", "", nil, nil, nil, invalidf("password must be at least %d characters", MinPasswordLength)
	}

	displayName = strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return "", "", "", "", "", nil, nil, nil, invalid("displayName must be a non-empty string")
	}
	if len(displayName) > MaxDisplayName {
		return "", "", "", "", "", nil, nil, nil, invalidf("displayName must be at most %d characters", MaxDisplayName)
	}

	bio = strings.TrimSpace(req.Bio)
	if len(bio) > MaxBioLength {
		return "", "", "", "", "", nil, nil, nil, invalidf("bio must be at most %d characters", MaxBioLength)
	}

	age, gender, country, err = validateTargeting(req.Age, req.Gender, req.Country)
	if err != nil {
		return "", "", "", "", "", nil, nil, nil, err
	}

	return username, email, password, displayName, bio, age, gender, country, nil
}

func ValidateLoginRequest(req LoginRequest) (email, username, password string, err error) {
	email = strings.ToLower(strings.TrimSpace(req.Email))
	username = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(req.Username, "@")))
	password = req.Password
	if password == "" {
		return "", "", "", invalid("password is required")
	}
	if email == "" && username == "" {
		return "", "", "", invalid("email or username is required")
	}
	return email, username, password, nil
}

func ValidateCreatePostRequest(req CreatePostRequest) (title, description, imageUrl, landingPageUrl string, err error) {
	title = strings.TrimSpace(req.Title)
	if title == "" {
		return "", "", "", "", invalid("title must be a non-empty string")
	}
	if len(title) > MaxTitleLength {
		return "", "", "", "", invalidf("title must be at most %d characters", MaxTitleLength)
	}

	description = strings.TrimSpace(req.Description)
	if len(description) > MaxDescription {
		return "", "", "", "", invalidf("description must be at most %d characters", MaxDescription)
	}

	return title, description, strings.TrimSpace(req.ImageUrl), strings.TrimSpace(req.LandingPageUrl), nil
}

func NormalizeUsername(raw string) (string, error) {
	return normalizeUsername(raw)
}

func normalizeUsername(raw string) (string, error) {
	username := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(raw, "@")))
	if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
		return "", invalidf("username must be between %d and %d characters", MinUsernameLength, MaxUsernameLength)
	}
	for _, r := range username {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return "", invalid("username may only contain letters, numbers, and underscores")
		}
	}
	return username, nil
}

func validateTargeting(age *int, gender, country *string) (*int, *string, *string, error) {
	if age != nil {
		if *age < 1 || *age > 100 {
			return nil, nil, nil, invalid("age must be between 1 and 100")
		}
	}

	var normalizedGender *string
	if gender != nil && strings.TrimSpace(*gender) != "" {
		g := strings.ToUpper(strings.TrimSpace(*gender))
		if _, ok := validGenders[g]; !ok {
			return nil, nil, nil, invalid("gender must be M or F")
		}
		normalizedGender = &g
	}

	var normalizedCountry *string
	if country != nil && strings.TrimSpace(*country) != "" {
		c := strings.ToUpper(strings.TrimSpace(*country))
		if !isValidCountry(c) {
			return nil, nil, nil, invalid("country must be a valid ISO 3166-1 alpha-2 code")
		}
		normalizedCountry = &c
	}

	return age, normalizedGender, normalizedCountry, nil
}
