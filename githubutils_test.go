package main

import (
	"testing"

	pb "github.com/brotherlogic/githubcard/proto"
	"golang.org/x/net/context"
)

func TestProcSticky(t *testing.T) {
	g := InitTest()
	g.issues = append(g.issues, &pb.Issue{Service: "blah", Title: "blah", Body: "blah"})
	g.procSticky(context.Background())

	if len(g.issues) != 0 {
		t.Errorf("Issue was not added: %v", g.issues)
	}
}
