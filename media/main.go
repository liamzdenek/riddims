package main

import (
    "net/http"
    "fmt"
    "io/ioutil"
    "os"
    "strings"

    "github.com/bitly/go-simplejson"
)

type Track struct {
    Name string
    Id string
    File string
}

type Album struct {
    Name string
    Artist string
    Id string
    Cover string
    Tracks map[string]*Track;
}

var Albums map[string]*Album;
var conf *simplejson.Json;

func main() {
    Albums = make(map[string]*Album)
    load_config();

    http.HandleFunc("/list", handle_list);
    http.HandleFunc("/track/", handle_track);
    http.HandleFunc("/cover/", handle_cover);
    http.ListenAndServe(":54321", nil);
}

func handle_cover( w http.ResponseWriter, r *http.Request ) {
    urlparts := strings.SplitN(r.URL.Path, "/", 4);

    if len(urlparts) != 3 {
        w.WriteHeader(400);
        fmt.Fprintf(w, "ERROR: Malformed cover request\n")
        return;
    }

    album_id := urlparts[2];

    album, ok := Albums[album_id];
    if !ok {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Album not found\n");
        return;
    }

    f, err := ioutil.ReadFile(album.Cover);
    if err != nil {
        w.WriteHeader(403);
        fmt.Fprintf(w, "ERROR: Internal error retrieving track: %s\n", err);
        return;
    }
    w.Write(f);
}

func handle_list( w http.ResponseWriter, r *http.Request ) {
    fmt.Printf("Someone fetched the list\n");
    for album_id,album := range Albums {
        fmt.Fprintf(w, "%s\t%s\t%s\t%s/cover/%s\n", album.Name, album.Artist, album_id, r.Host, album_id);
        for track_id,track := range album.Tracks {
            fmt.Fprintf(w, "\t%s\t%s\t%s/track/%s/%s\n", track.Name, track_id, r.Host, album_id, track_id);
        }
    }
}

func handle_track( w http.ResponseWriter, r *http.Request ) {
    urlparts := strings.SplitN(r.URL.Path, "/", 4);

    if len(urlparts) != 4 {
        w.WriteHeader(400);
        fmt.Fprintf(w, "ERROR: Malformed track request\n")
        return;
    }

    album_id := urlparts[2];
    track_id := urlparts[3];

    album, ok := Albums[album_id];
    if !ok {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Album not found\n");
        return;
    }

    track, ok := album.Tracks[track_id];
    if !ok {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Track not found\n");
        return;
    }

    f, err := ioutil.ReadFile(track.File);
    if err != nil {
        w.WriteHeader(403);
        fmt.Fprintf(w, "ERROR: Internal error retrieving track: %s\n", err);
        return;
    }

    w.Write( f);
}

func load_config() {
    f, err := ioutil.ReadFile("config.json");
    if err != nil {
        fmt.Printf("Couldn't read config file: %s\n", err);
        os.Exit(1);
    }

    conf, err = simplejson.NewJson(f);
    if err != nil {
        fmt.Printf("Couldn't parse config file: %s\n", err);
        os.Exit(1);
    }

    albums, err := conf.Get("albums").Array()
    if err != nil {
        fmt.Printf("Couldn't read albums from config file: %s\n", err);
        os.Exit(1);
    }

    for album_number,album_interface := range albums {
        album_data, ok := album_interface.(map[string]interface{});
        if !ok {
            fmt.Printf("Error parsing album #%d - %s\n(data: %v)\n", album_number, err, album_interface);
            continue;
        }

        album_id, ok := album_data["id"].(string);
        if !ok {
            fmt.Printf("Error parsing album #%d's ID\n(data: %v)\n", album_number, album_data);
            continue;
        }

        album_name, ok := album_data["name"].(string);
        if !ok {
            fmt.Printf("Error parsing album #%d's name\n(data: %v)\n", album_number, album_data);
            continue;
        }

        album_cover, ok := album_data["cover"].(string);
        if !ok {
            fmt.Printf("Error parsing album #%d's cover\n(data: %v)\n", album_number, album_data);
            continue;
        }

        album_artist, ok := album_data["artist"].(string);
        if !ok {
            fmt.Printf("Error parsing album #%d's artist\n(data: %v)\n", album_number, album_data);
            continue;
        }

        album_tracks_interface, ok := album_data["tracks"].([]interface{});
        if !ok {
            fmt.Printf("Error parsing album #%d's tracks\n(data: %v)\n", album_number, album_data);
            continue;
        }

        album_tracks := make(map[string]*Track);

        for track_number,track_interface := range album_tracks_interface {
            track_data, ok := track_interface.([]interface{})
            if !ok {
                fmt.Printf("Error parsing album #%d's track #%d\n(data: %v)\n", album_number, track_number, track_interface)
                continue;
            }

            track := &Track {
                Name: track_data[0].(string),
                File: track_data[2].(string),
                Id: track_data[1].(string),
            }

            album_tracks[track.Id] = track;
        }

        album := &Album{
            Name: album_name,
            Id: album_id,
            Cover: album_cover,
            Tracks: album_tracks,
            Artist: album_artist,
        }
        fmt.Printf("Loaded Album: %s\n", album.Name)
        Albums[album.Id] = album;
    }
}
