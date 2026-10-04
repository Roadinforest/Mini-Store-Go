package reviewservice

import (
	"context"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
	"testing"
)

func TestReviewUpdateRecomputesRatingWithoutDuplicatingReview(t *testing.T) {
	db := testutil.Postgres(t)
	user1 := model.User{ID: uuid.NewString(), Email: "one@test.invalid"}
	user2 := model.User{ID: uuid.NewString(), Email: "two@test.invalid"}
	product := model.Product{ID: "phone", Name: "Phone", Slug: "phone", Images: pq.StringArray{}}
	for _, row := range []any{&user1, &user2, &product} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := gormrepo.NewStore(db)
	service := NewService(db, store.Reviews, store.Products)
	ctx := context.Background()
	input := dto.UpsertReviewInput{ProductID: product.ID, Rating: 5, Title: "Review", Description: "Test"}
	first, err := service.Upsert(ctx, user1.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Rating = 3
	if _, err := service.Upsert(ctx, user2.ID, input); err != nil {
		t.Fatal(err)
	}
	input.Rating = 1
	updated, err := service.Upsert(ctx, user1.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != updated.ID {
		t.Fatal("update created new review")
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.NumReviews != 2 || !product.Rating.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("rating=%s count=%d", product.Rating, product.NumReviews)
	}
}
