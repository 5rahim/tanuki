# Tanuki

Tanuki is a Golang library for parsing anime video filenames. 

It is a **fork** of [Anitogo](https://github.com/nssteinbrenner/anitogo), which itself is based off of [Anitomy](https://github.com/erengy/anitomy) and [Anitopy](https://github.com/igorcmoura/anitopy).

## Changes

Tanuki simply handles more cases while avoiding regression.

Changes:
- Added `anime_part`
- Better parsing of `anime_title`
- Updated keywords
- Support for:
  - Absence of title, e.g, `S01E05 - Episode title.mkv`
  - More ranges, e.g, `S1-2`, `Seasons 1-2`, `Seasons 1 ~ 2`, etc...
  - Enclosed keywords, e.g, `Hyouka (2012) [Season 1+OVA] [BD 1080p HEVC OPUS] [Dual-Audio]`

## Example
The following filename...

    [Trix] Shingeki no Kyojin - S04E29-31 (Part 3) [Multi Subs] (1080p AV1 E-AC3)"

...is resolved into:

```json
{
  "file_name": "[Trix] Shingeki no Kyojin - S04E29-31 (Part 3) [Multi Subs] (1080p AV1 E-AC3)",
  "anime_title": "Shingeki no Kyojin",
  "anime_season": ["04"],
  "anime_part": ["3"],
  "episode_number": ["29", "31"],
  "release_group": "Trix",
  "video_resolution": "1080p",
  "video_term": ["AV1"]
}
```

The following example code:

```go
package main

import (
    "fmt"
    "encoding/json"

    "github.com/seanime-app/tanuki"
)

func main() {
    parsed := tanuki.Parse("[Nubles] Space Battleship Yamato 2199 (2012) episode 18 (720p 10 bit AAC)[1F56D642]", tanuki.DefaultOptions)
    jsonParsed, err := json.MarshalIndent(parsed, "", "    ")
    if err != nil {
        fmt.Println(err)
    }
    fmt.Println(string(jsonParsed) + "\n")

    // Accessing the elements directly
    fmt.Println("Anime Title:", parsed.AnimeTitle)
    fmt.Println("Anime Year:", parsed.AnimeYear)
    fmt.Println("Episode Number:", parsed.EpisodeNumber)
    fmt.Println("Release Group:", parsed.ReleaseGroup)
    fmt.Println("File Checksum:", parsed.FileChecksum)
}
```

Will output:

```go
{
    "anime_title": "Space Battleship Yamato 2199",
    "anime_year": "2012",
    "audio_term": ["AAC"],
    "episode_number": ["18"],
    "file_checksum": "1F56D642",
    "file_name": "[Nubles] Space Battleship Yamato 2199 (2012) episode 18 (720p 10 bit AAC)[1F56D642]",
    "release_group": "Nubles",
    "video_resolution": "720p"
}
```

The Parse function returns a pointer to an Elements struct. The full definition of the struct is here:

```go
type elements struct {
    AnimeSeason         []string  `json:"anime_season,omitempty"`
    AnimeSeasonPrefix   []string  `json:"anime_season_prefix,omitempty"`
    AnimePart           []string  `json:"anime_part,omitempty"`
    AnimePartPrefix     []string  `json:"anime_part_prefix,omitempty"`
    AnimeTitle          string    `json:"anime_title,omitempty"`
    AnimeType           []string  `json:"anime_type,omitempty"`
    AnimeYear           string    `json:"anime_year,omitempty"`
    AudioTerm           []string  `json:"audio_term,omitempty"`
    DeviceCompatibility []string  `json:"device_compatibility,omitempty"`
    EpisodeNumber       []string  `json:"episode_number,omitempty"`
    EpisodeNumberAlt    []string  `json:"episode_number_alt,omitempty"`
    EpisodePrefix       []string  `json:"episode_prefix,omitempty"`
    EpisodeTitle        string    `json:"episode_title,omitempty"`
    FileChecksum        string    `json:"file_checksum,omitempty"`
    FileExtension       string    `json:"file_extension,omitempty"`
    FileName            string    `json:"file_name,omitempty"`
    Language            []string  `json:"language,omitempty"`
    Other               []string  `json:"other,omitempty"`
    ReleaseGroup        string    `json:"release_group,omitempty"`
    ReleaseInformation  []string  `json:"release_information,omitempty"`
    ReleaseVersion      []string  `json:"release_version,omitempty"`
    Source              []string  `json:"source,omitempty"`
    Subtitles           []string  `json:"subtitles,omitempty"`
    VideoResolution     string    `json:"video_resolution,omitempty"`
    VideoTerm           []string  `json:"video_term,omitempty"`
    VolumeNumber        []string  `json:"volume_number,omitempty"`
    VolumePrefix        []string  `json:"volume_prefix,omitempty"`
    Unknown             []string  `json:"unknown,omitempty"`
    checkAltNumber      bool
}
```

Sample results encoded in JSON can be seen in the tests/data.json file.

## Installation
Get the package:

    go get -u github.com/seanime-app/tanuki

Then, import it in your code:

    import "github.com/seanime-app/tanuki"

## Options
The Parse function receives the filename and an Options struct. The default options are as follows:

    var DefaultOptions = Options{
        AllowedDelimiters:  " _.&+,|", // Parse these as delimiters
        IgnoredStrings:     []string{}, // Ignore these when they are in the filename
        ParseEpisodeNumber: true, // Parse the episode number and include it in the elements
        ParseEpisodeTitle:  true, // Parse the episode title and include it in the elements
        ParseFileExtension: true, // Parse the file extension and include it in the elements
        ParseReleaseGroup:  true, // Parse the release group and include it in the elements
    }