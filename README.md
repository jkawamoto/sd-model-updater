# StableDiffusion web UI model updater
[![Go application](https://github.com/jkawamoto/sd-model-updater/actions/workflows/test.yaml/badge.svg)](https://github.com/jkawamoto/sd-model-updater/actions/workflows/test.yaml)

Check for updates to your models on Civitai and download new versions.


## Quick Start
Download the latest version from [the release page](https://github.com/jkawamoto/sd-model-updater/releases)
and store the extracted binary to the root directory of your Stable Diffusion web UI.

Run `sd-model-updater` (or `sd-model-updater.exe` for windows users).

It’ll check for updates to each model. If there is a newer version, it’ll ask if you want to download the version:

```
Checkpoint ABC has a newer version
? Do you want to update v1.0 ➜ v2.0 (y/N)
```

Hit `y` and enter if you want, then it’ll download the version.

If there are multiple newer versions are available, you can choose which versions to download.

```
RoLA ABC has multiple newer versions
? Which versions do you want to download (current: v1) [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]
> [ ]  v2
  [ ]  v3
```

If you don’t select any versions, it’ll skip downloading any versions.


### Check for updates to specific files or directories
If you want to check for updates to specific files or directories, pass the paths to the files or directories to the command.
For example, this command will only check for updates to textual inversions.

```
sd-model-updater embeddings
```


### Download pickle files instead of safetensors
By default, this command downloads safetensor files. If you prefer pickle files, give `-format pickle` to the command.
However, if a model version only provides safetensor file, it will be downloaded.


## Command-line options
This is the usage of this command:
```
Usage:
  sd-model-updater [path...]

[path...] is an optional list of paths to the files or directories.
This command checks for updates to the given files or files in the given directories.

If not paths are given, this command considers the current directory is the root of web UI,
and checks updates to the files in the default model directories,
such as models/Stable-diffusion, models/Lora, etc.

Flags:
  -format value       prefered file format: safetensor or pickle (default safetensor)
```

## License
This software is released under the MIT License, see [LICENSE](LICENSE).
