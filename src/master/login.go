/****************************************************************************
 * This file is part of Builder.
 *
 * Copyright (C) 2015 Pier Luigi Fiorini
 *
 * Author(s):
 *    Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
 *
 * $BEGIN_LICENSE:AGPL3+$
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * $END_LICENSE$
 ***************************************************************************/

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/hawaii-desktop/builder/src/logging"
	"github.com/plimble/ace"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"strings"
	"time"
)

/*
 * Helpful resources:
 *   https://blog.kowalczyk.info/article/f/Accessing-GitHub-API-from-Go.html
 *   https://developer.github.com/v3/oauth/#scopes
 */

// OAuth configuration for GitHub
var oauthConf *oauth2.Config = nil

func generateNewOauthState(id string) string {
	var buffer bytes.Buffer
	buffer.WriteString(id)
	buffer.WriteString("#")
	buffer.WriteString(time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339))
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func oauthState(c *ace.C) string {
	session := c.Sessions("authentication")
	id := session.ID[:15]
	state := generateNewOauthState(id)
	session.Set("State", state)
	session.Set("ID", id)
	return state
}

func validateOauthState(c *ace.C, state string) (bool, error) {
	session := c.Sessions("authentication")
	sessionId := session.GetString("ID", "")
	sessionState := session.GetString("State", "")

	if sessionState != state {
		return false, fmt.Errorf("doesn't match with the one from session")
	}

	b, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return false, err
	}

	id := ""
	timeStr := ""
	for i := 0; i < len(b); i++ {
		if b[i] == '#' {
			id = string(b[:i])
			timeStr = string(b[i+1:])
			break
		}
	}

	if sessionId != id {
		return false, fmt.Errorf("session id doesn't match")
	}

	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return false, err
	}

	ok := t.UTC().After(time.Now().UTC())
	if !ok {
		return ok, fmt.Errorf("invalid time")
	}
	return ok, nil
}

func loginHandler(c *ace.C) {
	if oauthConf == nil {
		oauthConf = &oauth2.Config{
			ClientID:     Config.GitHub.ClientID,
			ClientSecret: Config.GitHub.ClientSecret,
			Scopes:       []string{"user:email", "read:org"},
			Endpoint:     githuboauth.Endpoint,
		}
	}

	url := oauthConf.AuthCodeURL(oauthState(c), oauth2.AccessTypeOnline)
	c.Redirect(url)
}

func ssoGitHubHandler(c *ace.C) {
	if oauthConf == nil {
		msg := fmt.Sprintf("No oauth configuration")
		logging.Errorf("%s\n", msg)

		data := c.GetAll()
		data["ErrorMessage"] = msg
		c.HTML("login_failed.html", data)
		return
	}

	session := c.Sessions("authentication")

	// Verify state
	state := c.MustQueryString("state", "")
	ok, err := validateOauthState(c, state)
	if !ok {
		msg := fmt.Sprintf("Invalid OAuth state: %s", err)
		logging.Errorf("%s\n", msg)

		data := c.GetAll()
		data["ErrorMessage"] = msg
		c.HTML("login_failed.html", data)
		return
	}

	// Get a token out of the code
	code := c.MustQueryString("code", "")
	token, err := oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		msg := fmt.Sprintf("OAuth exchange failed: %s", err)
		logging.Errorf("%s\n", msg)

		data := c.GetAll()
		data["ErrorMessage"] = msg
		c.HTML("login_failed.html", data)
	}

	// Convert token to JSON in order to save it
	tokenJson, err := json.Marshal(token)
	if err != nil {
		msg := fmt.Sprintf("Failed to convert token to JSON: %s", err)
		logging.Errorf("%s\n", msg)

		data := c.GetAll()
		data["ErrorMessage"] = msg
		c.HTML("login_failed.html", data)
	}

	// Authenticate
	oauthClient := oauthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get("")
	if err != nil {
		msg := fmt.Sprintf("Unable to get GitHub user information: %s", err)
		logging.Errorf("%s\n", msg)

		data := c.GetAll()
		data["ErrorMessage"] = msg
		c.HTML("login_failed.html", data)
		return
	}
	logging.Infof("Logged in as GitHub user \"%s\", checking authorization...\n", *user.Login)

	// Check whether this user is part of the organization and also
	// determine permissions based on what teams the user belongs to
	var groups []string
	found := false
	orgs, _, err := client.Organizations.List(*user.Login, nil)
	if err == nil {
		for _, org := range orgs {
			if *org.Login == Config.GitHub.Organization {
				found = true
				break
			}
		}
	}
	if found {
		found = false
		teams, _, err := client.Organizations.ListTeams(Config.GitHub.Organization, nil)
		if err == nil {
			for _, team := range teams {
				for _, matchTeam := range strings.Split(Config.GitHub.Teams, ",") {
					if *team.Name == matchTeam {
						groups = append(groups, *team.Name)
						found = true
					}
				}
			}
		}
	}
	if found {
		logging.Infof("Authorization for \"%s\" is clear\n", *user.Login)
	} else {
		logging.Errorf("User \"%s\" is not part of '%s'", *user.Login, Config.GitHub.Organization)
		c.String(403, fmt.Sprintf("You must be part of the '%s' organization", Config.GitHub.Organization))
		return
	}

	// Save session
	session.Set("Token", tokenJson)
	session.Set("Groups", groups)
	session.Set("IsLoggedIn", true)
	session.Set("UserName", *user.Login)
	session.Save(c.Writer)
	session.GetBool("IsLoggedIn", false)
	session.GetString("UserName", "")

	// Go home
	c.Redirect("/")
}
