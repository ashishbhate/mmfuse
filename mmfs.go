package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mattermost/mattermost-server/v5/model"
)

// MMFS models a mattermost fuse file system
//     |- root
//        |- team1
//        |  |- channel1
//        |  |  |- in
//        |  |  |- unread
//        |  |
//        |  |- DM1
//        |     |- in
//        |     |- unread
//        |
//        |- team1
//           |- channel1
//           |  |- in
//           |  |- unread
//           |
//           |- DM1
//              |- in
//              |- unread
type MMFS struct {
	mmClient *MMClient
	mmTeams  map[string]*MMTeam
}

// NewMMFS returns a new MMFS object
func NewMMFS(server, username, password, caCertPath string) (*MMFS, error) {
	c, err := NewMMClient(server, username, password, caCertPath)
	if err != nil {
		return nil, err
	}
	return &MMFS{
		mmClient: c,
	}, nil
}

// Root is called to obtain the Node for the file system root.
// satisfies the fs.Fuse interface
func (mmfs *MMFS) Root() (fs.Node, error) {
	teams, err := mmfs.mmClient.GetTeams()
	if err != nil {
		return nil, err
	}
	mmfs.mmTeams = make(map[string]*MMTeam)
	for _, team := range teams {
		mmfs.mmTeams[team.Name], err = NewMMTeam(team, mmfs.mmClient)
		if err != nil {
			return nil, err
		}
	}
	return mmfs, nil
}

// Attr fills attr with the standard metadata for the node.
// satisfies the fs.Node interface
func (mmfs *MMFS) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o555 // TODO fix permissions
	return nil
}

// Lookup returns a fs.Node that corresponds to the given entry inside the
// root directory, which in this case are directories corresponding to
// mattermost teams
// Satisfies the fs.NodeStringLookuper interface
func (mmfs *MMFS) Lookup(_ context.Context, name string) (fs.Node, error) {
	teamNode, ok := mmfs.mmTeams[name]
	if !ok {
		return nil, syscall.ENOENT
	}
	return teamNode, nil
}

// ReadDirAll returns all directory entries, i.e. mattermost teams, inside the root directory
// Satisfies the fs.HandleReadAller interface
func (mmfs *MMFS) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	dirs := make([]fuse.Dirent, 0, len(mmfs.mmTeams))
	for name := range mmfs.mmTeams {
		dirs = append(dirs, fuse.Dirent{
			Inode: mmfs.mmTeams[name].inode,
			Name:  name,
			Type:  fuse.DT_Dir,
		})
	}
	return dirs, nil
}

// MMTeam models a mattermost team as a FUSE directory
type MMTeam struct {
	mmClient   *MMClient
	id         string
	name       string
	inode      uint64
	mmChannels map[string]*MMChannel
}

// NewMMTeam returns a new MMTeam
func NewMMTeam(team *model.Team, client *MMClient) (*MMTeam, error) {
	channels, err := client.GetNormalizedChannels(team.Id)
	if err != nil {
		return nil, err
	}
	mmTeam := &MMTeam{
		mmClient:   client,
		id:         team.Id,
		name:       team.Name,
		inode:      fs.GenerateDynamicInode(1, team.Name),
		mmChannels: make(map[string]*MMChannel),
	}
	for _, channel := range channels {
		mmChannel := &MMChannel{
			id:     channel.Id,
			name:   channel.Name,
			inode:  fs.GenerateDynamicInode(mmTeam.inode, channel.Name),
			client: client,
		}
		mmChannel.unreadFile = &UnreadFile{
			mmclient:  client,
			channelID: channel.Id,
			inode:     fs.GenerateDynamicInode(mmChannel.inode, "unread"),
		}
		mmChannel.inFile = &InFile{
			mmclient:  client,
			channelID: channel.Id,
			inode:     fs.GenerateDynamicInode(mmChannel.inode, "in"),
		}
		mmTeam.mmChannels[channel.Name] = mmChannel
	}
	return mmTeam, nil
}

// Attr fills attr with the standard metadata for the MMTeam node.
// satisfies the fs.Node interface
func (mmt *MMTeam) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = mmt.inode
	a.Mode = os.ModeDir | 0o555 // TODO restrict permissions
	return nil
}

// Lookup returns a fs.Node that corresponds to the given entry inside this
// directory (team), which in this case are channels in a team
// Satisfies the fs.NodeStringLookuper interface
func (mmt *MMTeam) Lookup(_ context.Context, name string) (fs.Node, error) {
	channelNode, ok := mmt.mmChannels[name]
	if !ok {
		return nil, syscall.ENOENT
	}
	return channelNode, nil
}

// ReadDirAll returns all directory entries, i.e. mattermost channels, inside the directory
// corresponding to this team
// Satisfies the fs.HandleReadAller interface
func (mmt *MMTeam) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	dirs := make([]fuse.Dirent, 0, len(mmt.mmChannels))
	for name := range mmt.mmChannels {
		dirs = append(dirs, fuse.Dirent{
			Inode: mmt.mmChannels[name].inode,
			Name:  name,
			Type:  fuse.DT_Dir,
		})
	}
	return dirs, nil
}

// MMChannel models a mattermost channel as a FUSE directory
type MMChannel struct {
	id         string
	name       string
	inode      uint64
	client     *MMClient
	unreadFile *UnreadFile
	inFile     *InFile
}

// Attr fills attr with the standard metadata for the node.
// satisfies the fs.Node interface
func (mmc *MMChannel) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = mmc.inode
	a.Mode = os.ModeDir | 0o555 // TODO restrict permissions
	return nil
}

// Lookup returns a fs.Node that corresponds to the given entry inside this
// directory (channel). Currently only an unread file is supported.
// Satisfies the fs.NodeStringLookuper interface
func (mmc *MMChannel) Lookup(_ context.Context, name string) (fs.Node, error) {
	switch name {
	case "unread":
		return mmc.unreadFile, nil
	case "in":
		return mmc.inFile, nil
	}
	return mmc.unreadFile, nil
}

// ReadDirAll returns all directory entries inside the directory corresponding
// corresponding to this channel. Currently only an unread file is supported
// Satisfies the fs.HandleReadAller interface
func (mmc *MMChannel) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{
		{
			Inode: mmc.unreadFile.inode,
			Name:  "unread",
			Type:  fuse.DT_File,
		},
		{
			Inode: mmc.inFile.inode,
			Name:  "in",
			Type:  fuse.DT_File,
		},
	}, nil
}

// UnreadFile models a file containing unread messages in a channel
type UnreadFile struct {
	mmclient  *MMClient
	channelID string
	inode     uint64
}

// Attr fills attr with the standard metadata for the UnreadFile.
// satisfies the fs.Node interface
func (uf *UnreadFile) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = uf.inode
	a.Mode = 0o777
	return nil
}

// ReadAll returns the unread text of the corresponding channel
// satisfies the fs.HandleReadAller interface
func (uf *UnreadFile) ReadAll(_ context.Context) ([]byte, error) {
	// TODO use ctx to timeout calls
	postList, err := uf.mmclient.GetChannelUnread(uf.channelID)
	if err != nil {
		return nil, err
	}
	text, err := uf.mmclient.FormatPostsForDisplay(postList)
	if err != nil {
		return nil, err
	}
	err = uf.mmclient.MarkChannelAsRead(uf.channelID)
	if err != nil {
		return nil, err
	}

	return []byte(text), nil
}

// Open opens the UnreadFile
// Reads for a file are done upto the size reported by Attr. We can't know
// the unread file content (and therefore size) until we query the
// mattermost server in the ReadAll method. We set the file in DirectIO mode to
// get around this.
func (uf *UnreadFile) Open(_ context.Context, _ *fuse.OpenRequest, resp *fuse.OpenResponse) (
	fs.Handle, error) {
	resp.Flags = fuse.OpenDirectIO | fuse.OpenNonSeekable
	return uf, nil
}

// InFile models a file used to write data to a channel
type InFile struct {
	mmclient  *MMClient
	channelID string
	inode     uint64
}

// Attr fills attr with the standard metadata for the InFile.
// satisfies the fs.Node interface
func (ifl *InFile) Attr(_ context.Context, a *fuse.Attr) error {
	a.Inode = ifl.inode
	a.Mode = 0o333
	return nil
}

// Write posts a message to the channel
// satisfies the fs.HandleWriter interface
func (ifl *InFile) Write(
	_ context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse,
) error {
	err := ifl.mmclient.CreatePost(ifl.channelID, req.Data)
	if err != nil {
		fmt.Println(err)
		return err
	}
	resp.Size = len(req.Data)
	return nil
}

// Open opens the InFile
// Reads for a file are done upto the size reported by Attr. We can't know
// the unread file content (and therefore size) until we query the
// mattermost server in the ReadAll method. We set the file in DirectIO mode to
// get around this.
func (ifl *InFile) Open(
	_ context.Context, _ *fuse.OpenRequest, resp *fuse.OpenResponse,
) (fs.Handle, error) {

	resp.Flags = fuse.OpenDirectIO | fuse.OpenNonSeekable
	return ifl, nil
}
