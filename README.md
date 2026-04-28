Adhoc bot for community content suggestions. Submit ideas with `/suggest`, vote with reactions, and view daily summaries with `/suggestions`.

#### Prerequisites

- Go 1.22 or later.
- Discord bot token from the [Discord Developer Portal](https://discord.com/developers/applications).

#### Configuration

Copy and edit the configuration file:

```console
# cp config.example.yaml config.yaml # Edit config.yaml with your bot details
```

See [config.example.yaml](config.example.yaml) for all options.

#### Building and deployment

<details>
<summary>Container image (recommended)</summary>

Optionally, you can build local container images and deploy through `podman`.

```console
# make container
```

Once successfully built run the images as follows, passing the `config.yaml` configuration
file as a volume mount, replacing `$HASH` with the according git sha (or `:latest`):

```console
# podman run -v ./config.yaml:/app/config.yaml localhost/hottake:$HASH /app/hottake -V 2
```

Container images are automatically [published to GitHub](https://github.com/lcook?tab=packages&repo_name=hottake) on each
commit passing the build pipeline. Like above, run the following:

```console
# podman run -v ./config.yaml:/app/config.yaml ghcr.io/lcook/hottake/hottake:$HASH /app/hottake -V 2
```

</details>

<details>
<summary>Manually building</summary>

Run to build the `hottake` binary:

```console
# make build
```

Now to run the bot:

```console
# ./hottake -c config.yaml -V 2
```

</details>

#### License

[BSD 2-Clause](LICENSE)
