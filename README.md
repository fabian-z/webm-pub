# webm-pub

A simple webm streaming server written in go. Currently a work in progress.

Multiple clients can GET on a stream endpoint: ```/stream/channelname```.
Only one client per channel/endpoint may POST a correct webm stream for 
distribution.


FFmpeg stream publishing example:

```
ffmpeg -f video4linux2 -s 320x240 -r 16 -i /dev/video0 \
-f alsa -i pulse \ 
-acodec libvorbis -ab 256k \
-vcodec libvpx -vb 512k -f webm \ 
http://localhost:8090/stream/test
```

The resulting stream may be viewed in a modern browser (Firefox, Chrome, Opera) under the same 
URL - http://localhost:8090/stream/test in this example.

iOS does not support webm in the browser. Users may try alternative 
media player apps instead - see 
[PlayerXtreme](https://itunes.apple.com/us/app/media-player-playerxtreme/id456584471?mt=8) 
as an example.
