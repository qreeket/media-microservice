package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/admin"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	pb "github.com/qcodelabsllc/qreeket/media/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"io"
	"log"
	"net/http"
	"time"
)

const maxImageSize = 10 << 20 // 10MB

type QreeketMediaServer struct {
	pb.UnimplementedMediaServiceServer
	cld *cloudinary.Cloudinary
}

func NewQreeketMediaServerInstance(cld *cloudinary.Cloudinary) *QreeketMediaServer {
	return &QreeketMediaServer{
		UnimplementedMediaServiceServer: pb.UnimplementedMediaServiceServer{},
		cld:                             cld,
	}
}

// UploadMedia uploads a media to cloudinary
// done
func (pms *QreeketMediaServer) UploadMedia(ctx context.Context, req *pb.UploadMediaRequest) (*pb.UploadMediaResponse, error) {

	// convert to base64 string
	encodedString := base64.StdEncoding.EncodeToString(req.GetMedia())

	// format name
	var uploadName string
	if req.GetName() == "" {
		uploadName = fmt.Sprintf("qreeket-media-%d", time.Now().UnixNano())
	} else {
		uploadName = fmt.Sprintf("%s-%s", req.GetName(), req.GetOwner())
	}

	// get mime type from the media
	mimeType := http.DetectContentType(req.GetMedia())

	// prepend the mime type to the encoded string
	encodedString = fmt.Sprintf("data:%s;base64,%s", mimeType, encodedString)

	// upload to cloudinary
	var result *uploader.UploadResult
	if req.GetType() == pb.MediaType_IMAGE {
		if uploadResult, err := pms.cld.Upload.Upload(ctx, encodedString, uploader.UploadParams{PublicID: uploadName}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to upload media: %v", err)
		} else {
			result = uploadResult
		}
	} else {
		if uploadResult, err := pms.cld.Upload.Upload(ctx, encodedString, uploader.UploadParams{PublicID: uploadName,
			Eager:      "w_300,h_300,c_pad,ac_none|w_160,h_100,c_crop,g_south,ac_none",
			EagerAsync: api.Bool(true),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to upload media: %v", err)
		} else {
			result = uploadResult
		}
	}

	// Get the file size
	if size, err := pms.cld.Admin.Asset(ctx, admin.AssetParams{PublicID: uploadName}); err == nil {
		return &pb.UploadMediaResponse{
			Url:  result.SecureURL,
			Size: uint32(size.Bytes),
		}, nil
	}

	// else return only the url
	return &pb.UploadMediaResponse{
		Url: result.SecureURL,
	}, nil
}

// GetMedia gets a media from cloudinary
func (pms *QreeketMediaServer) GetMedia(_ context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// TODO -> implement this
	response := &wrapperspb.StringValue{Value: req.GetValue()}
	return response, nil
}

// DeleteMedia deletes a media from cloudinary
func (pms *QreeketMediaServer) DeleteMedia(ctx context.Context, uploadName *wrapperspb.StringValue) (*emptypb.Empty, error) {
	// destroy the previous media
	invalidate := true
	if _, err := pms.cld.Upload.Destroy(ctx, uploader.DestroyParams{Invalidate: &invalidate, PublicID: uploadName.GetValue()}); err != nil {
		log.Printf("failed to destroy previous media: %v", err)
		return nil, status.Errorf(codes.Internal, "Unable to delete media file")
	}
	return &emptypb.Empty{}, nil
}

// UploadLargeMedia uploads a large media to cloudinary
func (pms *QreeketMediaServer) UploadLargeMedia(req pb.MediaService_UploadLargeMediaServer) error {
	imageData := bytes.Buffer{}
	imageSize := 0
	var mediaType pb.MediaType
	var name string

	for {
		data, err := req.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive chunk data: %v", err)
		}

		// format name
		if data.GetName() == "" {
			name = fmt.Sprintf("qreeket-media-%d", time.Now().UnixNano())
		} else {
			name = fmt.Sprintf("%s-%s", data.GetName(), data.GetOwner())
		}

		media := data.GetMedia()
		mediaType = data.GetType()
		size := len(media)

		if size > maxImageSize {
			return status.Errorf(codes.InvalidArgument, "image size exceeds maximum size of %v  > %v", imageSize, maxImageSize)
		}
		imageSize += size

		if _, err = imageData.Write(media); err != nil {
			return status.Errorf(codes.Internal, "failed to write chunk data: %v", err)
		}
	}

	// perform upload
	request := &pb.UploadMediaRequest{
		Media: imageData.Bytes(),
		Name:  &name,
		Type:  mediaType,
	}
	uploadResponse, err := pms.UploadMedia(req.Context(), request)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to upload media: %v", err)
	}

	if err := req.SendAndClose(uploadResponse); err != nil {
		return status.Errorf(codes.Internal, "failed to send response: %v", err)
	}

	return nil
}
