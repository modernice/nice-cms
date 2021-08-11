package mediarpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	protomedia "github.com/modernice/nice-cms/internal/proto/gen/media/v1"
	"github.com/modernice/nice-cms/internal/proto/ptypes/v1"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
	"google.golang.org/grpc"
)

// Server is the media gRPC server.
type Server struct {
	protomedia.UnimplementedMediaServiceServer

	shelfs document.Repository
	lookup *document.Lookup

	galleries gallery.Repository

	storage media.Storage
}

// NewServer returns the media gRPC server.
func NewServer(
	shelfs document.Repository,
	lookup *document.Lookup,
	galleries gallery.Repository,
	storage media.Storage,
) *Server {
	return &Server{
		shelfs:    shelfs,
		lookup:    lookup,
		galleries: galleries,
		storage:   storage,
	}
}

// LookupShelfByName looks up the UUID of a shelf by its name.
func (s *Server) LookupShelfByName(ctx context.Context, req *protomedia.LookupShelfByNameReq) (*protomedia.LookupShelfResp, error) {
	id, ok := s.lookup.ShelfName(req.GetName())
	if !ok {
		return &protomedia.LookupShelfResp{Found: false}, nil
	}
	return &protomedia.LookupShelfResp{
		Found:      true,
		DocumentId: ptypes.UUIDProto(id),
	}, nil
}

// UploadDocument uploads a document to a shelf.
func (s *Server) UploadDocument(stream protomedia.MediaService_UploadDocumentServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := req.GetMetadata()
	if meta == nil {
		return errors.New("missing metadata")
	}

	receiveError := make(chan error)
	failReceive := func(err error) {
		select {
		case <-stream.Context().Done():
		case receiveError <- err:
		}
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				failReceive(err)
				return
			}

			chunk := req.GetChunk()
			if chunk == nil {
				failReceive(errors.New("missing chunk"))
				return
			}

			if _, err = pw.Write(chunk); err != nil {
				failReceive(err)
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case err := <-receiveError:
			if err != nil {
				cancel()
			}
		}
	}()

	var doc document.Document
	if err := s.shelfs.Use(ctx, ptypes.UUID(meta.GetShelfId()), func(shelf *document.Shelf) error {
		doc, err = shelf.Add(ctx, s.storage, pr, meta.GetUniqueName(), meta.GetName(), meta.GetDisk(), meta.GetPath())
		return err
	}); err != nil {
		return err
	}

	return stream.SendAndClose(ptypes.ShelfDocumentProto(doc))
}

// ReplaceDocument replaces a document within a shelf.
func (s *Server) ReplaceDocument(stream protomedia.MediaService_ReplaceDocumentServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := req.GetMetadata()
	if meta == nil {
		return errors.New("missing metadata")
	}

	receiveError := make(chan error)
	failReceive := func(err error) {
		select {
		case <-stream.Context().Done():
		case receiveError <- err:
		}
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				failReceive(err)
				return
			}

			chunk := req.GetChunk()
			if chunk == nil {
				failReceive(errors.New("missing chunk"))
				return
			}

			if _, err := pw.Write(chunk); err != nil {
				failReceive(err)
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case err := <-receiveError:
			if err != nil {
				cancel()
			}
		}
	}()

	var doc document.Document
	if err := s.shelfs.Use(ctx, ptypes.UUID(meta.GetShelfId()), func(shelf *document.Shelf) error {
		doc, err = shelf.Replace(ctx, s.storage, pr, ptypes.UUID(meta.GetDocumentId()))
		return err
	}); err != nil {
		return err
	}

	return stream.SendAndClose(ptypes.ShelfDocumentProto(doc))
}

func (s *Server) UploadImage(stream protomedia.MediaService_UploadImageServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := req.GetMetadata()
	if meta == nil {
		return errors.New("missing metadata")
	}

	receiveError := make(chan error)
	failReceive := func(err error) {
		select {
		case <-stream.Context().Done():
		case receiveError <- err:
		}
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				failReceive(err)
				return
			}

			chunk := req.GetChunk()
			if chunk == nil {
				failReceive(errors.New("missing chunk"))
				return
			}

			if _, err := pw.Write(chunk); err != nil {
				failReceive(err)
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case err := <-receiveError:
			if err != nil {
				cancel()
			}
		}
	}()

	var stack gallery.Stack
	if err := s.galleries.Use(ctx, ptypes.UUID(meta.GetGalleryId()), func(g *gallery.Gallery) error {
		stack, err = g.Upload(ctx, s.storage, pr, meta.GetName(), meta.GetDisk(), meta.GetPath())
		return err
	}); err != nil {
		return err
	}

	return stream.SendAndClose(ptypes.GalleryStackProto(stack))
}

func (s *Server) ReplaceImage(stream protomedia.MediaService_ReplaceImageServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	meta := req.GetMetadata()
	if meta == nil {
		return errors.New("missing metadata")
	}

	receiveError := make(chan error)
	failReceive := func(err error) {
		select {
		case <-stream.Context().Done():
		case receiveError <- err:
		}
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				failReceive(err)
				return
			}

			chunk := req.GetChunk()
			if chunk == nil {
				failReceive(errors.New("missing chunk"))
				return
			}

			if _, err := pw.Write(chunk); err != nil {
				failReceive(err)
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
		case err := <-receiveError:
			if err != nil {
				cancel()
			}
		}
	}()

	var stack gallery.Stack
	if err := s.galleries.Use(ctx, ptypes.UUID(meta.GetGalleryId()), func(g *gallery.Gallery) error {
		stack, err = g.Replace(ctx, s.storage, pr, ptypes.UUID(meta.GetStackId()))
		return err
	}); err != nil {
		return err
	}

	return stream.SendAndClose(ptypes.GalleryStackProto(stack))
}

// Client is the media gRPC client.
type Client struct{ client protomedia.MediaServiceClient }

// NewClient returns the media gRPC client.
func NewClient(conn grpc.ClientConnInterface) *Client {
	return &Client{client: protomedia.NewMediaServiceClient(conn)}
}

// LookupShelfByName looks up the UUID of a shelf by its name.
func (c *Client) LookupShelfByName(ctx context.Context, name string) (uuid.UUID, bool, error) {
	resp, err := c.client.LookupShelfByName(ctx, &protomedia.LookupShelfByNameReq{Name: name})
	if err != nil {
		return uuid.Nil, false, err
	}
	return ptypes.UUID(resp.GetDocumentId()), resp.GetFound(), nil
}

// UploadDocument uploads a document to a shelf.
func (c *Client) UploadDocument(
	ctx context.Context,
	shelfID uuid.UUID,
	r io.Reader,
	uniqueName, name, disk, path string,
) (document.Document, error) {
	stream, err := c.client.UploadDocument(ctx)
	if err != nil {
		return document.Document{}, err
	}

	if err := stream.Send(&protomedia.UploadDocumentReq{
		UploadData: &protomedia.UploadDocumentReq_Metadata{
			Metadata: &protomedia.UploadDocumentReq_UploadDocumentMetadata{
				ShelfId:    ptypes.UUIDProto(shelfID),
				UniqueName: uniqueName,
				Name:       name,
				Disk:       disk,
				Path:       path,
			},
		},
	}); err != nil {
		return document.Document{}, fmt.Errorf("send metadata: %w", err)
	}

	buf := make([]byte, 512)
L:
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break L
		}

		if err != nil {
			return document.Document{}, err
		}

		if err := stream.Send(&protomedia.UploadDocumentReq{
			UploadData: &protomedia.UploadDocumentReq_Chunk{Chunk: buf[:n]},
		}); err != nil {
			return document.Document{}, fmt.Errorf("send chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return document.Document{}, err
	}

	return ptypes.ShelfDocument(resp), nil
}

// ReplaceDocument replaces a document within a shelf.
func (c *Client) ReplaceDocument(ctx context.Context, shelfID, documentID uuid.UUID, r io.Reader) (document.Document, error) {
	stream, err := c.client.ReplaceDocument(ctx)
	if err != nil {
		return document.Document{}, err
	}

	if err := stream.Send(&protomedia.ReplaceDocumentReq{
		ReplaceData: &protomedia.ReplaceDocumentReq_Metadata{
			Metadata: &protomedia.ReplaceDocumentReq_ReplaceDocumentMetadata{
				ShelfId:    ptypes.UUIDProto(shelfID),
				DocumentId: ptypes.UUIDProto(documentID),
			},
		},
	}); err != nil {
		return document.Document{}, fmt.Errorf("send metadata: %w", err)
	}

	buf := make([]byte, 512)
L:
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break L
		}

		if err != nil {
			return document.Document{}, err
		}

		if err := stream.Send(&protomedia.ReplaceDocumentReq{
			ReplaceData: &protomedia.ReplaceDocumentReq_Chunk{Chunk: buf[:n]},
		}); err != nil {
			return document.Document{}, fmt.Errorf("send chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return document.Document{}, err
	}

	return ptypes.ShelfDocument(resp), nil
}

func (c *Client) UploadImage(ctx context.Context, galleryID uuid.UUID, r io.Reader, name, disk, path string) (gallery.Stack, error) {
	stream, err := c.client.UploadImage(ctx)
	if err != nil {
		return gallery.Stack{}, err
	}

	if err := stream.Send(&protomedia.UploadImageReq{
		UploadData: &protomedia.UploadImageReq_Metadata{
			Metadata: &protomedia.UploadImageReq_UploadImageMetadata{
				GalleryId: ptypes.UUIDProto(galleryID),
				Name:      name,
				Disk:      disk,
				Path:      path,
			},
		},
	}); err != nil {
		return gallery.Stack{}, fmt.Errorf("send metadata: %w", err)
	}

	buf := make([]byte, 512)
L:
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break L
		}
		if err != nil {
			return gallery.Stack{}, err
		}

		if err := stream.Send(&protomedia.UploadImageReq{
			UploadData: &protomedia.UploadImageReq_Chunk{Chunk: buf[:n]},
		}); err != nil {
			return gallery.Stack{}, fmt.Errorf("send chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return gallery.Stack{}, err
	}

	return ptypes.GalleryStack(resp), nil
}

func (c *Client) ReplaceImage(ctx context.Context, galleryID, stackID uuid.UUID, r io.Reader) (gallery.Stack, error) {
	stream, err := c.client.ReplaceImage(ctx)
	if err != nil {
		return gallery.Stack{}, err
	}

	if err := stream.Send(&protomedia.ReplaceImageReq{
		ReplaceData: &protomedia.ReplaceImageReq_Metadata{
			Metadata: &protomedia.ReplaceImageReq_ReplaceImageMetadata{
				GalleryId: ptypes.UUIDProto(galleryID),
				StackId:   ptypes.UUIDProto(stackID),
			},
		},
	}); err != nil {
		return gallery.Stack{}, fmt.Errorf("send metadata: %w", err)
	}

	buf := make([]byte, 512)
L:
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break L
		}
		if err != nil {
			return gallery.Stack{}, err
		}

		if err := stream.Send(&protomedia.ReplaceImageReq{
			ReplaceData: &protomedia.ReplaceImageReq_Chunk{
				Chunk: buf[:n],
			},
		}); err != nil {
			return gallery.Stack{}, fmt.Errorf("send chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return gallery.Stack{}, err
	}

	return ptypes.GalleryStack(resp), nil
}
