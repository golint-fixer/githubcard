package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/brotherlogic/goserver/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pbdi "github.com/brotherlogic/discovery/proto"
	pb "github.com/brotherlogic/githubcard/proto"
)

func findServer(name string) (string, int) {
	conn, _ := grpc.Dial(utils.Discover, grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	rs, _ := registry.ListAllServices(context.Background(), &pbdi.Empty{})

	for _, r := range rs.Services {
		if r.Name == name {
			log.Printf("%v -> %v", name, r)
			return r.Ip, int(r.Port)
		}
	}

	return "", -1
}

func main() {

	if len(os.Args) <= 1 {
		fmt.Printf("Commands: list run\n")
	} else {
		switch os.Args[1] {
		case "add":
			host, port := findServer("githubcard")

			conn, err := grpc.Dial(host+":"+strconv.Itoa(port), grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Cannot dial master: %v", err)
			}
			defer conn.Close()

			registry := pb.NewGithubClient(conn)
			res, err := registry.AddIssue(context.Background(), &pb.Issue{Title: "Testing", Service: "githubcard"})
			log.Printf("RESP %v with %v", res, err)
		}
	}
}
