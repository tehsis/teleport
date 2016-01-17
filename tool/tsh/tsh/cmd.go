/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tsh

import (
	"net"

	"github.com/gravitational/teleport/lib/client"

	"github.com/gravitational/teleport/Godeps/_workspace/src/github.com/gravitational/trace"
	"github.com/gravitational/teleport/Godeps/_workspace/src/golang.org/x/crypto/ssh"
	"github.com/gravitational/teleport/Godeps/_workspace/src/golang.org/x/crypto/ssh/agent"
	"github.com/gravitational/teleport/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v2"
)

func RunTSH(args []string) error {
	app := kingpin.New("tsh", "teleport SSH client")

	user := app.Flag("user", "SSH user").Required().String()
	sshAgentAddress := app.Flag("ssh-agent", "SSH agent address").OverrideDefaultFromEnvar("SSH_AUTH_SOCK").String()
	sshAgentNetwork := app.Flag("ssh-agent-network", "SSH agent address network type('tcp','unix' etc.)").Default("unix").String()
	webProxyAddress := app.Flag("web-proxy", "Web proxy address(used for login)").String()
	loginTTL := app.Flag("login-ttl", "Temporary ssh certificate will work for that time").Default("10h").Duration()

	connect := app.Command("connect", "Helper operations with SSH keypairs")
	connectAddress := connect.Arg("address", "Target server address").Required().String()
	connectProxy := connect.Flag("proxy", "Optional proxy address").String()
	connectCommand := connect.Flag("command", "Run proveded command instead of shell").String()

	upload := app.Command("upload", "Helper operations with SSH keypairs")
	uploadAddress := upload.Arg("address", "Target server address").Required().String()
	uploadProxy := upload.Flag("proxy", "Optional proxy address").String()
	uploadLocalSource := upload.Flag("source", "Local source path").Required().String()
	uploadRemoteDest := upload.Flag("dest", "Remote destination path").Required().String()

	download := app.Command("download", "Helper operations with SSH keypairs")
	downloadAddress := download.Arg("address", "Target server address").Required().String()
	downloadProxy := download.Flag("proxy", "Optional proxy address").String()
	downloadLocalDest := download.Flag("dest", "Local destination path").Required().String()
	downloadRemoteSource := download.Flag("source", "Remote source path").Required().String()
	downloadRecursively := download.Flag("r", "Source path is directory").Bool()

	getServers := app.Command("get-servers", "Returns list of servers")
	getServersProxy := getServers.Flag("proxy", "Target proxy address").String()
	getServersLabelName := getServers.Flag("label", "Label name").String()
	getServersLabelValue := getServers.Flag("value", "Label value regexp").String()

	selectedCommand := kingpin.MustParse(app.Parse(args[1:]))

	standartSSHAgent, err := connectToSSHAgent(*sshAgentNetwork, *sshAgentAddress)
	if err != nil {
		return trace.Wrap(err)
	}
	teleportFileSSHAgent, err := client.GetLocalAgent()
	if err != nil {
		return trace.Wrap(err)
	}
	passwordCallback := client.GetPasswordFromConsole(*user)

	authMethods := []ssh.AuthMethod{
		client.AuthMethodFromAgent(standartSSHAgent),
		client.AuthMethodFromAgent(teleportFileSSHAgent),
		client.GenerateCertificateCallback(
			teleportFileSSHAgent,
			*user,
			passwordCallback,
			*webProxyAddress,
			*loginTTL,
		),
	}

	err = trace.Errorf("No command")

	switch selectedCommand {
	case connect.FullCommand():
		err = Connect(*user, *connectAddress, *connectProxy, *connectCommand, authMethods)
	case upload.FullCommand():
		err = Upload(*user, *uploadAddress, *uploadProxy, *uploadLocalSource,
			*uploadRemoteDest, authMethods)
	case download.FullCommand():
		err = Download(*user, *downloadAddress, *downloadProxy,
			*downloadRemoteSource, *downloadLocalDest,
			*downloadRecursively, authMethods)
	case getServers.FullCommand():
		err = GetServers(*user, *getServersProxy, *getServersLabelName,
			*getServersLabelValue, authMethods)
	}

	return err
}

func connectToSSHAgent(network, address string) (agent.Agent, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return agent.NewClient(conn), nil

}