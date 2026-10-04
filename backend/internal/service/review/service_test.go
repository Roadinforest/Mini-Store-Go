package reviewservice

import (
	"context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/dto"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
	"testing"
	"time"
)

func TestReviewUpdateRecomputesRatingWithoutDuplicatingReview(t *testing.T) {
	db := testutil.Postgres(t)
	user1 := model.User{ID: uuid.NewString(), Email: "one@test.invalid"}
	user2 := model.User{ID: uuid.NewString(), Email: "two@test.invalid"}
	product := model.Product{ID: "phone", Name: "Phone", Slug: "phone", Images: []model.ProductImage{}}
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

func TestConcurrentReviewUpsertsRemainUniqueAndVerifyActualPurchase(t *testing.T) {
	db := testutil.Postgres(t)
	user := model.User{ID: uuid.NewString(), Email: "concurrent@test.invalid"}
	product := model.Product{ID: "concurrent", Name: "Phone", Slug: "concurrent"}
	for _, row := range []any{&user, &product} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := gormrepo.NewStore(db)
	service := NewService(db, store.Reviews, store.Products)
	ctx := context.Background()
	input := dto.UpsertReviewInput{ProductID: product.ID, Rating: 4, Title: "Review", Description: "Test"}
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		go func() { _, err := service.Upsert(ctx, user.ID, input); errs <- err }()
	}
	for i := 0; i < 12; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	review, err := service.GetByUserAndProduct(ctx, user.ID, product.ID)
	if err != nil || review.IsVerifiedPurchase {
		t.Fatalf("unverified purchase: %+v %v", review, err)
	}
	var count int64
	if err := db.Model(&model.Review{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("reviews=%d err=%v", count, err)
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.NumReviews != 1 || !product.Rating.Equal(decimal.NewFromInt(4)) {
		t.Fatalf("stats=%+v", product)
	}
	now := time.Now().UTC()
	order := model.Order{ID: uuid.NewString(), UserID: user.ID, ShippingAddress: valueobject.NewJSON(valueobject.ShippingAddress{}), PaymentMethod: "cash", PaidAt: &now, OrderItems: []model.OrderItem{{ProductID: product.ID, Qty: 1, Price: decimal.NewFromInt(10)}}}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	review, err = service.Upsert(ctx, user.ID, input)
	if err != nil || !review.IsVerifiedPurchase {
		t.Fatalf("paid purchase: %+v %v", review, err)
	}
}
