package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr,
		"%s -username USERNAME -password PASSWORD -server SERVER MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	username := flag.String("username", "", "username")
	password := flag.String("password", "", "password")
	server := flag.String("server", "", "server")
	caCert := flag.String("ca-cert", "",
		"optional path to a ca cert, when using a self signed certificate")
	flag.Parse()
	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)
	switch {
	case *username == "":
		log.Fatal("Username can't be empty\n")
	case *password == "":
		log.Fatal("Password can't be empty\n")
	case *server == "":
		log.Fatal("Server can't be empty\n")
	}

	mmfs, err := NewMMFS(*server, *username, *password, *caCert)
	if err != nil {
		log.Fatal(err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("mattermostfs"),
		fuse.Subtype("mattermostfs"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, mmfs)
	if err != nil {
		log.Fatal(err)
	}
}
