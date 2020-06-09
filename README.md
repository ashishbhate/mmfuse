# mmfuse

A simple FUSE file system for mattermost



### Build

```bash
go build .
```



### Run

```bash
./mmfuse -username USERNAME -password PASSWORD -server SERVER MOUNTPOINT
```

Unmount (you may need super user permissions):

```bash
umount MOUNTPOINT
```



### Notes:

The generated directory structure:

```
|- root
   |- team1
   |  |- channel1
   |  |  |- unread
   |  |
   |  |
   |  |- DM1
   |     |- unread
   |
   |- team1
      |- channel1
      |  |- unread
      |
      |- DM1
         |- unread
```


Currently only an "unread" file containing unread messages from the corresponding channel is implemented.

An "in" file to send messages and an "out" file containing messages since mounting are planned.

The code does not sync state. For example, if you're added to a new team or join a new channel,
 you'll need to remount the filesystem to see these changes. Support for live syncing of state is planned.

The code is in alpha. There will be breaking changes.