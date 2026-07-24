# MeDLey

A high-performance, interactive Go pipeline for downloading, re-encoding, and metadata-tagging
audio streams.

MeDLey automates the tedious steps of fetching media, converting formats, embedding artwork, and
injecting tags into your music library — all rendered inside a live-updating terminal interface.

***

### Generate audio files

Download a sample of
[*music.yaml*](https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml). Edit it and
optionally rename it as needed; and run the *MeDLey* Docker image to generate the audio files.

<details open>

<summary><i>Linux</i>, <i>Darwin</i></summary>

```
$ rm -rf /var/tmp/media/
$ mkdir /var/tmp/media/
$ curl -sSLkf -m 300 -w "%{http_code}" --output-dir /var/tmp/media -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
$ docker run --rm -it -v /var/tmp:/app mmkdcr/medley:latest yt -f media/music.yaml
```

</details>

<details>

<summary><i>Windows</i></summary>

```
> Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "C:\var\tmp\media"
> New-Item -ItemType Directory -Path "C:\var\tmp\media"
> curl.exe -sSLkf -m 300 -w "%{http_code}" --output-dir "C:\var\tmp\media" -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
> docker run --rm -it -v C:\var\tmp:/app mmkdcr/medley:latest yt -f media/music.yaml
```

</details>

### Play audio files

Now you can explore and play the generated audio files using your favorite audio player.

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

