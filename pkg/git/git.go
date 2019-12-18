/*******************************************************************************
*
* Copyright 2019 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	errGitNotInstalled = errors.New("git is not installed")
	errGitNotRemote    = errors.New("git remote has no remote configured")
)

// Git ...
type Git struct {
	cmd,
	branchName,
	remoteName,
	authorName,
	authorEmail string
	defaultArgs []string
}

// New returns a new Git or an error.
func New(path string) (*Git, error) {
	authorName, ok := os.LookupEnv("HELM_DEPENDENCY_AUTHOR_NAME")
	if !ok {
		authorName = "Helm Bot"
	}

	authorEmail, ok := os.LookupEnv("HELM_DEPENDENCY_AUTHOR_MAIL")
	if !ok {
		authorEmail = ""
	}

	g := &Git{
		cmd:         "git",
		branchName:  "master",
		remoteName:  "origin",
		authorName:  authorName,
		authorEmail: authorEmail,
		defaultArgs: []string{"-C", path},
	}

	if err := g.verify(); err != nil {
		return nil, err
	}

	return g, g.init()
}

// Commit adds and commits all changes.
func (g *Git) Commit(message string) (string, error) {
	return g.run("commit", "--all", "--author", fmt.Sprintf("%s <%s>", g.authorName, g.authorEmail), "--message", message)
}

// Diff shows the changes.
func (g *Git) Diff() (string, error) {
	return g.run("diff")
}

// PushToMaster rebases and pushes the commit(s) to upstream.
func (g *Git) RebaseAndPushToMaster() (string, error) {
	if out, err := g.PullRebase(); err != nil {
		return out, err
	}

	return g.run("push", fmt.Sprintf("%s/%s", g.remoteName, g.branchName))
}

// PullRebase pulls and rebases.
func (g *Git) PullRebase() (string, error) {
	return g.run("pull", "--rebase")
}

func (g *Git) init() error {
	if _, err := g.run("config", "user.name", g.authorName); err != nil {
		return err
	}

	_, err := g.run("config", "user.email", g.authorEmail)
	return err
}

// verify checks if git is installed and the given repository has a remote configured.
func (g *Git) verify() error {
	const (
		notFound     = "not found"
		noSuchRemote = "No such remote"
	)

	res, err := g.run("version")
	if err != nil && strings.Contains(err.Error(), notFound) || strings.Contains(res, notFound) {
		return errGitNotInstalled
	}

	res, err = g.GetRemoteURL(g.remoteName)
	if err != nil && strings.Contains(err.Error(), noSuchRemote) || strings.Contains(res, noSuchRemote) {
		return errGitNotRemote
	}

	return nil
}

func (g *Git) GetRemoteURL(name string) (string, error) {
	return g.run("remote", "get-url", name)
}

func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command(g.cmd, append(g.defaultArgs, args...)...)
	var stdOut bytes.Buffer
	cmd.Stdout = &stdOut

	if err := cmd.Start(); err != nil {
		return "", err
	}

	err := cmd.Wait()
	return stdOut.String(), err
}
