# CFSpy
A simple Codeforces utility bot for Discord

#### Who is this for?
If you have a Discord server where you talk about [Codeforces](https://codeforces.com), you could use this bot.

## Features
Embed link previews for Codeforces URLs on Discord are usually unhelpful because Codeforces does not have the meta tags that Discord looks for.
To improve that, CFSpy can
- Watch for blog links and show some basic information about the blog.
- Watch for comment links and show the comment.
- Watch for problem links and show some basic information about the problem.

To answer the common question _"Is Codeforces down?"_, there is a command to ping codeforces.com.

## Screenshot
![screenshot](https://i.imgur.com/WAcdoPP.png)

## Use it

#### Invite the bot to your server
[Click here](https://discord.com/api/oauth2/authorize?client_id=713443232834650152&permissions=8192&scope=bot) to authorize the bot. CFSpy requires the "Manage messages" permission to remove the default embeds.

#### Or run your own instance
1. To set up a Discord bot, follow [these steps](https://discordpy.readthedocs.io/en/latest/discord.html).
2. With [Go](https://golang.org/) installed, run
```
$ go get -u github.com/meooow25/cfspy
$ TOKEN=<your_bot_token> cfspy
```

## Thanks
[aryanc403](https://github.com/aryanc403) for the original idea :bulb:
