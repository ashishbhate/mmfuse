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

1. The generated directory structure:

   ```
   |- root
      |- team1
      |  |- channel1
      |  |  |- in
      |  |  |- unread
      |  |
      |  |
      |  |- DM1
      |     |- in
      |     |- unread
      |
      |- team1
         |- channel1
         |  |- in
         |  |- unread
         |
         |- DM1
            |- in
            |- unread
   ```

2. The "unread" file contains unread messages from the corresponding channel.
   The message display format is:
   ```
   DateTime <ThreadId> Username: Message
   ```

3. The "in" file can be written to to post messages to the channel.

3. The code does not sync state. For example, if you're added to a new team or join a new channel,
   you'll need to remount the filesystem to see these changes. Support for live syncing of state is planned.

4. The code is in alpha. There will be breaking changes.
