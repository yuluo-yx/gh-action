package retest

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/actions-go/toolkit/core"
	"github.com/actions-go/toolkit/github"
)

var (
	githubClient = github.NewClient()
)

func InitRetestCommands() *Runtime {

	commentInput, _ := core.GetInput("comment-id")
	comment, _ := strconv.Atoi(commentInput)
	pr, _ := core.GetInput("pr-url")
	nwo := os.Getenv("GITHUB_REPOSITORY")
	debug := os.Getenv("CI_DEBUG") != "" && os.Getenv("CI_DEBUG") != "false"
	var repo, owner string

	if nwo != "" {
		owner = strings.Split(nwo, "/")[0]
		repo = strings.Split(nwo, "/")[1]
	} else {
		log.Fatal("GITHUB_REPOSITORY must not be nil")
	}

	return &Runtime{
		Pr:      pr,
		Comment: comment,
		Repo:    repo,
		Nwo:     nwo,
		Owner:   owner,
		Debug:   debug,
	}

}

func getPRNumber(pr string) int {

	prSplit := strings.Split(pr, "/")
	prNumber, _ := strconv.Atoi(prSplit[len(prSplit)-1])

	return prNumber
}

func getPR(rt *Runtime) *PullRequest {

	if rt.Pr == "" {

		log.Fatal("env.pr url is nil")
	}

	pr, prResp, err := githubClient.PullRequests.Get(
		context.Background(),
		rt.Owner,
		rt.Repo,
		getPRNumber(rt.Pr),
	)

	if pr == nil && (prResp.StatusCode != 200 || prResp.StatusCode != 201) && err != nil {

		log.Fatal("pr not found, err: ", err)
	}

	return &PullRequest{
		Branch: pr.Head.GetRef(),
		Number: pr.GetNumber(),
		Commit: pr.Head.GetSHA(),
	}

}

func addReaction(rt *Runtime, content string) bool {

	commentR, response, err := githubClient.Reactions.CreateIssueCommentReaction(
		context.Background(),
		rt.Owner,
		rt.Repo,
		int64(rt.Comment),
		content,
	)

	fmt.Printf("%v\n", commentR.GetContent())

	if (response.StatusCode != 200 || response.StatusCode != 201) && err != nil {

		log.Fatal("failed to add reaction, error: ", err)
		return false
	}

	return true

}

func getRetestActionTask(rt *Runtime, pr *PullRequest) (failedChecks []*GHRetest) {

	ref, response, err := githubClient.Checks.ListCheckRunsForRef(
		context.Background(),
		rt.Owner,
		rt.Repo,
		pr.Commit,
		nil,
	)

	if (response.StatusCode != 200 || response.StatusCode != 201) && err != nil {

		log.Fatal("failed to get check runs, error: ", err)

	}

	fmt.Printf("ListCheckRunsForRef: %v\n", ref)

	for i, run := range ref.CheckRuns {
		fmt.Printf("CheckRuns 检查到任务: %d, %v\n", i, run)
	}

	return failedChecks
}

func stringPtr(str string) *string {

	return &str
}

func retestRuns(pr *PullRequest, rt *Runtime, failedChecks []*GHRetest) (result *GHRetestResult) {

	errorNum := 0

	fmt.Printf("进入 retest runs 函数！\n")

	return &GHRetestResult{
		Error:    errorNum,
		Retested: len(failedChecks),
	}
}

func retest() {

	rt := InitRetestCommands()
	rt.Debug = true
	pr := getPR(rt)
	failedCheckList := getRetestActionTask(rt, pr)

	if len(failedCheckList) == 0 {

		log.Println("no failed checks found")
		return
	}

	if rt.Debug {
		log.Printf("Runtime info: %v\n: ", rt)
		log.Printf("pr info: %v", pr)
	}

	result := retestRuns(pr, rt, failedCheckList)
	if result.Error != 0 {

		addReaction(rt, "-1")
	}
	if result.Error == 0 {

		log.Println("all checks have been restarted")
		addReaction(rt, "rocket")
	} else {

		log.Printf("failed to restart some checks, error times: %v\n", result.Error)
		addReaction(rt, "confused")
	}

}

func Run() {

	defer func() {
		if err := recover(); err != nil {

			log.Println("retest error: ", err)
			core.SetFailedf("Retest action failure, error is ", err)
		}
	}()

	retest()
}
