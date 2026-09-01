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

Optionally, login to the media site in a browser and export the cookies in
*Netscape HTTP Cookie File* format to a file specified below. Without the auth cookies some media
may be restricted.

<details open>
<summary><i>Linux</i>, <i>Darwin</i></summary>

```
sudo rm -rf /var/tmp/medley
mkdir -p /var/tmp/medley/.cache/medley
curl -sSLkf -m 300 -w "%{http_code}" --output-dir /var/tmp/medley -O https://github.com/murali-koppula/medley/raw/refs/heads/main/music.yaml
sudo chown -R nobody /var/tmp/medley
# Optionally, copy any exported cookies to /var/tmp/medley/.config/medley/cookies.txt
docker run --rm -it -v /etc/localtime:/etc/localtime:ro -v /var/tmp/medley:/app mmkdcr/medley:latest yt -f music.yaml -V
```
</details>

### Play audio files

Now you can explore and play the generated audio files using your favorite audio player.

<details open>
<summary><i>Linux</i>, <i>Darwin</i></summary>

```
vlc /var/tmp/media/m4a/western-film-musical/theramin/ecstasy-of-gold.m4a
```
</details>

