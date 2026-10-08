package repository

import (
	"context"
	"fmt"

	"github.com/1chooo/ad-service/internal/database"
	"github.com/1chooo/ad-service/internal/model"
)

type PostRepository struct {
	db database.DB
}

func NewPostRepository(db database.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) CreatePost(ctx context.Context, post *model.Post) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO posts (user_id, title, description, image_url, landing_page_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, post.UserID, post.Title, post.Description, post.ImageUrl, post.LandingPageUrl).Scan(&post.ID, &post.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}
	return nil
}

func (r *PostRepository) ListPosts(ctx context.Context) ([]model.Post, error) {
	return r.queryPosts(ctx, `
		SELECT p.id, p.user_id, p.title, p.description, p.image_url, p.landing_page_url, p.created_at,
		       u.username, u.display_name, u.bio, u.age, u.gender, u.country
		FROM posts p
		JOIN users u ON u.id = p.user_id
		ORDER BY p.created_at DESC
	`)
}

func (r *PostRepository) ListPostsByUsername(ctx context.Context, username string) ([]model.Post, error) {
	return r.queryPosts(ctx, `
		SELECT p.id, p.user_id, p.title, p.description, p.image_url, p.landing_page_url, p.created_at,
		       u.username, u.display_name, u.bio, u.age, u.gender, u.country
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE u.username = $1
		ORDER BY p.created_at DESC
	`, username)
}

func (r *PostRepository) queryPosts(ctx context.Context, query string, args ...any) ([]model.Post, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	if posts == nil {
		posts = []model.Post{}
	}
	return posts, nil
}

func scanPost(row rowScanner) (model.Post, error) {
	var post model.Post
	if err := row.Scan(
		&post.ID,
		&post.UserID,
		&post.Title,
		&post.Description,
		&post.ImageUrl,
		&post.LandingPageUrl,
		&post.CreatedAt,
		&post.Author.Username,
		&post.Author.DisplayName,
		&post.Author.Bio,
		&post.Author.Age,
		&post.Author.Gender,
		&post.Author.Country,
	); err != nil {
		return model.Post{}, fmt.Errorf("scan post: %w", err)
	}
	return post, nil
}
