# skiptro

Ever find yourself watching your non-pirated media and wishing it has the Netflix "skip intro" button?
Well, this is as close it gets.

`skiptro` analyzes your media library and tries to find common sequence in the first 10 minutes of the files.
Then, it creates an [EDL file](https://en.wikipedia.org/wiki/Edit_decision_list) (or other format) which most players
know how to interpret to skip to a particular time in the media file.

## Usage

`skiptro TheOffice/S01`

## Flags

Table of flags

Flag | Description | Default
---|---|---
duration | Length of search time | 10m
hashtype | Hash algorithm to be used (perception, difference, average) | difference
fps | Samples per second to extract for comparison | 3
tolerance | Similarity between frames (lower = more similar) | 15
workers | Workers to spin for JPEG processing | NumCPU (ex. 8)
debug | Profile and output debugging artifacts | false
edl | Output an EDL file | true

## Status

Not suitable for usage yet.

## Features missing

- Full directory processing
- Other formats than EDL supported (VLC uses m3u for example)
- More precise intro detection
- Daemon mode which scans for new files and processes them automatically
