# StableDiffusion web UI model updater
Check for updates to your models on Civitai and download new versions.


## Quick Start
Download the latest version from [the release page](https://github.com/jkawamoto/sd-model-updater/releases)
and store the extracted binary to the root directory of your Stable Diffusion web UI.

Run `sd-model-updater` (or `sd-model-updater.exe` for windows users).

It’ll check for updates to each model. If there is a newer version, it’ll ask if you want to download the version:

```
Checking for updates to checkpoint_v10.safetensors... Newer version is found
? Do you want to update v1.0 ➜ v2.0 (y/N)
```

Hit `y` and enter if you want, then it’ll download the version and ask if you want to remove the old one.
```
Downloading v2.0 from https://civitai.com/api/download/models/01234... Done
? Do you want to remove the old version: checkpoint_v10.safetensors (y/N)
```

If there are multiple newer versions are available, you can choose which versions to download.

```
Checking for updates to rola_v1.safetensors... Multiple newer versions are found
? Which versions do you want to download  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [ ]  v2
  [ ]  v3
```

If you don’t select any versions, it’ll skip downloading any versions.


## Check for updates to specific files or directories
If you want to check for updates to specific files or directories, pass the paths to the files or directories to the command.
For example, this command will only check for updates to textual inversions.

```
sd-model-updater embeddings
```


## Known issues
* If you already have multiple versions in your computer, it may offer to update to the version you already have.
* If a version has multiple files such as `.safetensors` and `.ckpt`, it’ll only download the primary file.


## License
This software is released under the MIT License, see [LICENSE](LICENSE).
