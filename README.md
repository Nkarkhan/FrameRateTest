# FrameRateTest
Test FrameRate across a wifi tcp/udp connection

Install golang, install ffmpeg.

serve_files.go can be run in vscode (there is .vscode/launch.json) or can be buildt by go build serve_files.go

Paths for ffmeg download and images may be hardcoded.

In general this code reads the image files numbered sequentially. And then in a loop pipes them to ffmpeg which is instructed to send the resulting mpegts stream to a udp address
if you had ffplay running locally listening on that udp stream you'd see the video.

