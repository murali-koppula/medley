# MeDLey

A high-performance, interactive Go pipeline for downloading, re-encoding, and metadata-tagging
media streams.

MeDLey automates the tedious steps of fetching media, converting formats, embedding artwork, and
injecting tags into your music library — all rendered inside a live-updating terminal interface.

***

The tool can be used to generate media files using one of these methods:

* *MeDLey* docker image from *Docker Hub*.
* Static binaries of *MeDLey* (🐧 `Linux`, 🍎 `Darwin`).

### Generate media files using Docker image

> * Before you run *MeDLey* docker image, install necessary docker packages on your platform and
    make sure you can run the *docker* client on your platform.
> * Optionally, login to the media site in a browser and export the cookies in
    *Netscape HTTP Cookie File* format to a file specified below. Without the auth cookies some
    media may be restricted.

Download a sample of
[*music.yaml*](https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml). Edit it and
optionally rename it as needed; and run the *MeDLey* Docker image to generate the media files.

<details open>

<summary><i>Linux</i>, <i>Darwin</i></summary>

```
$ sudo rm -rf /var/tmp/medley
$ mkdir -p /var/tmp/medley/.cache/medley
$ curl -sSLkf -m 300 -w "%{http_code}" --output-dir /var/tmp/medley -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
$ sudo chown -R nobody /var/tmp/medley
# Optionally, copy any exported cookies to /var/tmp/medley/.config/medley/cookies.txt
$ docker run --rm -it -v /etc/localtime:/etc/localtime:ro -v /var/tmp/medley:/app mmkdcr/medley:latest yt -f music.yaml -V
```

</details>

### Generate media files using pre-compiled static binary

* **Download the MeDLey Binary**

<details open>

<summary><i>Linux</i></summary>

```
# Make sure ~/.local/bin/ exists and is in PATH.
$ curl -sSLkf -m 300 -w %{http_code} -O https://github.com/murali-koppula/medley/releases/download/v0.1.0/medley-linux-amd64.tar.gz
$ tar -xzf medley-linux-amd64.tar.gz
$ rm medley-linux-amd64.tar.gz
$ mv medley ~/.local/bin/     # ~/.local/bin/ must exist, and be in PATH
```

</details>

<details>

<summary><i>Darwin</i></summary>

```
# Make sure ~/.local/bin/ exists and is in PATH.
% curl -sSLkf -m 300 -w "%{http_code}" -O https://github.com/murali-koppula/medley/releases/download/v0.1.0/medley-darwin-arm64.tar.gz
% tar -xzf medley-darwin-arm64.tar.gz
% rm medley-darwin-arm64.tar.gz
% mv medley ~/.local/bin/
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
# Optionally, copy any exported cookies to ~/.config/medley/cookies.txt
$ medley yt -f music.yaml
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

