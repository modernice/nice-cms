package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/modernice/cms/media/image/gallery"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type galleryStacks struct{ col *mongo.Collection }

type galleryStack struct {
	ID      uuid.UUID       `bson:"id"`
	Gallery string          `bson:"gallery"`
	Images  []gallery.Image `bson:"images"`
}

// GalleryStacks returns the MongoDB implementation of gallery.StackRepository.
func GalleryStacks(ctx context.Context, col *mongo.Collection) (gallery.StackRepository, error) {
	r := &galleryStacks{col: col}
	if err := r.createIndexes(ctx); err != nil {
		return r, fmt.Errorf("create indexes: %w", err)
	}
	return r, nil
}

func (r *galleryStacks) Save(ctx context.Context, stack gallery.Stack) error {
	_, err := r.col.ReplaceOne(ctx, bson.D{
		{Key: "id", Value: stack.ID},
	}, galleryStack{
		ID:      stack.ID,
		Gallery: stack.Gallery,
		Images:  stack.Images,
	}, options.Replace().SetUpsert(true))
	return err
}

func (r *galleryStacks) Get(ctx context.Context, id uuid.UUID) (gallery.Stack, error) {
	res := r.col.FindOne(ctx, bson.D{{Key: "id", Value: id}})

	var stack galleryStack
	if err := res.Decode(&stack); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return gallery.Stack{}, gallery.ErrStackNotFound
		}
		return gallery.Stack{}, fmt.Errorf("decode: %w", err)
	}

	return gallery.Stack{
		ID:      stack.ID,
		Gallery: stack.Gallery,
		Images:  stack.Images,
	}, nil
}

func (r *galleryStacks) Delete(ctx context.Context, stack gallery.Stack) error {
	_, err := r.col.DeleteOne(ctx, bson.D{{Key: "id", Value: stack.ID}})
	return err
}

func (r *galleryStacks) createIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "gallery", Value: 1}}},
	})
	return err
}
