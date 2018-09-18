# README

Thanks to [fonsleenaars](https://github.com/fonsleenaars/) for information about Usher API and overall [info about HLS and VODs](https://github.com/fonsleenaars/twitch-hls-vods)

## Info

``ttvldr`` is a CLI tool that uses Twitch API to download VODs from Twitch and ``ffmpeg`` to process downloaded chunks in a single file. So, that's why you should [download FFMpeg](https://www.ffmpeg.org/download.html) explicitly.

**I don't use any compression** because this operation is very expensive in case of time and CPU usage even with ``-crf 25`` option or ``ultrafast`` preset (or even both of them). Even with ``libx265`` codec.

So if you want to compress file — you should do it by hand after downloading the VOD.

Source codes does not import any third party packages so you don't need to do an extra ``go get`` if you want to make changes in code.

## Warning

Since that official Twitch API does not support most of API versions that used in ``ttvldr`` — they can be closed in any moment. Keep it in mind when you will download any VODs from [Twitch.tv](https://twitch.tv).

If you're continuing getting errors while _connecting to the server_ — try to run ``ttvldr`` with ``-debug`` option — it may make more clear for you if something went wrong with Twitch API or it's just errors in ``ttvldr`` itself.

## Download

You can find the latest release [here](https://github.com/zerospiel/ttvldr/releases). There are versions for:

- [x] Windows (__binary tested__)
- [x] GNU/Linux
- [x] macOS

If you are experiencing any problems with a binary for any listed OS — please, [contact me](mailto:ww@bk.ru).

## Usage

Basic usage of ``ttvldr`` is next:

```raw
ttvldr <option> <link to VOD>
```

Examples of usage the tool:

```raw
ttvldr https://www.twitch.tv/videos/123456789 — download a full VOD
ttvldr -start 1h2m3s -end 1h5m33s twitch.tv/videos/123456789 — download a part a of given VOD
```

All options you can find under with ``ttvldr -help`` command.

If you are experienced user — **you can make a CPU or MEM profiles**. I don't know why but I given this opportunity:

```raw
ttvldr -cpuprofile cpu.out -memprofile mem.out twitch.tv/videos/123456789
```
