# skiptro

Ever find yourself watching your non-pirated media and wishing it has the Netflix "skip intro" button?
Well, this is as close it gets.

`skiptro` analyzes your media library and tries to find common sequence in the first 5 minutes of the files.
Then, it creates an [EDL file](https://en.wikipedia.org/wiki/Edit_decision_list) (and `.m3u` playlist) which most players
know how to interpret to skip to a particular time in the media file.

## Usage

To create skip file for the "target" file by comparing it to the "source" file:

`skiptro -source TheOffice/S01/E02.mp4 -target TheOffice/S01/E03.mp4`

If using VLC, use the `.m3u` file produced to auto-skip the intro. Kodi will automatically detect the EDL file produced
and skip the intro accordingly.

## Flags

Flag | Description | Default
---|---|---
duration | Length of search time | 10m
hashtype | Hash algorithm to be used (perception, difference, average) | difference
fps | Samples per second to extract for comparison | 3
tolerance | Similarity between frames (lower = more similar) | 15
workers | Workers to spin for JPEG processing | NumCPU (ex. 8)
skipfile | Create EDL and M3U skip files next to the target | true
debug | Profile and output debugging artifacts | false

## Status

Not suitable for mass library usage yet. Can be used to manually create the skip files.

## Features missing

- Full directory processing
- Daemon mode which scans for new files and processes them automatically
