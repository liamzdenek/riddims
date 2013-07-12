package main;

import (
    "net/http"
    "io/ioutil"
    "sync"
    "fmt"
    "os"
    "time"
    "strings"
    "errors"

    "github.com/bitly/go-simplejson"
);

type Server struct {
    Address string
    Albums map[string]*Album
    Mutex sync.Mutex
    Error error
}

func NewServer(addr string) *Server {
    iter := time.Duration(conf.Get("refresh_iteration_seconds").MustInt() * 1000000000)

    if iter == 0 {
        panic("No refresh iteration set");
    }

    s := &Server{
        Address: addr,
        Mutex: sync.Mutex{},
        Albums: make(map[string]*Album),
    }

    fmt.Printf("Starting monitor\n");
    go s.Monitor(iter);

    return s
}

func (s *Server) Monitor(iter time.Duration) {
    s.Mutex.Lock();
    list := fmt.Sprintf("http://%s/list", s.Address);
    s.Mutex.Unlock();

    for {
        s.Mutex.Lock();
        resp, err := http.Get(list)
        if err == nil {
            data, err := ioutil.ReadAll(resp.Body)
            resp.Body.Close()
            if err == nil {
                err := s.ParseRequest(string(data));
                if err != nil {
                    s.Error = err;
                } else {
                    s.Error = nil;
                }
            } else {
                s.Error = err;
            }
        } else {
            s.Error = err;
        }
        s.Mutex.Unlock();
        time.Sleep(iter);
    }
}

func (s *Server) ParseRequest(data string) error {
    var cur_album *Album;

    lines := strings.SplitN(data, "\n", -1);

    fmt.Printf("parsing request - lines: %v\n", data);
    for line_number,line := range lines {
        line_parts := strings.SplitN(line, "\t", -1);
        if len(line_parts[0]) != 0 {
            // album definition
            album_name  := line_parts[0]
            artist_name := line_parts[1]
            album_id    := line_parts[2]
            cover_image := line_parts[3]

            fmt.Printf("Got album: %s\n", album_name);
            cur_album = &Album{
                Id: album_id,
                Name: album_name,
                Artist: artist_name,
                Cover: cover_image,
                Tracks: make(map[string]*Track),
                Server: s,
            };

            s.Albums[album_id] = cur_album
        } else if len(line_parts) == 4 {
            // track definition
            if cur_album == nil {
                return errors.New(fmt.Sprintf("Track definition before any album definition on line #%d",line_number));
            }

            track_name := line_parts[1]
            track_id   := line_parts[2]
            track_url  := line_parts[3]

            cur_album.Tracks[track_id] = &Track{
                Id: track_id,
                Name: track_name,
                Url: track_url,
                Album: cur_album,
            }
        }
    }
    return nil;
}

type Album struct {
    Id string
    Name string
    Artist string
    Cover string
    Tracks map[string]*Track
    Server *Server
}

type Track struct {
    Id string
    Name string
    Url string
    Album *Album
}

var conf *simplejson.Json
var Servers []*Server;

func main() {
    Init();

    http.HandleFunc("/", handle_main);
    http.ListenAndServe(":54320", nil);
}

func handle_main( w http.ResponseWriter, r *http.Request ) {
    w.Header().Add("Content-Type", "text/html");
    for _,server := range Servers {
        fmt.Fprintf(w, "<h3>%s</h3>", server.Address);
        for _,album := range server.Albums {
            fmt.Fprintf(w, "<h4>\"%s\" by %s</h4><img src=\"//%s\" style=\"max-width: 250px;\"><br/>", album.Name, album.Artist, album.Cover);
            track_number := 0;
            for _,track := range album.Tracks {
                track_number++;
                fmt.Fprintf(w, "%d) <a href=\"//%s\">%s</a><br/>", track_number, track.Url, track.Name)
            }
        }
    }
}

func Init() {
    f, err := ioutil.ReadFile("config.json")
    if err != nil {
        fmt.Printf("Couldn't read config file: %s\n", err)
        os.Exit(1);
    }

    conf, err = simplejson.NewJson(f)
    if err != nil {
        fmt.Printf("Couldn't parse config file: %s\n", err);
        os.Exit(1);
    }

    servers, err := conf.Get("servers").Array()
    if err != nil {
        fmt.Printf("Couldn't read servers from config file: %s\n", err);
        os.Exit(1);
    }
    
    for _,addr := range servers {
        Servers = append(Servers, NewServer(addr.(string)))
    }
}
