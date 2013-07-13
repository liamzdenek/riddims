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
    Id string
    Cover string
    Tracks []*Track
}

type Artist struct {
    Name string
    Id string
    Albums []*Album
}

var Artists []*Artist;
var conf *simplejson.Json;

func main() {
    load_config();

    http.HandleFunc("/list", handle_list);
    http.HandleFunc("/track/", handle_track);
    http.HandleFunc("/cover/", handle_cover);
    http.ListenAndServe(":54321", nil);
}

func handle_list( w http.ResponseWriter, r *http.Request ) {
    fmt.Printf("fetched the list - %s\n", r.RemoteAddr);
    for _,artist := range Artists {
        fmt.Fprintf(w, "%s\t%s\n", artist.Name, artist.Id);
        for _,album := range artist.Albums {
            fmt.Fprintf(w, "\t%s\t%s\t%s/cover/%s/%s\n", album.Name, album.Id, r.Host, artist.Id, album.Id);
            for _,track := range album.Tracks {
                fmt.Fprintf(w, "\t\t%s\t%s\t%s/track/%s/%s/%s\n", track.Name, track.Id, r.Host, artist.Id, album.Id, track.Id);
            }
        }
    }
}

func find_artist(artist_id string) *Artist {
    var artist *Artist;
    for _,tartist := range Artists {
        if tartist.Id == artist_id {
            artist = tartist;
            break;
        }
    }
    return artist
}

func find_album_with_artist(album_id string, artist *Artist) *Album {
    var album *Album;
    for _,talbum := range artist.Albums {
        if talbum.Id == album_id {
            album = talbum;
            break;
        }
    }
    return album
}

func find_track_with_album(track_id string, album *Album) *Track {
    var track *Track;
    for _,ttrack := range album.Tracks {
        if ttrack.Id == track_id {
            track = ttrack;
            break;
        }
    }
    return track;
}

func handle_cover( w http.ResponseWriter, r *http.Request ) {
    urlparts := strings.SplitN(r.URL.Path, "/", 4);

    if len(urlparts) != 4 {
        w.WriteHeader(400);
        fmt.Fprintf(w, "ERROR: Malformed cover request\n")
        return;
    }

    artist_id := urlparts[2];
    album_id := urlparts[3];

    var artist *Artist;
    var album *Album;

    artist = find_artist(artist_id);
    if artist == nil {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Artist not found\n");
        return;
    }

    album = find_album_with_artist(album_id, artist);
    if album == nil {
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

func handle_track( w http.ResponseWriter, r *http.Request ) {
    urlparts := strings.SplitN(r.URL.Path, "/", 5);

    if len(urlparts) != 5 {
        w.WriteHeader(400);
        fmt.Fprintf(w, "ERROR: Malformed track request\n")
        return;
    }

    artist_id := urlparts[2];
    album_id  := urlparts[3];
    track_id  := urlparts[4];

    artist := find_artist(artist_id);
    if artist == nil {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Artist not found\n");
        return;
    }

    album := find_album_with_artist(album_id, artist);
    if album == nil {
        w.WriteHeader(404);
        fmt.Fprintf(w, "ERROR: Album not found\n");
        return;
    }

    track := find_track_with_album(track_id, album);
    if track == nil {
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

    w.Header().Add("Content-Type", "audio/mpeg");
    w.Write(f);
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

    artists, err := conf.Get("artists").Array()
    if err != nil {
        fmt.Printf("Couldn't read albums from config file: %s\n", err);
        os.Exit(1);
    }

    for artist_number, artist_interface := range artists {
        artist_data, ok := artist_interface.(map[string]interface{});
        if !ok {
            fmt.Printf("Error parsing artist #%d's data\n(data: %v)\n", artist_number, artist_data);
            continue;
        }
        artist_name, ok := artist_data["name"].(string);
        if !ok {
            fmt.Printf("Error parsing artist #%d's name\n(data: %v)\n", artist_number, artist_data);
            continue;
        }

        artist_id, ok := artist_data["id"].(string);
        if !ok {
            fmt.Printf("Error parsing artist #%d's id\n(data: %v)\n", artist_number, artist_data);
        }

        album_data, ok := artist_data["albums"].([]interface{})
        if !ok {
            fmt.Printf("Error parsing artist #%d's album list\n(data: %v)\n", artist_number, artist_data);
        }

        artist_albums := make([]*Album, 0);

        for album_number,album_interface := range album_data {
            album_data, ok := album_interface.(map[string]interface{});
            if !ok {
                fmt.Printf("Error parsing album #%d's data\n(data: %v)\n", album_number, album_interface);
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

            album_tracks_interface, ok := album_data["tracks"].([]interface{});
            if !ok {
                fmt.Printf("Error parsing album #%d's tracks\n(data: %v)\n", album_number, album_data);
                continue;
            }

            album_tracks := make([]*Track,0);

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

                album_tracks = append(album_tracks, track);
            }

            album := &Album{
                Name: album_name,
                Id: album_id,
                Cover: album_cover,
                Tracks: album_tracks,
            }
            fmt.Printf("Loaded Album: %s\n", album.Name)
            artist_albums = append(artist_albums, album);
        }

        artist := &Artist{
            Name: artist_name,
            Id: artist_id,
            Albums: artist_albums,
        }

        Artists = append(Artists, artist)
    }
}
