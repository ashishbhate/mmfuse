package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

// MMClient wraps a mattermost client and some data
type MMClient struct {
	client *model.Client4
	user   *model.User

	teams     []*model.Team
	teamsEtag string

	channelsInTeam      map[string][]*model.Channel
	teamsToChannelsEtag map[string]string // map[teamId]etag
}

// NewMMClient Returns a new MMClient
func NewMMClient(server, username, password, caCertPath string) (*MMClient, error) {
	client := model.NewAPIv4Client(server)

	if caCertPath != "" {
		fixedClient := loadCert(caCertPath)
		client.HttpClient = fixedClient
	}

	var resp *model.Response
	user, resp := client.Login(username, password)
	if resp.StatusCode != 200 {
		return nil, resp.Error
	}
	return &MMClient{
		client:              client,
		user:                user,
		channelsInTeam:      make(map[string][]*model.Channel),
		teamsToChannelsEtag: make(map[string]string),
	}, nil
}

// GetTeams returns a list of mattermost teams the user belongs to
func (mc *MMClient) GetTeams() ([]*model.Team, error) {
	teams, resp := mc.client.GetTeamsForUser(mc.user.Id, mc.teamsEtag)
	if resp.StatusCode != 200 {
		return nil, resp.Error
	}
	if resp.Etag != "" && resp.Etag == mc.teamsEtag {
		return mc.teams, nil
	}
	mc.teams = teams
	mc.teamsEtag = resp.Etag
	return teams, nil
}

// GetChannels returns a list of mattermost channels in a given team the user belongs to
func (mc *MMClient) GetChannels(teamId string) ([]*model.Channel, error) {
	channels, resp := mc.client.GetChannelsForTeamForUser(
		teamId, mc.user.Id, false, mc.teamsToChannelsEtag[teamId])
	if resp.StatusCode != 200 {
		return nil, resp.Error
	}
	if resp.Etag != "" && resp.Etag == mc.teamsToChannelsEtag[teamId] {
		return mc.channelsInTeam[teamId], nil
	}
	mc.channelsInTeam[teamId] = channels
	mc.teamsToChannelsEtag[teamId] = resp.Etag
	return channels, nil
}

// NormalizedChannel models a mattermost channel with normalized names.
// A mattermost channel can be a public channel, a private channel, a group
// or a direct message. This object tries to model a simple abstraction.
type NormalizedChannel struct {
	Id, Name string
}

// GetNormalizedChannels returns a list of NormalizedChannel in the given team
// that the user belongs to
func (mc *MMClient) GetNormalizedChannels(teamId string) ([]*NormalizedChannel, error) {
	channels, err := mc.GetChannels(teamId)
	if err != nil {
		return nil, err
	}
	normalizedChannels := make([]*NormalizedChannel, 0, len(channels))
	for _, c := range channels {
		name, err := mc.normalizeChannelName(c)
		if err != nil {
			return nil, err
		}
		normalizedChannels = append(normalizedChannels,
			&NormalizedChannel{
				Id:   c.Id,
				Name: name,
			},
		)
	}
	return normalizedChannels, nil
}

func (mc *MMClient) normalizeChannelName(c *model.Channel) (string, error) {
	name := c.DisplayName
	switch c.Type {
	case model.CHANNEL_DIRECT:
		otherUserID := c.GetOtherUserIdForDM(mc.user.Id)
		user, resp := mc.client.GetUser(otherUserID, "")
		switch resp.StatusCode {
		case 200:
			name = user.Username
		case 404:
			name = "Missing_User_" + otherUserID
		default:
			return "", resp.Error
		}
		name = "[D]" + name
	case model.CHANNEL_GROUP:
		users, resp := mc.client.GetUsersInChannel(c.Id, 0, 100, "")
		if resp.StatusCode != 200 {
			return "", resp.Error
		}
		name = "[G]" + model.GetGroupDisplayNameFromUsers(users, true)
	case model.CHANNEL_PRIVATE:
		name = "[P]" + c.DisplayName
	}
	return name, nil
}

// GetChannelUnread returns a string containing unread messages in the given channel
func (mc *MMClient) GetChannelUnread(channelId string) (*model.PostList, error) {
	postList, resp := mc.client.GetPostsAroundLastUnread(mc.user.Id, channelId, 0, 200)
	if resp.StatusCode != 200 {
		return nil, resp.Error
	}
	return postList, nil
}

// MarkChannelAsRead marks the given channel as read
func (mc *MMClient) MarkChannelAsRead(channelId string) error {
	_, resp := mc.client.ViewChannel(mc.user.Id, &model.ChannelView{PrevChannelId: channelId})
	if resp.StatusCode != 200 {
		return resp.Error
	}
	return nil
}

// FormatPostsForDisplay formosts a mattermost post for display
func (mc *MMClient) FormatPostsForDisplay(postList *model.PostList) (string, error) {
	var text strings.Builder
	order := postList.Order
	for i := len(order) - 1; i >= 0; i-- {
		post := postList.Posts[order[i]]
		user, resp := mc.client.GetUser(post.UserId, "")
		if resp.StatusCode != 200 {
			return "", resp.Error
		}
		id := post.Id
		if post.ParentId != "" {
			id = post.ParentId
		}
		fmt.Fprintf(&text, "%s <%s> %s: %s\n",
			humanTime(post.CreateAt), id[len(id)-6:], user.Username, post.Message)
	}
	return text.String(), nil
}
