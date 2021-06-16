package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type images struct{ col *mongo.Collection }

// Images returns the mongodb implementation of media.ImageRepository.
func Images(ctx context.Context, col *mongo.Collection) (image.Repository, error) {
	r := &images{col: col}
	if err := r.createIndexes(ctx); err != nil {
		return r, fmt.Errorf("create indexes: %w", err)
	}
	return r, nil
}

func (r *images) Save(ctx context.Context, img media.Image) error {
	if _, err := r.col.ReplaceOne(ctx, bson.D{
		{Key: "disk", Value: img.Disk},
		{Key: "path", Value: img.Path},
	}, img, options.Replace().SetUpsert(true)); err != nil {
		return fmt.Errorf("mongo: %w", err)
	}
	return nil
}

func (r *images) Get(ctx context.Context, disk, path string) (media.Image, error) {
	res := r.col.FindOne(ctx, bson.D{
		{Key: "disk", Value: disk},
		{Key: "path", Value: path},
	})

	var img media.Image
	if err := res.Decode(&img); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return media.Image{}, media.ErrUnknownImage
		}
		return img, fmt.Errorf("decode image: %w", err)
	}

	return img, nil
}

func (r *images) Delete(ctx context.Context, disk, path string) error {
	if _, err := r.col.DeleteOne(ctx, bson.D{
		{Key: "disk", Value: disk},
		{Key: "path", Value: path},
	}); err != nil {
		return fmt.Errorf("mongo: %w", err)
	}
	return nil
}

func (r *images) createIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "disk", Value: 1},
				{Key: "path", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	})
	return err
}
