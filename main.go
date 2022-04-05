package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"
)

var (
	rulesPath  = flag.String("rule", "rule.json", "Rule file")
	outputPath = flag.String("output", "", "PAC file output path")
	serve      = flag.String("serve", "", "listen on a port and serve pac as wpad.dat")
	pacCache   = cache.New(24*time.Hour, 24*time.Hour)
)

func handler(w http.ResponseWriter, r *http.Request) {
	if p, found := pacCache.Get("pac"); found {
		b := p.([]byte)
		bytes.NewBuffer(b).WriteTo(w)
		return
	}
	b := new(bytes.Buffer)
	err := convert(*rulesPath, b)
	if err != nil {
		errtext := fmt.Sprintf("%v", err)
		w.Write([]byte(errtext))
	}
	pacCache.Set("pac", b.Bytes(), cache.DefaultExpiration)
	bytes.NewBuffer(b.Bytes()).WriteTo(w)
}

func main() {
	flag.Parse()

	if *outputPath != "" {
		f, err := os.Create(*outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
		err = convert(*rulesPath, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
		f.Close()
	}

	if *serve != "" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)
		go func() {
			for {
				<-c
				fmt.Println("Refresh triggered.")
				pacCache.Delete("pac")
			}
		}()

		fmt.Println("Serve on http://" + *serve + "/wpad.dat")
		http.HandleFunc("/wpad.dat", handler)
		http.ListenAndServe(*serve, nil)
	}
}
