# CFSpy
A simple Codeforces utility bot for Discord

## Features
Embed link previews for Codeforces URLs on Discord are usually unhelpful because Codeforces does not have appropriate meta tags that Discord uses to get information about the page.
To improve that, CFSpy can
- Watch for blog links and show some basic information about the blog.
- Watch for comment links and show the comment.

There is also a command to ping codeforces.com to answer the common question _"Is Codeforces down?"_.

## Run it
To set up a Discord bot, follow [these steps](https://discordpy.readthedocs.io/en/latest/discord.html). CFSpy requires the "Manage messages" permission to remove the default link embeds.  
With [Go](https://golang.org/) installed, run
```
$ go get -u github.com/meooow25/cfspy
$ TOKEN=<your_bot_token> cfspy
```
