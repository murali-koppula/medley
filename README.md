# MeDLey

A high-performance, interactive Go pipeline for downloading, re-encoding, and metadata-tagging
media streams.

MeDLey automates the tedious steps of fetching media, converting formats, embedding artwork, and
injecting tags into your music library — all rendered inside a live-updating terminal interface.

***

The tool can be used to generate media files using one of these methods:

* *MeDLey* docker image from *Docker Hub*.
* Static binaries of *MeDLey* (🐧 `Linux`, 🍎 `Darwin`, 🪟 `Windows`).

### Generate media files using Docker image

> Before you run *MeDLey* docker image, install necessary docker packages on your platform and make
  sure you can run the *docker* client on your platform.

Download a sample of
[*music.yaml*](https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml). Edit it and
optionally rename it as needed; and run the *MeDLey* Docker image to generate the media files.

<details open>

<summary><i>Linux</i>, <i>Darwin</i></summary>

```
$ rm -rf /var/tmp/media/
$ mkdir /var/tmp/media/
$ curl -sSLkf -m 300 -w "%{http_code}" --output-dir /var/tmp/media -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
$ docker run --rm -it -v /var/tmp/media:/var/tmp/media mmkdcr/medley:latest -f /var/tmp/media/music.yaml
```

</details>

<details>

<summary><i>Windows</i></summary>

```
> Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "C:\var\tmp\media"
> New-Item -ItemType Directory -Path "C:\var\tmp\media"
> curl.exe -sSLkf -m 300 -w "%{http_code}" --output-dir "C:\var\tmp\media" -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
> docker run --rm -it -v C:\var\tmp\media:/var/tmp/media mmkdcr/medley:latest -f /var/tmp/media/music.yaml
```

</details>

### Generate media files using pre-compiled static binary

* **Download the MeDLey Binary**

<details open>

<summary><i>Linux</i></summary>

```
# Make sure ~/.local/bin/ exists and is in PATH.
$ curl -sSLkf -m 300 -w %{http_code} -O https://github.com/murali-koppula/medley/releases/download/v1.0.0/medley-linux-amd64.tar.gz
$ tar -xzf medley-linux-amd64.tar.gz
$ rm medley-linux-amd64.tar.gz
$ mv medley ~/.local/bin/     # ~/.local/bin/ must exist, and be in PATH
```

</details>

<details>

<summary><i>Darwin</i></summary>

```
# Make sure ~/.local/bin/ exists and is in PATH.
% curl -sSLkf -m 300 -w "%{http_code}" -O https://github.com/murali-koppula/medley/releases/download/v1.0.0/medley-darwin-arm64.tar.gz
% tar -xzf medley-darwin-arm64.tar.gz
% rm medley-darwin-arm64.tar.gz
% mv medley ~/.local/bin/
```

</details>

<details>

<summary><i>Windows</i></summary>

```
# Make sure "$env:USERPROFILE\bin\" exists and is in Path
> curl.exe -sSLkf -m 300 -w "%{http_code}" -O https://github.com/murali-koppula/medley/releases/download/v1.0.0/medley-windows-amd64.zip
> Expand-Archive -Path .\medley-windows-amd64.zip -DestinationPath .\
> del .\medley-windows-amd64.zip
> New-Item -ItemType Directory -Path "$env:USERPROFILE\bin"
> Move-Item .\medley.exe "$env:USERPROFILE\bin\"
```

</details>

* **Generate media files**

> Before you run *MeDLey* binary locally, install necessary *audio*/*video*  packages on your
  platform —
  [*yt-dlp*](https://github.com/yt-dlp/yt-dlp),
  [*ffmpeg*](https://ffmpeg.org),
  [*ImageMagick (convert)*](https://imagemagick.org),
  [*eyeD3*](https://eyed3.readthedocs.io),
  [*atomicparsley*](https://github.com/wez/atomicparsley)

Download a sample of
[*music.yaml*](https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml). Edit it and
optionally rename it as needed; and run the downloaded *MeDLey* binary to generate the media files.

<details open>

<summary><i>Linux</i>, <i>Darwin</i></summary>

```
$ curl -sSLkf -m 300 -w "%{http_code}" -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
$ medley -f music.yaml
```

</details>

<details>

<summary><i>Windows</i></summary>

```

> curl.exe -sSLkf -m 300 -w "%{http_code}" -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
> medley.exe -f music.yaml
```

</details>

### Play media files

Now you can explore and play the generated media files using your favorite media player.

<details open>

<summary><i>Linux</i>, <i>Darwin</i></summary>

```
$ vlc /var/tmp/media/m4a/western-film-musical/theramin/ecstasy-of-gold.m4a
```

</details>

<details>

<summary><i>Windows</i></summary>

```
& "C:\Program Files\VideoLAN\VLC\vlc.exe" "C:\var\tmp\media\ecstasy-of-gold.m4a"
```

</details>

