package main

import (
	"context"
	"io"
         "crypto/tls"
         "net/http/httptrace"
         // "net"
	 "strconv"
         "time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
        "google.golang.org/grpc/credentials"
	"grpc-health-check/proto"
         grpcMetadata "google.golang.org/grpc/metadata"
         // "github.com/johanbrandhorst/grpc-auth-example/insecure"
)

type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}

func streamRequest() {

        serverAddr := "cixing-http2-new.westus2-main.inference.ml.azure.com"
        port := "443"
        token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IkQ5RTJFQzI1NDFEMDkwMTc0REU4OEFCMTIzMDA2MjMwMzZBNTI5QUUiLCJ0eXAiOiJKV1QifQ.eyJjYW5SZWZyZXNoIjoiRmFsc2UiLCJ3b3Jrc3BhY2VJZCI6ImE4NjAwY2ZjLWM5Y2UtNGFmNS05NjcxLTYwNjM4YTRhNDIxMyIsInRpZCI6IjcyZjk4OGJmLTg2ZjEtNDFhZi05MWFiLTJkN2NkMDExZGI0NyIsIm9pZCI6ImZmZmMxYzY2LTI3NWYtNDkzNS1iYjA0LTcwYTc2MGM4MmZkYSIsImFjdGlvbnMiOiJbXCJNaWNyb3NvZnQuTWFjaGluZUxlYXJuaW5nU2VydmljZXMvd29ya3NwYWNlcy9vbmxpbmVFbmRwb2ludHMvc2NvcmUvYWN0aW9uXCJdIiwiZW5kcG9pbnROYW1lIjoiY2l4aW5nLWh0dHAyLW5ldyIsInNlcnZpY2VJZCI6ImNpeGluZy1odHRwMi1uZXciLCJleHAiOjE2MzczNzMzNzIsImlzcyI6ImF6dXJlbWwiLCJhdWQiOiJhenVyZW1sIn0.CLZe799Y8PjuMpR7vTF6n4Y2Gx4KmHzJ_ZAQgiQ9owrEy0x8v9a4S9U2eoyjN5lroUHA6cMaomE79WVbf7SFRP88-UJet3tfQrOmO8v6sYXceZT0zYI4Pe6I5eos1Wxso9YwcXl47YDef_szr-qmy6TJLaQSa6TDpFyJPVqYh9qomJ3bYNgMJ5ROd39oxZegAHLKE6Z8fn_gSV-dH-zBa_mMSESSDpBSmizG7OqWaaNJy0bGhkz3JSW1k6b8XeQEwHIYpAgJmlG3Z7pqKZmScH7m3NnK7KGWaHkSHXrp2-CfREPmS7vO14iESs7WUd5b5LeigPuh7LnMm8OEjDasdQ"

        ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

        trace := &httptrace.ClientTrace {
 		GotConn: func(info httptrace.GotConnInfo) {
			// connGot = time.Now()
			logrus.Println("Connection reused %v\n",info.Reused)
		},
        }

        ctx_new := httptrace.WithClientTrace(ctx, trace)

        config := &tls.Config{
         InsecureSkipVerify: false,
        }

        cred := credentials.NewTLS(config)

        // conn, err := grpc.DialContext(ctx, net.JoinHostPort(serverAddr, port),
        conn, err := grpc.DialContext(ctx_new, serverAddr + ":" + port, 
        grpc.WithTransportCredentials(cred), 
        grpc.WithPerRPCCredentials(tokenAuth{
        token: token,
        }),
        ) 
       
        // conn, err := grpc.Dial("cixing-http2.westus2-main.inference.ml.azure.com:443", grpc.WithTransportCredentials(credentials.NewTLS(config)), grpc.WithBlock())

        // serverAddr := ":5000"   

	// conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(creds))
	// conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		logrus.Fatalf("Couldn't dial %v", err)
	}

        ctx_new = grpcMetadata.AppendToOutgoingContext(ctx_new, "authorization", "Bearer "+ token)
        

	defer conn.Close()

        helloClient := proto.NewGreetServiceClient(conn)

        for i := 0; i < 10; i++ {
	stream, err := helloClient.Hello(ctx_new, &proto.HelloRequest{
		Hello: "World" + strconv.Itoa(i),
	})

        if err != nil {
               logrus.Fatalf("Couldn't dial grpc server %v", err)
        }

	for {
		streamData, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			//logrus.Fatalf("%v.Greet = _, %v", helloClient, err)
                        logrus.Println("stream call failed %v", err)
                        break
		}
		logrus.Println(streamData)
	}
      }
}

func main() {
	

        for i := 0; i < 10; i++ {
            logrus.Println("Here is the index %d", i)
            go streamRequest()
        }

        time.Sleep(10 * time.Second)

	logrus.Println("Doing a health check on the server")

}
