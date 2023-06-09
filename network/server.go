package network

import (
	"github.com/qcodelabsllc/qreeket/media/config"
	pb "github.com/qcodelabsllc/qreeket/media/generated"
	svc "github.com/qcodelabsllc/qreeket/media/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"strconv"
)

func InitServer() {
	// create a new grpc server
	s := grpc.NewServer(
		grpc.UnaryInterceptor(AuthUnaryInterceptor),
		grpc.StreamInterceptor(AuthStreamInterceptor),
	)

	// register the grpc server for reflection
	reflection.Register(s)

	// create cloudinary instance
	cld := config.InitCloudinary()

	// register the server
	pb.RegisterMediaServiceServer(s, svc.NewQreeketMediaServerInstance(cld))

	// get the port number from .env file
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	// listen on the port
	if lis, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(port)); err == nil {
		log.Printf("server started on %v\n", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("unable to start grpc server: %+v\n", err)
		}
	} else {
		panic(err)
	}
}
