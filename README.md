# CFSpy

[![go-badge](https://img.shields.io/static/v1?label=Built%20with&color=00acd7&style=for-the-badge&message=Go)](https://golang.org/)

A simple Codeforces utility bot for Discord

## ⚠️ Attention!

As of 2025-03-09, this repository is archived and my instance of the bot is offline.

Codeforces began blocking scraping since around 2024-08. This made the bot nearly useless, because most of the features rely on scraping.  
Can the anti-scraping measures be bypassed? Perhaps, but I do not intend to work against the website.  
There exists a Codeforces API, but it does not offer any way to access blog, comment, or submission data and is also notoriously unreliable.  
With no other way forward, it is time to put CFSpy to rest.

---

#### Who is this for?
If you have a Discord server where you discuss [Codeforces](https://codeforces.com), you could use this bot.

## Features
Embed previews for Codeforces links on Discord are usually not helpful, because Codeforces does not have the meta tags that Discord looks for.  
You can let CFSpy watch for these links instead and respond with useful previews. Supported links include
- **Blogs**: Shows the blog information and content.
- **Comments**: Shows the comment information and content.
- **Problems**: Shows some information about the problem.
- **Profiles**: Shows some information about the user profile.
- **Submissions**: Shows some information about the submission.
- **Submissions with line numbers**: Shows a snippet from the submission containing the specified lines. Install this [userscript](https://greasyfork.org/en/scripts/403747-cf-linemaster) to get line selection and highlighting support in your browser.

To make CFSpy ignore links wrap them in <kbd>\<</kbd><kbd>\></kbd>, this is also how Discord's [default embeds](https://support.discord.com/hc/en-us/articles/206342858--How-do-I-disable-auto-embed-) work.

To answer the common question _"Is Codeforces down?"_, there is a command to ping `codeforces.com`.

## Sample
![screenshot](https://i.imgur.com/oBTlBKz.png)

## Use it

#### Invite the bot to your server
[Click here](https://discord.com/api/oauth2/authorize?client_id=713443232834650152&permissions=8192&scope=bot) to authorize the bot. CFSpy requires the <kbd>Manage messages</kbd> permission to remove the default embeds.
<details>
  <summary><i>If you get an error...</i></summary>
  <sub>
    It may be because the bot is already in 100 servers, and Discord does not allow a bot to be in more than 100 servers without <a href="https://support.discord.com/hc/en-us/articles/360040720412-Bot-Verification-and-Data-Whitelisting">verification</a>.
    Discord requires a real-life ID of the developer to verify a bot, which is simply ridiculous. If you agree, consider showing your support on this <a href="https://support.discord.com/hc/en-us/community/posts/360061029252-Remove-ID-verification-for-Bots">article</a>, but to be honest I don't expect them to do anything about it.<br>
    I'm sorry if you aren't able to add the bot because of this, but feel free to run your own instance (see below).
  </sub>
</details>

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
