# CFSpy
A simple Codeforces utility bot for Discord

## Features
Embed link previews for Codeforces URLs on Discord are usually unhelpful because Codeforces does not have the meta tags that Discord looks for.
To improve that, CFSpy can
- Watch for blog links and show some basic information about the blog.
- Watch for comment links and show the comment.

To answer the common question _"Is Codeforces down?"_, there is a command to ping codeforces.com.

## Run it
1. To set up a Discord bot, follow [these steps](https://discordpy.readthedocs.io/en/latest/discord.html). CFSpy requires the "Manage messages" permission to remove the default link embeds.  
2. With [Go](https://golang.org/) installed, run
```
$ go get -u github.com/meooow25/cfspy
$ TOKEN=<your_bot_token> cfspy
```

## Thanks
[aryanc403](https://github.com/aryanc403) for the original idea :bulb:
