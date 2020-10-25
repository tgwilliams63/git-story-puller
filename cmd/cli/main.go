package main

import (
	"context"
	"flag"
	"os"
	"regexp"

	"github.com/google/go-github/v32/github"
	mylog_debugger "github.com/tgwilliams63/mylog-debugger"
	"golang.org/x/oauth2"
)

var ML mylog_debugger.MyLog

func main() {
	ML.SetLogLevel(os.Getenv("LOG_LEVEL"))
	accessToken := os.Getenv("GITHUB_TOKEN")

	var repoOwner string
	var repoName string
	var prBranch string
	var baseTag string
	var currCommit string
	flag.StringVar(&repoOwner, "owner", "", "Repo Owner Name from URL")
	flag.StringVar(&repoName, "repo", "", "Name of Repo")
	flag.StringVar(&prBranch, "branch", "", "Branch being deployed")
	flag.StringVar(&baseTag, "prevRef", "", "Ref of previous deploy. Can be a tag or commit id")
	flag.StringVar(&currCommit, "currRef", "", "Ref being deployed. Can be a tag or commit")
	flag.Parse()

	opt := github.ListOptions{}
	//wbcArray := make(map[plumbing.Hash]bool)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	diffCommits := make(map[string]bool)
	cc, _, err := client.Repositories.CompareCommits(ctx, repoOwner, repoName, baseTag, currCommit)
	ML.Check(err)
	for _, commit := range cc.Commits {
		ML.Debug("Commit ", commit.GetSHA(), "is not included in ", baseTag)
		diffCommits[commit.GetSHA()] = true
	}

	ML.Debug("There are ", len(diffCommits), " commits different between ", baseTag, " and ", currCommit)

	prNums := make(map[int]bool)
	var id string
	var v1Stories []string
	re := regexp.MustCompile(`VersionOne Stories: (?P<ID>[a-zA-Z]{1}-\d*)`)
	for commit := range diffCommits {
		ML.Debug("Finding PRs for Commit SHA: ", commit)
		prs, _, err := client.PullRequests.ListPullRequestsWithCommit(ctx, repoOwner, repoName, commit, &github.PullRequestListOptions{State: "closed", Base: prBranch, ListOptions: opt})
		ML.Check(err)
		for _, pr := range prs {
			prNum := pr.GetNumber()
			prBody := pr.GetBody()
			if !prNums[prNum] {
				prNums[prNum] = true
				ML.Debug("PR #", prNum, " was introduced in this release")
				ML.Debug("PR #", prNum, " Body: \"", prBody, "\"")
				regexResults := re.FindStringSubmatch(prBody)
				if len(regexResults) > 0 {
					id = regexResults[len(regexResults)-1]
					v1Stories = append(v1Stories, id)
					ML.Debug("PR #", prNum, " V1 Story ID: ", id)
				} else {
					ML.Print("No VersionOne Story ID Found for PR #", prNum)
				}
			}
		}
	}

	ML.Print("V1 Stories: ", v1Stories)
}
