# CFSpy

[![go-badge](https://img.shields.io/static/v1?label=Built%20with&color=00acd7&style=for-the-badge&message=Go)](https://golang.org/)&ensp;[![server-count-badge](https://img.shields.io/badge/dynamic/json?label=Servers&logo=discord&logoColor=white&color=7289DA&style=for-the-badge&query=%24.serverCount&url=https%3A%2F%2Fgist.githubusercontent.com%2Fmeooow25%2Fe550658ac19cc0cdd515a414afea23bb%2Fraw%2Fserver-count.json)](https://discord.com/api/oauth2/authorize?client_id=713443232834650152&permissions=8192&scope=bot)

A simple Codeforces utility bot for Discord

#### Who is this for?
If you have a Discord server where you discuss [Codeforces](https://codeforces.com), you could use this bot.

## Features
Embed previews for Codeforces links on Discord are usually not helpful, because Codeforces does not have the meta tags that Discord looks for.  
You can let CFSpy watch for these links instead and respond with useful previews. Supported links include
- **Blogs**: Shows some information about the blog.
- **Comments**: Shows the comment information and content.
- **Problems**: Shows some information about the problem.
- **Profiles**: Shows some information about the user profile.
- **Submissions**: Shows some information about the submission.
- **Submissions with line numbers**: Shows a snippet from the submission containing the specified lines. Install this [userscript](https://greasyfork.org/en/scripts/403747-cf-linemaster) to get line selection and highlighting support in your browser.

To answer the common question _"Is Codeforces down?"_, there is a command to ping `codeforces.com`.

## Sample
![screenshot](https://i.imgur.com/oBTlBKz.png)

## Use it

#### Invite the bot to your server
[Click here](https://discord.com/api/oauth2/authorize?client_id=713443232834650152&permissions=8192&scope=bot) to authorize the bot. CFSpy requires the `Manage messages` permission to remove the default embeds.

#### Or run your own instance
1. To set up a Discord bot, follow [these steps](https://discordpy.readthedocs.io/en/latest/discord.html).
2. With [Go](https://golang.org/) ≥1.14 installed, run
```
$ GO111MODULE=on go get github.com/meooow25/cfspy@latest
$ TOKEN=<your_bot_token> cfspy
```

## Thanks
[aryanc403](https://github.com/aryanc403) for the original idea :bulb:  
[git-the-lines](https://github.com/dolphingarlic/git-the-lines) which was the inspiration for submission snippets
