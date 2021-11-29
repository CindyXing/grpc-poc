package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
        // "google.golang.org/grpc/health"
	// "google.golang.org/grpc/health/grpc_health_v1"
	"grpc-health-check/proto"
	//"grpc-health-check/server/healthcheck"
	"net"
        "net/http"
)

type server struct{}

func init(){
	logrus.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:       true,
	})
}

func (s *server) Hello(helloReq *proto.HelloRequest, srv proto.GreetService_HelloServer) error {
	logrus.Infof("Server received an rpc request with the following parameter %v", helloReq.Hello)

	for i := 0; i<=10 ; i++ {
		resp := &proto.HelloResponse{
			Greet: fmt.Sprintf("Hello %s for %d time",helloReq.Hello, i),
		}
		srv.SendMsg(resp)
	}
	return nil
}

type healthHandler struct {
}

func (m *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logrus.Println("Listening on 5001: health ")
        w.Write([]byte("Listening on 5001: foo "))
}

func main() {

        go func() {
                        logrus.Println("I am here")
                        http.Handle("/healthz", &healthHandler{}) 
			http.ListenAndServe(":5001", nil)
	}()

	serverAdr := ":5000"
	listenAddr, err := net.Listen("tcp", serverAdr)
	if err != nil {
		logrus.Fatalf("Error while starting the listening service %v", err.Error())
	}

	grpcServer := grpc.NewServer()
	proto.RegisterGreetServiceServer(grpcServer, &server{})


	// healthService := healthcheck.NewHealthChecker()
        // grpc_health_v1.RegisterHealthServer(grpcServer, healthService)

        // grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	
	logrus.Infof("Server starting to listen on %s", serverAdr)
	if err = grpcServer.Serve(listenAddr); err!= nil {
		logrus.Fatalf("Error while starting the gRPC server on the %s listen address %v", listenAddr, err.Error())
	}

}
