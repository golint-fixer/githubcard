package main

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/githubcard/proto"
)

type addResponse struct {
	Number int32
}

//AddIssue adds an issue to github
func (g *GithubBridge) AddIssue(ctx context.Context, in *pb.Issue) (*pb.Issue, error) {
	g.added[in.GetTitle()] = time.Now()
	b, err := g.AddIssueLocal("brotherlogic", in.GetService(), in.GetTitle(), in.GetBody())
	if err != nil {
		return nil, err
	}
	r := &addResponse{}
	err2 := json.Unmarshal(b, &r)
	if err2 != nil {
		return nil, err2
	}
	in.Number = r.Number
	return in, nil
}

//Get gets an issue from github
func (g *GithubBridge) Get(ctx context.Context, in *pb.Issue) (*pb.Issue, error) {
	b, err := g.GetIssueLocal("brotherlogic", in.GetService(), int(in.GetNumber()))
	return b, err
}
