package productservice

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
)

func TestProductImagesReplaceAtomicallyAndPreserveReviewStatistics(t *testing.T) {
	db := testutil.Postgres(t)
	store := gormrepo.NewStore(db)
	service := NewService(store.Products)
	ctx := context.Background()
	input := dto.UpsertProductInput{Name: "Phone", Slug: "phone", Category: "Phones", Brand: "Brand", Description: "Phone", Stock: 10, Images: []string{"cover", "detail", "cover"}, Price: "10.00"}
	product, err := service.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := service.GetBySlug(ctx, product.Slug)
	if err != nil || strings.Join(loaded.ImageURLs(), ",") != "cover,detail,cover" {
		t.Fatalf("images=%+v err=%v", loaded, err)
	}
	if err := db.Model(&model.Product{}).Where("id = ?", product.ID).Updates(map[string]interface{}{"rating": 4, "numReviews": 2}).Error; err != nil {
		t.Fatal(err)
	}
	input.Images = []string{"new-detail", "new-cover"}
	updated, err := service.Update(ctx, product.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if updated.NumReviews != 2 || !updated.Rating.Equal(decimal.NewFromInt(4)) {
		t.Fatal("admin edit overwrote review statistics")
	}
	latest, err := service.ListLatest(ctx, 1)
	if err != nil || len(latest) != 1 || strings.Join(latest[0].ImageURLs(), ",") != "new-detail,new-cover" {
		t.Fatalf("updated images=%+v err=%v", latest, err)
	}
	// A failed child insert must roll back the parent edit and old-image deletion.
	candidate := *updated
	candidate.Name = "Must roll back"
	candidate.Images = []model.ProductImage{{URL: "invalid"}}
	if err := db.Exec(`ALTER TABLE "ProductImage" ADD CONSTRAINT reject_invalid CHECK (url <> 'invalid')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := store.Products.Update(ctx, &candidate); err == nil {
		t.Fatal("expected child insert failure")
	}
	loaded, err = service.GetByID(ctx, product.ID)
	if err != nil || loaded.Name != "Phone" || strings.Join(loaded.ImageURLs(), ",") != "new-detail,new-cover" {
		t.Fatalf("update was not atomic: %+v %v", loaded, err)
	}
	if err := service.Delete(ctx, product.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.ProductImage{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("orphan images=%d %v", count, err)
	}
}
