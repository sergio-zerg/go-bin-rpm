package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SetupRepo creates an rpm repository
func SetupRepo(reposlug, ghToken, email, version, archs, outbuild string, push, keep bool) {

	x := strings.Split(reposlug, "/")
	user := x[0]
	name := x[1]

	gopath := os.Getenv("GOPATH")
	repoPath := filepath.Join(gopath, "src", "github.com", reposlug)
	fmt.Println("repoPath", repoPath)

	setupGitRepo(repoPath, reposlug, user, email)
	chdir(repoPath)

	if maybesudo(`dnf install createrepo git -y --quiet`) != nil {
		maybesudo(`yum install createrepo git -y --quiet`)
	}

	if tryexec(`latest -v`) != nil {
		exec(`git clone https://github.com/mh-cbon/latest.git %v/src/github.com/mh-cbon/latest`, gopath)
		exec(`go install github.com/mh-cbon/latest`)
	}
	if tryexec(`gh-api-cli -v`) != nil {
		exec(`latest -repo=%v`, "mh-cbon/gh-api-cli")
	}

	resetGit(repoPath)
	tryexec(`git remote -vv`)
	tryexec(`git branch -aav`)
	getBranchGit(repoPath, reposlug, "gh-pages", "rpmorigin")
	defer tryexec(`git remote rm rpmorigin`)
	tryexec(`git remote -vv`)
	tryexec(`git branch -aav`)

	resetGit(repoPath)
	exec(`git status`)

	for _, arch := range strings.Split(archs, ",") {

		darch := arch
		if darch == "386" {
			darch = "i386"
		} else if darch == "amd64" {
			darch = "x86_64"
		}

		archOut := outbuild + "/" + darch
		removeAll(archOut)
		mkdirAll(archOut)
		exec(`gh-api-cli dl-assets -t %q -o %v -r %v -g '*%v*rpm' -out '%v/%%r-%%v_%%a.%%e'`, ghToken, user, name, arch, archOut)

		chdir(archOut)
		exec(`createrepo .`)
		tryexec("ls -al")
	}

	chdir(repoPath)

	desc := getexec(`rpm -qip %v/*/*.rpm | grep Summary | cut -d ':' -f2 | cut -d ' ' -f2- | tail -n 1`, outbuild)
	confFile := fmt.Sprintf(`%v/%v.repo`, outbuild, name)
	conf := `[%v]
name=%v
baseurl=https://%v.github.io/%v/rpm/$basearch/
enabled=1
skip_if_unavailable=1
gpgcheck=0`
	conf = fmt.Sprintf(conf, name, desc, user, name)
	writeFile(confFile, conf)

	tryexec(`git status`)

	fmt.Println("push", push)
	if push {
		commitPushGit(repoPath, ghToken, reposlug, "gh-pages", "rpm repository")
		if keep == false {
			removeAll(outbuild)
		}
	}
}
