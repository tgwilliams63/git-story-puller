package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/go-git/go-git/v5/plumbing"

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

	fs := memfs.New()
	storer := memory.NewStorage()
	//baseRef := plumbing.NewReferenceFromStrings("base", fmt.Sprintf("refs/tags/%s", baseTag))
	baseRef := plumbing.NewTagReferenceName(baseTag)
	ML.Print(baseRef.String())

	r, err := git.Clone(storer, fs, &git.CloneOptions{
		URL:           fmt.Sprintf("https://github.com/%s/%s.git", repoOwner, repoName),
		ReferenceName: baseRef,
	})
	ML.Check(err)

	rh, err := r.Head()
	ML.Check(err)

	ri, err := r.Log(&git.LogOptions{From: rh.Hash()})
	ML.Check(err)

	err = ri.ForEach(func(c *object.Commit) error {
		ML.Print(c.Hash)
		return nil
	})

	//fileInfo, err := fs.Stat("./test.txt")
	file, err := fs.Open("./test.txt")
	ML.Check(err)
	fileBytes, err := ioutil.ReadAll(file)
	ML.Check(err)
	ML.Print(string(fileBytes))

}
