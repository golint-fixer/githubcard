package main

import (
	"context"
	"encoding/json"

	pb "github.com/brotherlogic/githubcard/proto"
)

type addResponse struct {
	Number int32
}

//AddIssue adds an issue to github
func (g *GithubBridge) AddIssue(ctx context.Context, in *pb.Issue) (*pb.Issue, error) {
	b, err := g.AddIssueLocal("brotherlogic", in.GetService(), in.GetTitle(), in.GetBody())
	r := &addResponse{}
	json.Unmarshal(b, &r)
	in.Number = r.Number
	return in, err
}

//GetIssue gets an issue from github
func (g *GithubBridge) GetIssue(ctx context.Context, in *pb.Issue) (*pb.Issue, error) {
	b, err := g.GetIssueLocal("brotherlogic", in.GetService(), int(in.GetNumber()))
	return b, err
}
