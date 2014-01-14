// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ssh_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	gc "launchpad.net/gocheck"

	"launchpad.net/juju-core/testing"
	jc "launchpad.net/juju-core/testing/checkers"
	"launchpad.net/juju-core/testing/testbase"
	"launchpad.net/juju-core/utils/ssh"
)

type ExecuteSSHCommandSuite struct {
	testbase.LoggingSuite
	testbin string
	fakessh string
}

var _ = gc.Suite(&ExecuteSSHCommandSuite{})

func (s *ExecuteSSHCommandSuite) SetUpTest(c *gc.C) {
	s.LoggingSuite.SetUpTest(c)
	s.testbin = c.MkDir()
	s.fakessh = filepath.Join(s.testbin, "ssh")
	newPath := s.testbin + ":" + os.Getenv("PATH")
	s.PatchEnvironment("PATH", newPath)
}

func (s *ExecuteSSHCommandSuite) fakeSSH(c *gc.C, cmd string) {
	err := ioutil.WriteFile(s.fakessh, []byte(cmd), 0755)
	c.Assert(err, gc.IsNil)
}

func (s *ExecuteSSHCommandSuite) TestCaptureOutput(c *gc.C) {
	s.fakeSSH(c, echoSSH)

	rc, stdout, stderr, err := ssh.ExecuteCommandOnMachine(ssh.ExecParams{
		Host:    "hostname",
		Command: "sudo apt-get update\nsudo apt-get upgrade",
		Timeout: testing.ShortWait,
	})

	c.Assert(err, gc.IsNil)
	c.Assert(rc, gc.Equals, 0)
	c.Assert(string(stdout), gc.Equals, "sudo apt-get update\nsudo apt-get upgrade\n")
	c.Assert(string(stderr), gc.Equals,
		"-o StrictHostKeyChecking no -o PasswordAuthentication no hostname -- /bin/bash -s\n")
}

func (s *ExecuteSSHCommandSuite) TestIdentityFile(c *gc.C) {
	s.fakeSSH(c, echoSSH)

	_, _, stderr, err := ssh.ExecuteCommandOnMachine(ssh.ExecParams{
		IdentityFile: "identity-file",
		Host:         "hostname",
		Timeout:      testing.ShortWait,
	})

	c.Assert(err, gc.IsNil)
	c.Assert(string(stderr), jc.Contains, " -i identity-file ")
}

func (s *ExecuteSSHCommandSuite) TestTimoutCaptureOutput(c *gc.C) {
	s.fakeSSH(c, slowSSH)

	rc, stdout, stderr, err := ssh.ExecuteCommandOnMachine(ssh.ExecParams{
		IdentityFile: "identity-file",
		Host:         "hostname",
		Command:      "ignored",
		Timeout:      testing.ShortWait,
	})

	c.Check(err, gc.ErrorMatches, "command timed out")
	c.Assert(rc, gc.Equals, 0)
	c.Assert(string(stdout), gc.Equals, "stdout\n")
	c.Assert(string(stderr), gc.Equals, "stderr\n")
}

func (s *ExecuteSSHCommandSuite) TestCapturesReturnCode(c *gc.C) {
	s.fakeSSH(c, passthroughSSH)

	rc, stdout, stderr, err := ssh.ExecuteCommandOnMachine(ssh.ExecParams{
		IdentityFile: "identity-file",
		Host:         "hostname",
		Command:      "echo stdout; exit 42",
		Timeout:      testing.ShortWait,
	})

	c.Check(err, gc.IsNil)
	c.Assert(rc, gc.Equals, 42)
	c.Assert(string(stdout), gc.Equals, "stdout\n")
	c.Assert(string(stderr), gc.Equals, "")
}

// echoSSH outputs the command args to stderr, and copies stdin to stdout
var echoSSH = `#!/bin/bash
# Write the args to stderr
echo "$*" >&2
cat /dev/stdin
`

// slowSSH sleeps for a while after outputting some text to stdout and stderr
var slowSSH = `#!/bin/bash
echo "stderr" >&2
echo "stdout"
sleep 5s
`

// passthroughSSH creates an ssh that executes stdin.
var passthroughSSH = `#!/bin/bash -s`
